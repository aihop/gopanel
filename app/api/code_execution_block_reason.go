package api

import (
	"errors"
	"fmt"

	"github.com/aihop/gopanel/app/model"
)

// 交付等不到工作区时，阻塞源可能是三种完全不同的东西。
//
// 以前一律报「会话仍有正在运行的 AI 执行或终端」，用户照着去停自己的会话，
// 停完再试还是失败——因为真正占着的是别的会话，或者干脆是并发槽满了。
// 提示指错方向比没有提示更糟：它让人反复做无效操作。
var (
	errCodeDeliveryWorkspaceBusy = errors.New(
		"本会话仍有正在运行的 AI 执行或终端，交付无法独占工作区；请先停止本会话的执行再重试交付",
	)
	errCodeDeliveryRepositoryBusy = errors.New(
		"目标仓库正被其他会话占用，交付无法独占；请等对方结束后重试",
	)
	errCodeDeliveryCapacityBusy = errors.New(
		"面板同时进行的交付已达上限，请稍后重试；如需放开可调大 GOPANEL_CODE_MAX_DELIVERY_CONCURRENCY",
	)
)

// deliveryBlockReason 在交付等待超时后回答「到底是谁挡住了」。
//
// 判定顺序就是用户能采取的行动的顺序：先看是不是自己会话占着（自己能停），
// 再看是不是别的会话占着（只能等），最后才是并发槽满（要调配置）。
func (coordinator *codeExecutionCoordinator) deliveryBlockReason(session *model.AIDevSession) error {
	if coordinator == nil || session == nil {
		return errCodeDeliveryWorkspaceBusy
	}
	keys := codeExecutionDeliveryKeys(session)
	coordinator.mu.Lock()
	holders := coordinator.conflicts(keys, codeExecutionDelivery)
	deliverySlotsFree := len(coordinator.deliveryCapacity) < cap(coordinator.deliveryCapacity)
	coordinator.mu.Unlock()

	for _, holder := range holders {
		if holder.sessionID == session.ID {
			return errCodeDeliveryWorkspaceBusy
		}
	}
	if len(holders) > 0 {
		return fmt.Errorf("%w（占用方：会话 #%d）", errCodeDeliveryRepositoryBusy, holders[0].sessionID)
	}
	if !deliverySlotsFree {
		return fmt.Errorf("%w（当前上限 %d）", errCodeDeliveryCapacityBusy, cap(coordinator.deliveryCapacity))
	}
	// 键没冲突、槽也没满却仍然超时：多半是刚刚被别人抢走又放开，
	// 归到「稍后重试」这一类，别再误导用户去停自己的会话。
	return errCodeDeliveryCapacityBusy
}
