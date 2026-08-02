import axios from "axios"
import http from "@/api"
import type { ResultData } from "@/api/interface"
import type {
	AIProject,
	CodeApproval,
	CodeApprovalPolicy,
	CodeExecutor,
	CodeInstructionResponse,
	CodeSession,
	CodeSessionState,
	CodeWorktreeCapability
} from "@/api/interface/code"
import type { Dashboard } from "@/api/interface/dashboard"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import type { CodeProjectSyncStatus } from "@/api/interface/codeOverview"
import type { CodeDeliveryJob, CodeGitDeliveryResult, CodeGitStatus } from "@/api/interface/codeGit"
import type { NodeSummary, NodeWarning } from "./node"

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

export interface MobileNode {
	id: number
	name: string
	isLocal: boolean
	isProd: boolean
	status: "online" | "offline" | "unauthorized" | "unknown"
	version: string
	lastSeenAt?: string
	summary: NodeSummary
	warnings: NodeWarning[]
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

export interface MobileContainerPublishOptions {
	ports: Array<{ hostPort: number; containerPort: string }>
	websites: Array<{ id: number; alias: string; primaryDomain: string }>
}

export interface MobileWebsite {
	id: number
	alias: string
	primaryDomain: string
	otherDomains: string
	protocol: string
	type: string
	status: string
	appName: string
	redirectDomainsToPrimary: boolean
}

export interface MobileDatabase {
	type: string
	name: string
	server: string
	encoding: string
	comment: string
}

export interface MobileSSL {
	id: number
	primaryDomain: string
	type: string
	provider: string
	status: string
	autoRenew: boolean
	expireDate: string
}

export interface MobileApp {
	id: number
	name: string
	version: string
	status: string
	description: string
	httpPort: number
	httpsPort: number
	runtimeHost: string
	runtimeKind: string
}

export interface MobileResourceList<T> {
	items: T[]
	total: number
	warningCount?: number
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
	version: string
}

export interface MobileVersionInfo {
	versionName: string
	versionCode: number
	buildTime: string
	installPath: string
}

export interface MobileUpdateInfo {
	needUpdate: boolean
	curVersion: string
	latestVersionName: string
	latestVersionCode: number
	downloadUrl: string
	createAt: string
	content: string
	title: string
	description: string
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
	return mobileRequest(
		mobileHttp.post<ResultData<{ device: MobileDevice }>>("/mobile/pair/exchange", { code, deviceName })
	)
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

export function getMobileSystemVersion() {
	return mobileRequest(mobileHttp.get<ResultData<MobileVersionInfo>>("/mobile/app/system/version"))
}

export function checkMobileSystemUpdate(lang: string, appBrand: string) {
	return mobileRequest(
		mobileHttp.get<ResultData<MobileUpdateInfo>>("/mobile/app/system/check", {
			params: { lang, appBrand }
		})
	)
}

export function startMobileSystemUpgrade(currentVersion: string, targetVersion: string, lang: string) {
	return mobileRequest(
		mobileHttp.post<ResultData<{ log: string }>>("/mobile/app/system/upgrade", {
			containerName: "gopanel",
			currentVersion,
			targetVersion,
			lang
		})
	)
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

export function getMobileContainerPublishOptions(container: MobileContainer) {
	return mobileRequest(
		mobileHttp.get<ResultData<MobileContainerPublishOptions>>(
			`/mobile/app/containers/${encodeURIComponent(container.containerID)}/publish-options`,
			{ params: { runtimeHost: container.runtimeHost || "" } },
		),
	)
}

export function publishMobileContainerWebsite(data: {
	containerId: string
	runtimeHost: string
	websiteId: number
	hostPort: number
	scheme: "http" | "https"
}) {
	return mobileRequest(
		mobileHttp.post<ResultData<void>>("/mobile/app/containers/publish-website", data),
	)
}

function getMobileResourceList<T>(resource: string) {
	return mobileRequest(mobileHttp.get<ResultData<MobileResourceList<T>>>(`/mobile/app/resources/${resource}`)).then(
		result => ({ ...result, items: result.items || [] })
	)
}

export function getMobileWebsites() {
	return getMobileResourceList<MobileWebsite>("websites")
}

export function updateMobileWebsiteDomainBindings(data: {
	websiteId: number
	primaryDomain: string
	otherDomains: string
	redirectDomainsToPrimary: boolean
}) {
	return mobileRequest(
		mobileHttp.post<ResultData<void>>("/mobile/app/resources/websites/domains", { ...data, confirm: true }),
	)
}

export function getMobileDatabases() {
	return getMobileResourceList<MobileDatabase>("databases")
}

export function getMobileSSLs() {
	return getMobileResourceList<MobileSSL>("ssl")
}

export function getMobileApps() {
	return getMobileResourceList<MobileApp>("apps")
}

export function getMobileNodes() {
	return mobileRequest(mobileHttp.get<ResultData<MobileNode[]>>("/mobile/app/nodes"))
}

export function getMobileSessions(projectId: number, page = 1, limit = 50) {
	return mobileRequest(
		mobileHttp.get<ResultData<{ items: CodeSession[]; total: number }>>("/mobile/app/sessions", {
			params: { page, limit, projectId }
		})
	).then(result => ({
		...result,
		items: result.items || []
	}))
}

export function getMobileProjects() {
	return mobileRequest(
		mobileHttp.get<ResultData<{ items: AIProject[]; total: number }>>("/mobile/app/projects", {
			params: { page: 1, limit: 100 }
		})
	).then(result => ({ ...result, items: result.items || [] }))
}

export function getMobileProjectSyncStatus(projectId: number) {
	return mobileRequest(
		mobileHttp.get<ResultData<CodeProjectSyncStatus>>(`/mobile/app/projects/${projectId}/git/sync`)
	)
}

export function syncMobileProject(projectId: number) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeProjectSyncStatus>>(
			`/mobile/app/projects/${projectId}/git/sync`,
			{ confirm: true },
			{ timeout: 70000 }
		)
	)
}

export function getMobileExecutors() {
	return mobileRequest(mobileHttp.get<ResultData<CodeExecutor[]>>("/mobile/app/executors"))
}

export function getMobileWorktreeCapability(projectId: number) {
	return mobileRequest(
		mobileHttp.get<ResultData<CodeWorktreeCapability>>(`/mobile/app/projects/${projectId}/worktree-capability`)
	)
}

export function openMobileProjectTerminal(projectId: number) {
	return mobileRequest(
		mobileHttp.post<ResultData<HostTerminalSession>>(`/mobile/app/projects/${projectId}/terminal`, {})
	)
}

export function createMobileSession(data: {
	title: string
	projectId: number
	executorId: string
	approvalPolicy: CodeApprovalPolicy
}) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeSession>>("/mobile/app/sessions", {
			...data,
			workDir: "",
			isolated: true,
			includeUncommitted: false
		})
	)
}

