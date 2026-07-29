import axios from "axios"
import http from "@/api"
import type { ResultData } from "@/api/interface"
import type { AIGroup, CodeApproval, CodeApprovalPolicy, CodeExecutor, CodeSession, CodeSessionState } from "@/api/interface/code"
import type { Dashboard } from "@/api/interface/dashboard"

const mobileHttp = axios.create({
	baseURL: import.meta.env.VITE_API_URL as string,
	timeout: 15000,
	withCredentials: true
})

mobileHttp.interceptors.request.use(config => {
	config.headers.set("X-Mobile-Request", "1")
	return config
})

async function mobileRequest<T>(request: Promise<{ data: ResultData<T> }>) {
	const response = await request
	if (response.data.code !== 0) {
		throw new Error(response.data.msg || response.data.message || "Request failed")
	}
	return response.data.data
}

async function managementRequest<T>(request: Promise<ResultData<T>>) {
	const response = await request
	if (response.code !== 0) {
		throw new Error(response.msg || response.message || "Request failed")
	}
	return response.data
}

export interface MobileDevice {
	id: number
	name: string
	expiresAt: string
	lastSeenAt?: string
	lastIp: string
	revokedAt?: string
}

export interface MobileOverview {
	system: Dashboard.CurrentInfo
	sessions: CodeSession[]
	sessionTotal: number
	pendingApprovals: CodeApproval[]
	serverTime: string
}

export interface MobileContainer {
	containerID: string
	name: string
	imageName: string
	state: string
	runTime: string
	runtimeHost: string
	runtimeKind: string
	sourceType: string
	ports: string[]
	cpuPercent: number
	memoryUsage: number
	memoryLimit: number
	memoryPercent: number
}

export interface MobileContainerList {
	items: MobileContainer[]
	total: number
	running: number
	stopped: number
}

export interface MobileCodeStructureEntry {
	name: string
	path: string
	isDir: boolean
	extension: string
}

export interface MobileCodeStructureResult {
	path: string
	entries: MobileCodeStructureEntry[]
	truncated: boolean
}

export interface MobileCodeSessionFile {
	path: string
	content: string
	extension: string
	size: number
}

export function issueMobilePairing(deviceTtlDays: number) {
	return managementRequest(
		http.post<{ code: string; expiresAt: string; deviceTtlDays: number }>("/mobile/management/pair/issue", {
			deviceTtlDays
		})
	)
}

export function getMobileDevices() {
	return managementRequest(http.get<{ items: MobileDevice[]; total: number }>("/mobile/management/devices"))
}

export function revokeMobileDevice(id: number) {
	return managementRequest(http.post<void>(`/mobile/management/devices/${id}/revoke`))
}

export function exchangeMobilePairing(code: string, deviceName: string) {
	return mobileRequest(mobileHttp.post<ResultData<{ device: MobileDevice }>>("/mobile/pair/exchange", { code, deviceName }))
}

export function loginMobileDevice(data: {
	email: string
	password: string
	captchaToken?: string
	deviceName: string
}) {
	return mobileRequest(mobileHttp.post<ResultData<{ device: MobileDevice }>>("/mobile/login", data))
}

export function getMobileOverview() {
	return mobileRequest(mobileHttp.get<ResultData<MobileOverview>>("/mobile/app/overview")).then(result => ({
		...result,
		sessions: result.sessions || [],
		pendingApprovals: result.pendingApprovals || []
	}))
}

export function getMobileContainers() {
	return mobileRequest(mobileHttp.get<ResultData<MobileContainerList>>("/mobile/app/containers")).then(result => ({
		...result,
		items: result.items || []
	}))
}

export function operateMobileContainer(container: MobileContainer, operation: "start" | "stop" | "restart") {
	return mobileRequest(
		mobileHttp.post<ResultData<void>>("/mobile/app/containers/operate", {
			containerID: container.containerID,
			operation
		})
	)
}

export function getMobileSessions(page = 1, limit = 20) {
	return mobileRequest(
		mobileHttp.get<ResultData<{ items: CodeSession[]; total: number }>>("/mobile/app/sessions", {
			params: { page, limit }
		})
	).then(result => ({
		...result,
		items: result.items || []
	}))
}

export function getMobileProjects() {
	return mobileRequest(
		mobileHttp.get<ResultData<{ items: AIGroup[]; total: number }>>("/mobile/app/projects", {
			params: { page: 1, limit: 100 }
		})
	).then(result => ({ ...result, items: result.items || [] }))
}

export function getMobileExecutors() {
	return mobileRequest(mobileHttp.get<ResultData<CodeExecutor[]>>("/mobile/app/executors"))
}

export function createMobileSession(data: { title: string; projectId: number; executorId: string; approvalPolicy: CodeApprovalPolicy }) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeSession>>("/mobile/app/sessions", {
			...data,
			workDir: "",
			isolated: false
		})
	)
}

export function getMobileSessionState(sessionId: number) {
	return mobileRequest(mobileHttp.get<ResultData<CodeSessionState>>(`/mobile/app/sessions/${sessionId}/state`)).then(result => ({
		...result,
		recentMessages: result.recentMessages || [],
		previews: result.previews || [],
		timelineEvents: result.timelineEvents || [],
		changedFiles: result.changedFiles || []
	}))
}

export function getMobileSessionStructure(sessionId: number, path = "") {
	return mobileRequest(
		mobileHttp.get<ResultData<MobileCodeStructureResult>>(`/mobile/app/sessions/${sessionId}/structure`, {
			params: { path }
		})
	)
}

export function getMobileSessionFile(sessionId: number, path: string) {
	return mobileRequest(
		mobileHttp.get<ResultData<MobileCodeSessionFile>>(`/mobile/app/sessions/${sessionId}/file`, {
			params: { path }
		})
	)
}

export function saveMobileSessionFile(sessionId: number, path: string, content: string) {
	return mobileRequest(
		mobileHttp.put<ResultData<{ path: string; size: number }>>(`/mobile/app/sessions/${sessionId}/file`, {
			path,
			content
		})
	)
}

export function sendMobileInstruction(sessionId: number, content: string) {
	return mobileRequest(
		mobileHttp.post<ResultData<void>>(`/mobile/app/sessions/${sessionId}/instructions`, {
			content,
			allowCode: true,
			autoPreview: true
		})
	)
}

export function decideMobileApproval(approvalId: number, approved: boolean) {
	const decision = approved ? "approve" : "reject"
	return mobileRequest(mobileHttp.post<ResultData<void>>(`/mobile/app/approvals/${approvalId}/${decision}`, {}))
}

export function stopMobileSession(sessionId: number) {
	return mobileRequest(mobileHttp.post<ResultData<void>>(`/mobile/app/sessions/${sessionId}/stop`, {}))
}

export function retryMobileInstruction(instructionId: number) {
	return mobileRequest(mobileHttp.post<ResultData<void>>(`/mobile/app/instructions/${instructionId}/retry`, {}))
}

export function logoutMobileDevice() {
	return mobileRequest(mobileHttp.post<ResultData<void>>("/mobile/app/logout", {}))
}
