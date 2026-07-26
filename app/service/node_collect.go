package service

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
)

// collectMutex 保证同一时刻只有一轮采集在跑。
// 定时任务和用户手动刷新可能撞上，串行化比让两轮互相覆盖写要省心。
var collectMutex sync.Mutex

// CollectAllNodes 拉取全部节点摘要。
// 3~10 台规模下串行足够：单节点超时 5 秒，最坏情况一轮 50 秒，仍在采集间隔内。
func CollectAllNodes() {
	collectMutex.Lock()
	defer collectMutex.Unlock()

	nodes, err := repo.NewNode().List()
	if err != nil {
		global.LOG.Errorf("[Node] 加载节点列表失败: %v", err)
		return
	}
	for _, node := range nodes {
		// 定时轮采里单台失败不影响其他节点，失败原因已写进该节点的 status_msg
		if err := CollectNode(node.ID); err != nil {
			global.LOG.Warnf("[Node] 采集节点 %s 失败: %v", node.Name, err)
		}
	}
}

// CollectNode 采集单个节点。
// 采集结果无论成败都会落库，同时把失败原因返回给调用方——
// 手动「采集/测试连接」必须能看到失败，不能只写进库里就对用户报成功。
func CollectNode(id uint) error {
	node, err := repo.NewNode().GetByID(id)
	if err != nil {
		global.LOG.Errorf("[Node] 节点 %d 不存在: %v", id, err)
		return err
	}

	summary, err := FetchNodeSummary(node)
	if err != nil {
		status := NodeStatusOffline
		if errors.Is(err, ErrNodeUnauthorized) {
			status = NodeStatusUnauthorized
		}
		// 保留上一次的 summary 和 last_seen_at，让前端能显示“最后在线于 X”
		if updateErr := repo.NewNode().UpdateStatus(node.ID, status, truncateStatusMsg(err.Error())); updateErr != nil {
			global.LOG.Errorf("[Node] 更新节点 %s 状态失败: %v", node.Name, updateErr)
		}
		return err
	}

	if updateErr := repo.NewNode().UpdateSummary(node.ID, model.Node{
		Status:     NodeStatusOnline,
		StatusMsg:  "",
		Version:    summary.Version,
		LastSeenAt: time.Now(),
		Summary:    summary,
	}); updateErr != nil {
		global.LOG.Errorf("[Node] 更新节点 %s 摘要失败: %v", node.Name, updateErr)
		return updateErr
	}
	return nil
}

// truncateStatusMsg status_msg 字段是 varchar(255)，超长的网络错误要截断，否则 sqlite 会写入失败
func truncateStatusMsg(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) > 240 {
		return msg[:240]
	}
	return msg
}

// ---- 以下为被控侧：本机只读令牌的管理 ----

// LocalNodeTokenEnabled 本机是否已开启只读接入
func LocalNodeTokenEnabled() bool {
	setting, err := repo.NewISettingRepo().Get(repo.NewISettingRepo().WithByKey(constant.NodeAccessTokenKey))
	if err != nil {
		return false
	}
	return strings.TrimSpace(setting.Value) != ""
}

// IssueLocalNodeToken 生成并保存本机只读令牌，返回明文。
// 明文只在这一次返回，之后库里只有密文——用户没记下来就只能重新生成。
func IssueLocalNodeToken() (string, error) {
	token := common.RandStr(NodeTokenLength)
	cipherToken, err := encrypt.StringEncrypt(token)
	if err != nil {
		return "", err
	}
	if err := repo.NewISettingRepo().UpdateOrCreate(constant.NodeAccessTokenKey, cipherToken); err != nil {
		return "", err
	}
	global.LOG.Info("[Node] 已重新签发本机只读令牌")
	return token, nil
}

// RevokeLocalNodeToken 关闭本机只读接入。
// 这里必须用 Update 而不是 UpdateOrCreate：后者内部是 Assign(struct) + FirstOrCreate，
// GORM 会跳过结构体里的零值字段，导致清空操作被静默忽略、令牌其实还生效。
func RevokeLocalNodeToken() error {
	if err := repo.NewISettingRepo().Update(constant.NodeAccessTokenKey, ""); err != nil {
		return err
	}
	global.LOG.Info("[Node] 已关闭本机只读接入")
	return nil
}
