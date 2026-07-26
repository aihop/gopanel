export namespace Disk {
	export interface DiskUsage {
		path: string
		type: string
		device: string
		total: number
		free: number
		used: number
		usedPercent: number
		inodesTotal: number
		inodesUsed: number
		inodesFree: number
		inodesUsedPercent: number
	}

	export interface FileItem {
		path: string
		size: number
		modTime: string
		/** log / cache / container / archive / coredump / temp / backup / other */
		category: string
		/** 容器存储目录下的文件，不能直接删，要走 prune */
		isContainer: boolean
		/** 当前运行条件下能否清理（只读卷、属主不符、系统路径都会是 false） */
		removable: boolean
		/** removable 为 false 时的原因 */
		reason?: string
	}

	export interface DirItem {
		path: string
		size: number
		count: number
	}

	export interface Progress {
		scannedFiles: number
		scannedBytes: number
		currentDir: string
		errors: number
	}

	export interface ScanResult {
		roots: string[]
		minSize: number
		files: FileItem[]
		dirs: DirItem[]
		scannedFiles: number
		scannedBytes: number
		errors: number
		startedAt: string
		finishedAt: string
	}

	export interface ScanTask {
		id: string
		/** running / success / failed / canceled */
		status: string
		error?: string
		viaGpc: boolean
		/** 非 root 且 gpc 不可用时退回本机权限扫描，结果不完整 */
		degraded: boolean
		degradedReason?: string
		/** 是否有实时进度。走 gpc 时为 false（单请求单响应，扫描期间没有中间输出） */
		progressLive: boolean
		progress: Progress
		result?: ScanResult
		createdAt: string
		expiresAt: string
	}

	export interface ScanRequest {
		roots: string[]
		minSize: number
		topN: number
		crossDevice: boolean
	}

	export interface CleanResult {
		path: string
		ok: boolean
		freed: number
		message?: string
	}

	/** gpc helper 可用性诊断：非 root 面板没有它时扫描/清理只能覆盖当前用户可读的文件 */
	export interface GpcStatus {
		/** 面板非 root 才需要 gpc */
		needed: boolean
		available: boolean
		installed: boolean
		socketPath: string
		/** 一句话结论，直接展示 */
		hint: string
		/** 建议在服务器上以 root 执行的修复命令 */
		commands?: string[]
	}
}
