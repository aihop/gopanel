package service

import (
	"strings"

	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/encrypt"
)

// 被控侧「控制令牌」的管理。
//
// 与只读令牌（NodeAccessToken）严格分开是刻意的：只读令牌只能取一份状态摘要，
// 控制令牌可以通过主控的代理执行这台机器上的任何管理操作，等价于面板管理员密码。
// 合成一个令牌会让"只读接入"这个承诺变成假的。

// LocalControlTokenEnabled 本机是否已开启控制接入
func LocalControlTokenEnabled() bool {
	token, err := LoadLocalControlToken()
	return err == nil && token != ""
}

// LoadLocalControlToken 读取本机控制令牌明文，未开启时返回空串
func LoadLocalControlToken() (string, error) {
	setting, err := repo.NewISettingRepo().Get(repo.NewISettingRepo().WithByKey(constant.NodeControlTokenKey))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(setting.Value) == "" {
		return "", nil
	}
	return encrypt.StringDecrypt(setting.Value)
}

// IssueLocalControlToken 生成并保存本机控制令牌，返回明文（只这一次返回）
func IssueLocalControlToken() (string, error) {
	token := common.RandStr(NodeTokenLength)
	cipherToken, err := encrypt.StringEncrypt(token)
	if err != nil {
		return "", err
	}
	if err := repo.NewISettingRepo().UpdateOrCreate(constant.NodeControlTokenKey, cipherToken); err != nil {
		return "", err
	}
	global.LOG.Info("[Node] 已重新签发本机控制令牌（该令牌可执行写操作）")
	return token, nil
}

// RevokeLocalControlToken 关闭本机控制接入。
// 必须用 Update 而不是 UpdateOrCreate：后者内部 Assign(struct) 会跳过零值，清空会被静默忽略。
func RevokeLocalControlToken() error {
	if err := repo.NewISettingRepo().Update(constant.NodeControlTokenKey, ""); err != nil {
		return err
	}
	global.LOG.Info("[Node] 已关闭本机控制接入")
	return nil
}
