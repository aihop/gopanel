export namespace HostTool {
	export interface HostTool {
		type: string
		config: {}
	}

	export interface Daemon extends HostTool {
		configPath: string
		includeDir: string
		logPath: string
		isExist: boolean
		init: boolean
		msg: string
		version: string
		status: string
		ctlExist: boolean
		serviceName: string
	}

	export interface DaemonConfig {
		type: string
		operate: string
		content?: string
	}

	export interface DaemonConfigRes {
		type: string
		content: string
	}

	export interface DaemonInit {
		type: string
		configPath: string
		serviceName: string
	}

	export interface DaemonProcess {
		operate: string
		name: string
		command: string
		user: string
		dir: string
		numprocs: string
		status?: ProcessStatus[]
	}

	export interface ProcessStatus {
		PID: string
		status: string
		uptime: string
		name: string
	}

	export interface ProcessReq {
		operate: string
		name: string
	}

	export interface ProcessFileReq {
		operate: string
		name: string
		content?: string
		file: string
	}
}
