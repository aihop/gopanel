export namespace Process {
	export interface ListReq {
		pid?: number
		username?: string
		name?: string
	}

	export interface StopReq {
		PID: number
	}

	export interface PortReq {
		port: number
	}
}
