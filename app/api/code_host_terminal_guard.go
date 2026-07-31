package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
)

var codeHostTerminalLifecycle sync.Mutex

func validateHostTerminalDevelopmentOpen() error {
	var count int64
	if err := global.DB.Model(&model.AIDevSession{}).Where("status = ?", codeSessionStatusDelivering).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("当前有 Code 会话正在统一交付，交付完成前不能打开宿主终端")
	}
	return nil
}

func stopHostTerminalsForCodeDelivery(ctx context.Context, userID uint) error {
	stopContext, cancel := context.WithTimeout(ctx, codeDeliveryQueueTimeout)
	defer cancel()
	return hostTerminals.stopAllForCodeDelivery(stopContext, userID)
}

func (manager *hostTerminalManager) stopAllForCodeDelivery(ctx context.Context, userID uint) error {
	manager.mu.Lock()
	ids := make([]uint, 0, len(manager.sessions))
	for id := range manager.sessions {
		ids = append(ids, id)
	}
	manager.mu.Unlock()
	for _, id := range ids {
		if !manager.stopAndWait(ctx, id) && manager.get(id) != nil {
			return fmt.Errorf("统一交付前无法停止宿主终端会话 %d", id)
		}
		recordHostTerminalAudit(id, userID, "delivery_stop", "success", "", "统一交付前自动停止宿主终端")
	}
	return nil
}
