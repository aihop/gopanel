package api

import (
	"context"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

// 切到对话只是换视图，不再抢占会话——但「真要写的时候必须先接管」这条边界不能松。
// 前端不再在切换时请求接管之后，这条就成了唯一的保障：发指令走
// acquireCodeInstructionSession，它会先停掉终端再拿指令租约。
//
// 这道边界防的是两边同时往一棵工作树写：坏掉的是 Git 对象和分支指针，
// 不是「两个人各改各的文件」。
func TestInstructionAcquisitionStillStopsTerminalFirst(t *testing.T) {
	coordinator := newCodeExecutionCoordinator(4, 2)
	original := codeExecutions
	codeExecutions = coordinator
	t.Cleanup(func() { codeExecutions = original })

	session := &model.AIDevSession{ID: 51, SourceWorkDir: "/repo/handover", WorkDir: "/repo/handover"}
	terminal, err := coordinator.acquireOwned(
		context.Background(), session.ID, codeExecutionWorkspaceKeys(session), codeExecutionInteractive, false,
	)
	if err != nil {
		t.Fatalf("终端应能先占住工作区：%v", err)
	}

	// 真实链路：cancel 回调杀掉 CLI 进程，进程退出后 terminal.wait 释放租约。
	// 这里照样先记录再释放，否则接管会一直等租约归还直到超时。
	killed := make(chan struct{})
	terminal.SetCancel(func() {
		close(killed)
		terminal.Release()
	})

	lease, err := acquireCodeInstructionSession(context.Background(), session)
	if err != nil {
		t.Fatalf("发指令时应能完成接管并拿到租约：%v", err)
	}
	defer lease.Release()

	select {
	case <-killed:
	default:
		t.Fatal("拿到指令租约前必须已经停掉终端，否则两边会同时写同一棵工作树")
	}
}
