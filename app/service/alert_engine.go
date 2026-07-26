package service

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
)

// alertMutex 一轮评估必须串行：节点采集和本机检查都会触发评估，
// 并发跑会把同一事件的去抖计数加两次，还可能重复发信。
var alertMutex sync.Mutex

// alertTarget 一个被监控对象（本机或某个节点）。
// 本机也包装成 model.Node，就能和节点共用 buildNodeWarnings——
// 阈值只有一处定义，本地和远程的告警口径永远一致。
type alertTarget struct {
	SourceType string
	NodeID     uint
	Name       string
	Node       model.Node
}

// EvaluateAlerts 跑一轮告警评估：算出当前该告警的项，与库里的活动事件比对，
// 做去抖/静默/恢复判定，最后把本轮变化聚合成一封邮件发出去。
//
// 聚合是必需的：10 个节点同时磁盘满，要发 1 封列 10 行，而不是 10 封。
func EvaluateAlerts() {
	alertMutex.Lock()
	defer alertMutex.Unlock()

	notifyRepo := repo.NewNotify()
	cfg, err := notifyRepo.GetConfig()
	if err != nil || !cfg.Enabled {
		return
	}

	targets, err := collectAlertTargets()
	if err != nil {
		global.LOG.Errorf("[Alert] 加载监控对象失败: %v", err)
		return
	}

	active, err := notifyRepo.ActiveEvents()
	if err != nil {
		global.LOG.Errorf("[Alert] 加载活动事件失败: %v", err)
		return
	}
	activeMap := make(map[string]*model.AlertEvent, len(active))
	for i := range active {
		activeMap[eventKey(active[i].SourceType, active[i].NodeID, active[i].Type)] = &active[i]
	}

	now := time.Now()
	var fired, resolved []model.AlertEvent
	seen := make(map[string]struct{})

	for _, target := range targets {
		for _, w := range buildNodeWarnings(target.Node) {
			if !alertTypeEnabled(cfg, w.Type) {
				continue
			}
			key := eventKey(target.SourceType, target.NodeID, w.Type)
			seen[key] = struct{}{}

			event := activeMap[key]
			if event == nil {
				event = &model.AlertEvent{
					SourceType:  target.SourceType,
					NodeID:      target.NodeID,
					SourceName:  target.Name,
					Type:        w.Type,
					Status:      model.AlertStatusPending,
					FirstSeenAt: now,
				}
			}
			event.SourceName = target.Name
			event.Level = w.Level
			event.Value = w.Value
			event.Detail = describeWarning(target, w)
			event.LastSeenAt = now
			event.HitCount++

			switch {
			case event.Status == model.AlertStatusPending && event.HitCount >= debounceTimes(cfg):
				// 达到去抖次数才算真的出问题，避免采集抖动误报
				event.Status = model.AlertStatusFiring
				event.LastNotifiedAt = now
				fired = append(fired, *event)
			case event.Status == model.AlertStatusFiring && shouldRemind(cfg, event.LastNotifiedAt, now):
				// 持续未恢复，隔一个静默期再提醒一次
				event.LastNotifiedAt = now
				fired = append(fired, *event)
			}

			if err := notifyRepo.SaveEvent(event); err != nil {
				global.LOG.Errorf("[Alert] 保存事件失败 %s: %v", key, err)
			}
		}
	}

	// 这一轮没再出现的活动事件 = 已恢复
	for key, event := range activeMap {
		if _, still := seen[key]; still {
			continue
		}
		wasFiring := event.Status == model.AlertStatusFiring
		event.Status = model.AlertStatusResolved
		event.ResolvedAt = now
		event.HitCount = 0
		if err := notifyRepo.SaveEvent(event); err != nil {
			global.LOG.Errorf("[Alert] 保存恢复状态失败 %s: %v", key, err)
			continue
		}
		// 只有通知过的告警才需要通知恢复——没发过告警就发"已恢复"会让人莫名其妙
		if wasFiring && cfg.NotifyResolved {
			resolved = append(resolved, *event)
		}
	}

	if len(fired) == 0 && len(resolved) == 0 {
		return
	}
	sendAlertMail(cfg, fired, resolved)
}

func collectAlertTargets() ([]alertTarget, error) {
	targets := make([]alertTarget, 0, 8)

	// 本机：包装成一个 online 的 Node，直接复用节点那套判定
	localName := constant.AppBrand + " 本机"
	summary := LocalNodeSummary()
	if strings.TrimSpace(summary.Hostname) != "" {
		localName = summary.Hostname + "（本机）"
	}
	targets = append(targets, alertTarget{
		SourceType: model.AlertSourceLocal,
		Name:       localName,
		Node:       model.Node{Name: localName, Status: NodeStatusOnline, Summary: summary},
	})

	nodes, err := repo.NewNode().List()
	if err != nil {
		return targets, err
	}
	for _, node := range nodes {
		targets = append(targets, alertTarget{
			SourceType: model.AlertSourceNode,
			NodeID:     node.ID,
			Name:       node.Name,
			Node:       node,
		})
	}
	return targets, nil
}

func alertTypeEnabled(cfg model.NotifyConfig, warnType string) bool {
	switch warnType {
	case "disk":
		return cfg.EnableDisk
	case "container":
		return cfg.EnableContainer
	case "offline", "unauthorized":
		return cfg.EnableOffline
	case "cert":
		return cfg.EnableCert
	default:
		return false
	}
}

func debounceTimes(cfg model.NotifyConfig) int {
	if cfg.DebounceTimes < 1 {
		return 1
	}
	return cfg.DebounceTimes
}

