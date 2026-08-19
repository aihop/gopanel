import { mobileMessages } from "./mobile"

const alignmentMessages = {
	zh: {
		mobile: {
			recentTasks: "最近任务",
			archivedTasks: "已归档",
			activeTasks: "进行中",
			archiveTask: "归档任务",
			restoreTask: "恢复任务",
			taskArchived: "任务已归档",
			taskRestored: "任务已恢复",
			taskArchiveFailed: "任务归档操作失败",
			taskListLoadFailed: "任务列表加载失败",
			noRecentTasks: "暂无最近任务",
			noArchivedTasks: "暂无归档任务",
			taskStatus_active: "进行中",
			taskStatus_idle: "等待任务",
			taskStatus_interactive: "可交互",
			taskStatus_task_ready: "任务就绪",
			taskStatus_instruction_queued: "指令排队中",
			taskStatus_awaiting_approval: "等待确认",
			taskStatus_approval_rejected: "已拒绝",
			taskStatus_executing: "执行中",
			taskStatus_preview_ready: "预览就绪",
			taskStatus_queued: "排队中",
			taskStatus_running: "执行中",
			taskStatus_pending_approval: "等待确认",
			taskStatus_delivering: "交付中",
			taskStatus_completed: "已完成",
			taskStatus_failed: "失败",
			taskStatus_cancelled: "已取消",
			taskStatus_unknown: "未知状态",
			remoteCodeUnavailable: "远程节点暂不开放开发工作区",
			remoteCodeControllerHint:
				"开发会话包含用户级身份和审计上下文，请切回主控机后使用；远程节点资源仍可在资源页管理。"
		}
	},
	en: {
		mobile: {
			recentTasks: "Recent tasks",
			archivedTasks: "Archived",
			activeTasks: "Active",
			archiveTask: "Archive task",
			restoreTask: "Restore task",
			taskArchived: "Task archived",
			taskRestored: "Task restored",
			taskArchiveFailed: "Failed to update task archive",
			taskListLoadFailed: "Failed to load tasks",
			noRecentTasks: "No recent tasks",
			noArchivedTasks: "No archived tasks",
			taskStatus_active: "Active",
			taskStatus_idle: "Waiting",
			taskStatus_interactive: "Interactive",
			taskStatus_task_ready: "Ready",
			taskStatus_instruction_queued: "Queued",
			taskStatus_awaiting_approval: "Approval required",
			taskStatus_approval_rejected: "Rejected",
			taskStatus_executing: "Running",
			taskStatus_preview_ready: "Preview ready",
			taskStatus_queued: "Queued",
			taskStatus_running: "Running",
			taskStatus_pending_approval: "Approval required",
			taskStatus_delivering: "Delivering",
			taskStatus_completed: "Completed",
			taskStatus_failed: "Failed",
			taskStatus_cancelled: "Cancelled",
			taskStatus_unknown: "Unknown",
			remoteCodeUnavailable: "Code workspace is unavailable on remote nodes",
			remoteCodeControllerHint:
				"Development sessions require user-scoped identity and audit context. Switch to the controller; remote node resources remain available under Resources."
		}
	}
} as const

export const mobileAlignmentMessages = {
	zh: { mobile: { ...mobileMessages.zh.mobile, ...alignmentMessages.zh.mobile } },
	en: { mobile: { ...mobileMessages.en.mobile, ...alignmentMessages.en.mobile } }
} as const
