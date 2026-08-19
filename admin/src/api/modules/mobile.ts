import http from "@/api"
import type { ResultData } from "@/api/interface"
import type { NodeSummary, NodeWarning } from "./node"
import type { MobileOverview } from "../interface/mobileControlPlane"
import { mobileHttp, mobileNodePath, mobileRequest } from "./mobileClient"
export type { MobileOverview } from "../interface/mobileControlPlane"

// Code 项目 / 会话 / 交付接口拆到 mobileCode.ts，这里重新导出以保持导入路径不变。
export * from "./mobileCode"

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
	hasControlToken: boolean
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
		http.post<{ code: string; expiresAt: string; deviceTtlDays: number; entrancePath: string }>(
			"/mobile/management/pair/issue",
			{
				deviceTtlDays
			}
		)
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

export function getMobileContainers(nodeId = 0) {
	return mobileRequest(mobileHttp.get<ResultData<MobileContainerList>>(mobileNodePath(nodeId, "/containers"))).then(
		result => ({
			...result,
			items: result.items || []
		})
	)
}

export function operateMobileContainer(
	container: MobileContainer,
	operation: "start" | "stop" | "restart",
	nodeId = 0
) {
	return mobileRequest(
		mobileHttp.post<ResultData<void>>(mobileNodePath(nodeId, "/containers/operate"), {
			containerID: container.containerID,
			operation
		})
	)
}

export function getMobileContainerPublishOptions(container: MobileContainer, nodeId = 0) {
	return mobileRequest(
		mobileHttp.get<ResultData<MobileContainerPublishOptions>>(
			mobileNodePath(nodeId, `/containers/${encodeURIComponent(container.containerID)}/publish-options`),
			{ params: { runtimeHost: container.runtimeHost || "" } }
		)
	)
}

export function publishMobileContainerWebsite(
	data: {
		containerId: string
		runtimeHost: string
		websiteId: number
		hostPort: number
		scheme: "http" | "https"
	},
	nodeId = 0
) {
	return mobileRequest(mobileHttp.post<ResultData<void>>(mobileNodePath(nodeId, "/containers/publish-website"), data))
}

function getMobileResourceList<T>(resource: string, nodeId = 0) {
	return mobileRequest(
		mobileHttp.get<ResultData<MobileResourceList<T>>>(mobileNodePath(nodeId, `/resources/${resource}`))
	).then(result => ({ ...result, items: result.items || [] }))
}

export function getMobileWebsites(nodeId = 0) {
	return getMobileResourceList<MobileWebsite>("websites", nodeId)
}

export function updateMobileWebsiteDomainBindings(
	data: {
		websiteId: number
		primaryDomain: string
		otherDomains: string
		redirectDomainsToPrimary: boolean
	},
	nodeId = 0
) {
	return mobileRequest(
		mobileHttp.post<ResultData<void>>(mobileNodePath(nodeId, "/resources/websites/domains"), {
			...data,
			confirm: true
		})
	)
}

export function getMobileDatabases(nodeId = 0) {
	return getMobileResourceList<MobileDatabase>("databases", nodeId)
}

export function getMobileSSLs(nodeId = 0) {
	return getMobileResourceList<MobileSSL>("ssl", nodeId)
}

export function getMobileApps(nodeId = 0) {
	return getMobileResourceList<MobileApp>("apps", nodeId)
}

export function getMobileNodes() {
	return mobileRequest(mobileHttp.get<ResultData<MobileNode[]>>("/mobile/app/nodes"))
}

export function logoutMobileDevice() {
	return mobileRequest(mobileHttp.post<ResultData<void>>("/mobile/app/logout", {}))
}
