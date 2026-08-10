import type { ResultData } from "@/api/interface"
import type {
	AIProject,
	CodeApprovalPolicy,
	CodeExecutor,
	CodeInstructionResponse,
	CodeSession,
	CodeSessionState,
	CodeWorktreeCapability
} from "@/api/interface/code"
import type { HostTerminalSession } from "@/api/interface/hostTerminal"
import type { CodeProjectSyncStatus } from "@/api/interface/codeOverview"
import type {
	CodeDeliveryJob,
	CodeDeliveryPushResult,
	CodeGitDeliveryResult,
	CodeGitStatus
} from "@/api/interface/codeGit"
import { waitForCodeSessionInitialization } from "./codeSessionInitialization"
import { mobileHttp, mobileRequest } from "./mobileClient"

// 移动端的 Code 项目 / 会话 / 交付接口。与设备、资源类接口分文件维护。

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

export function createMobileSession(
	data: {
		title: string
		projectId: number
		executorId: string
		approvalPolicy: CodeApprovalPolicy
	},
	messages: { initializationFailed: string; initializationTimedOut: string }
) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeSession>>("/mobile/app/sessions", {
			...data,
			workDir: "",
			isolated: true,
			includeUncommitted: true
		})
	).then(async session => {
		if (session.status !== "initializing") return session
		await waitForCodeSessionInitialization(
			() => mobileRequest(
				mobileHttp.get<ResultData<import("@/api/interface/code").CodeSessionInitialization>>(`/mobile/app/sessions/${session.id}/initialization`)
			),
			{ failed: messages.initializationFailed, timedOut: messages.initializationTimedOut }
		)
		const state = await getMobileSessionState(session.id)
		return state.session
	})
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

export function getMobileDeliveryPushStatus(sessionId: number) {
	return mobileRequest(
		mobileHttp.get<ResultData<CodeDeliveryPushResult>>(`/mobile/app/sessions/${sessionId}/delivery/push`)
	)
}

export function pushMobileSessionDelivery(sessionId: number) {
	return mobileRequest(
		mobileHttp.post<ResultData<CodeDeliveryPushResult>>(`/mobile/app/sessions/${sessionId}/delivery/push`, {
			confirm: true
		})
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
