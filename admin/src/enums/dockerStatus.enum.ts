export const dockerStatus = {
	Running: "Running",
	Stopped: "Stopped"
} as const

export type dockerStatusEnum = (typeof dockerStatus)[keyof typeof dockerStatus]

export const dockerStatusText: Record<dockerStatusEnum, string> = {
	[dockerStatus.Running]: "已启动",
	[dockerStatus.Stopped]: "已停止"
}