func shouldRemind(cfg model.NotifyConfig, lastNotified, now time.Time) bool {
	if cfg.SilenceHours <= 0 {
		return false // 0 表示不重复提醒
	}
	if lastNotified.IsZero() {
		return true
	}
	return now.Sub(lastNotified) >= time.Duration(cfg.SilenceHours)*time.Hour
}

func eventKey(sourceType string, nodeID uint, warnType string) string {
	return fmt.Sprintf("%s:%d:%s", sourceType, nodeID, warnType)
}

// describeWarning 生成人能看懂的一句话。邮件里只有数字没有上下文，
// 收件人还得登面板才知道发生了什么。
func describeWarning(target alertTarget, w dto.NodeWarning) string {
	switch w.Type {
	case "disk":
		path := strings.TrimSpace(target.Node.Summary.DiskMaxPath)
		if path == "" {
			path = "磁盘"
		}
		return fmt.Sprintf("%s 使用率 %.1f%%", path, w.Value)
	case "container":
		return fmt.Sprintf("%d 个容器处于异常状态（dead/restarting/paused）", int(w.Value))
	case "offline":
		if w.Value > 0 {
			return fmt.Sprintf("节点已离线，最后在线于 %.1f 小时前", w.Value)
		}
		return "节点离线，且没有历史在线记录"
	case "unauthorized":
		return "节点令牌校验失败，主控已无法采集该节点"
	case "cert":
		days := int(w.Value)
		if days < 0 {
			return fmt.Sprintf("已有证书过期 %d 天", -days)
		}
		return fmt.Sprintf("最近一张证书还有 %d 天到期", days)
	default:
		return w.Type
	}
}

func sendAlertMail(cfg model.NotifyConfig, fired, resolved []model.AlertEvent) {
	subject := buildAlertSubject(fired, resolved)
	body := buildAlertBody(fired, resolved)
	if err := SendNotifyMail(cfg, subject, body); err != nil {
		global.LOG.Errorf("[Alert] 邮件发送失败（本轮 %d 个告警 / %d 个恢复）: %v", len(fired), len(resolved), err)
		return
	}
	global.LOG.Infof("[Alert] 已发送通知邮件：%d 个告警 / %d 个恢复", len(fired), len(resolved))
}

func buildAlertSubject(fired, resolved []model.AlertEvent) string {
	brand := constant.AppBrand
	switch {
	case len(fired) > 0 && len(resolved) > 0:
		return fmt.Sprintf("[%s] %d 项告警 / %d 项恢复", brand, len(fired), len(resolved))
	case len(fired) > 0:
		// 单条告警时把主体写进标题，手机通知栏一眼能看清
		if len(fired) == 1 {
			return fmt.Sprintf("[%s] %s %s", brand, fired[0].SourceName, alertTypeLabel(fired[0].Type))
		}
		return fmt.Sprintf("[%s] %d 项告警", brand, len(fired))
	default:
		if len(resolved) == 1 {
			return fmt.Sprintf("[%s] %s %s 已恢复", brand, resolved[0].SourceName, alertTypeLabel(resolved[0].Type))
		}
		return fmt.Sprintf("[%s] %d 项告警已恢复", brand, len(resolved))
	}
}

func alertTypeLabel(t string) string {
	switch t {
	case "disk":
		return "磁盘告警"
	case "container":
		return "容器异常"
	case "offline":
		return "节点离线"
	case "unauthorized":
		return "节点令牌失效"
	case "cert":
		return "证书即将到期"
	default:
		return t
	}
}

func buildAlertBody(fired, resolved []model.AlertEvent) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s 监控通知  %s\n", constant.AppBrand, time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString(strings.Repeat("-", 46) + "\n\n")

	if len(fired) > 0 {
		b.WriteString(fmt.Sprintf("【告警】%d 项\n", len(fired)))
		for _, e := range sortEvents(fired) {
			b.WriteString(fmt.Sprintf("  · [%s] %s —— %s\n", levelLabel(e.Level), e.SourceName, e.Detail))
			b.WriteString(fmt.Sprintf("    首次出现：%s\n", e.FirstSeenAt.Format("2006-01-02 15:04:05")))
		}
		b.WriteString("\n")
	}

	if len(resolved) > 0 {
		b.WriteString(fmt.Sprintf("【恢复】%d 项\n", len(resolved)))
		for _, e := range sortEvents(resolved) {
			b.WriteString(fmt.Sprintf("  · %s —— %s 已恢复\n", e.SourceName, alertTypeLabel(e.Type)))
			b.WriteString(fmt.Sprintf("    持续时长：%s\n", humanDuration(e.ResolvedAt.Sub(e.FirstSeenAt))))
		}
		b.WriteString("\n")
	}

	b.WriteString(strings.Repeat("-", 46) + "\n")
	b.WriteString("本邮件由 " + constant.AppBrand + " 自动发送，请登录面板查看详情。\n")
	return b.String()
}

// sortEvents 危险级排前面，同级按名字排，保证邮件内容顺序稳定
func sortEvents(events []model.AlertEvent) []model.AlertEvent {
	out := make([]model.AlertEvent, len(events))
	copy(out, events)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Level != out[j].Level {
			return out[i].Level == "danger"
		}
		if out[i].SourceName != out[j].SourceName {
			return out[i].SourceName < out[j].SourceName
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func levelLabel(level string) string {
	if level == "danger" {
		return "严重"
	}
	return "警告"
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return "不到 1 分钟"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1f 小时", d.Hours())
}
