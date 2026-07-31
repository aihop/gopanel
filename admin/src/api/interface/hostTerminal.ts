export interface HostTerminalSession {
	id: number
	createdAt: string
	updatedAt: string
	userId: number
	status: "starting" | "running" | "exited" | "stopped" | "failed" | "interrupted"
	shell: string
	workDir: string
	pid: number
	exitCode: number
	clientIp: string
	outputBytes: number
	errorMessage?: string
	startedAt: string
	endedAt?: string
}

export interface HostTerminalAuditEvent {
	id: number
	createdAt: string
	sessionId: number
	userId: number
	action: string
	status: string
	ip: string
	detail: string
}

export interface CreateHostTerminalRequest {
	shell: string
	workDir: string
	cols: number
	rows: number
}

export interface HostTerminalCapabilities {
	defaultShell: string
	shells: string[]
}