export function updateMobileSessionTitle(sessionId: number, title: string) {
	return mobileRequest(mobileHttp.put<ResultData<CodeSession>>(`/mobile/app/sessions/${sessionId}/title`, { title }))
}

export function deliverMobileSession(sessionId: number) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeDeliveryJob>>(`/mobile/app/sessions/${sessionId}/worktree/merge`, {
			confirm: true
		})
	)
}

export function getMobileGitStatus(sessionId: number) {
	return mobileRequest(
		mobileHttp.get<ResultData<CodeGitStatus>>(`/mobile/app/sessions/${sessionId}/git/status`)
	)
}

export function saveMobileGitChanges(sessionId: number, message = "") {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeGitDeliveryResult>>(`/mobile/app/sessions/${sessionId}/git/save`, { message })
	)
}

export function getMobileSessionState(sessionId: number) {
	return mobileRequest(mobileHttp.get<ResultData<CodeSessionState>>(`/mobile/app/sessions/${sessionId}/state`)).then(
		result => ({
			...result,
			recentMessages: result.recentMessages || [],
			previews: result.previews || [],
			timelineEvents: result.timelineEvents || [],
			changedFiles: result.changedFiles || []
		})
	)
}

export function createMobileInstruction(sessionId: number, content: string) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeInstructionResponse>>(`/mobile/app/sessions/${sessionId}/instructions`, {
			content,
			allowCode: true,
			autoPreview: true
		})
	)
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

export function saveMobileSessionFile(sessionId: number, path: string, content: string, baseVersion: string) {
	return mobileRequest(
		mobileHttp.put<ResultData<{ path: string; size: number; version: string }>>(
			`/mobile/app/sessions/${sessionId}/file`,
			{
				path,
				content,
				baseVersion
			}
		)
	)
}

export function decideMobileApproval(approvalId: number, approved: boolean, reason = "") {
	const decision = approved ? "approve" : "reject"
	return mobileRequest(
		mobileHttp.post<ResultData<void>>(`/mobile/app/approvals/${approvalId}/${decision}`, { reason })
	)
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
