import http from "@/api"
import type { Website } from "../interface/website"
import type { ResPage } from "../interface"
import type { WebsiteDiagnosticSetting, WebsiteIssue, WebsiteIssueDetail, WebsiteProbe } from "../interface/websiteDiagnostic"
 
export const websiteListAPI = () => {
	return http.post<ResPage<Website.WebsiteDTO>>(`/website/list`)
}

export const websiteCreateAPI = (req: Website.WebSiteCreateReq) => {
	return http.post<void>(`/website/create`, req)
}
 
export const websiteUpdateAPI = (req: Website.WebSiteUpdateReq) => {
	return http.put<void>(`/website/${req.id}`, req)
}

export const websiteDeleteAPI = (params: Website.WebSiteDel) => {
	return http.delete<void>(`/website/${params.id}`,params)
}

export const WebsiteLogAPI = (req: Website.WebSiteLogReadReq) => {
	return http.post<Website.WebSiteLog>(`/website/log`, req)
}

export const WebsiteTodayIPStatsAPI = (req: Website.WebSiteTodayIPStatsReq) => {
	return http.post<Website.WebSiteTodayIPStats>(`/website/log/today-ip`, req)
}

export const getWebsiteDiagnosticSettingAPI = (websiteId: number) => {
	return http.get<WebsiteDiagnosticSetting>(`/website/${websiteId}/diagnostics/settings`)
}

export const saveWebsiteDiagnosticSettingAPI = (
	websiteId: number,
	setting: Omit<WebsiteDiagnosticSetting, "websiteId" | "configured" | "trackingDir" | "hookSecretConfigured" | "remoteEndpoint">
) => {
	return http.put<WebsiteDiagnosticSetting>(`/website/${websiteId}/diagnostics/settings`, setting)
}

export const rotateWebsiteDiagnosticHookSecretAPI = (websiteId: number) => {
	return http.post<{ secret: string }>(`/website/${websiteId}/diagnostics/hook-secret`)
}

export const listWebsiteDiagnosticIssuesAPI = (websiteId: number, params: { page: number; limit: number; status?: string }) => {
	return http.get<ResPage<WebsiteIssue>>(`/website/${websiteId}/diagnostics/issues`, params)
}

export const getWebsiteDiagnosticIssueAPI = (websiteId: number, issueId: number) => {
	return http.get<WebsiteIssueDetail>(`/website/${websiteId}/diagnostics/issues/${issueId}`)
}

export const updateWebsiteDiagnosticIssueAPI = (websiteId: number, issueId: number, action: "confirm" | "ignore" | "reopen") => {
	return http.post<WebsiteIssue>(`/website/${websiteId}/diagnostics/issues/${issueId}/action`, { action })
}

export const handoffWebsiteDiagnosticIssueAPI = (
	websiteId: number,
	issueId: number,
	payload: { requirement: string; allowCode: boolean; runChecks: boolean }
) => {
	return http.post<WebsiteIssue>(`/website/${websiteId}/diagnostics/issues/${issueId}/code`, payload)
}

export const verifyWebsiteDiagnosticIssueAPI = (websiteId: number, issueId: number, release: string) => {
	return http.post<WebsiteIssue>(`/website/${websiteId}/diagnostics/issues/${issueId}/verify`, { release })
}

export const listWebsiteDiagnosticProbesAPI = (websiteId: number) => {
	return http.get<WebsiteProbe[]>(`/website/${websiteId}/diagnostics/probes`)
}

export const saveWebsiteDiagnosticProbesAPI = (websiteId: number, probes: WebsiteProbe[]) => {
	return http.put<WebsiteProbe[]>(`/website/${websiteId}/diagnostics/probes`, probes)
}

export const runWebsiteDiagnosticProbeAPI = (websiteId: number, probeId: number) => {
	return http.post<WebsiteProbe>(`/website/${websiteId}/diagnostics/probes/${probeId}/run`)
}


export const AppDeployListAPI = (req: { websiteId: number }) => {
	return http.post<Website.AppDeployRecord[]>(`/website/app-deploy/list`, req)
}

export const AppDeploySwitchAPI = (req: { deployId: number }) => {
	return http.post<void>(`/website/app-deploy/switch`, req)
}

export const AppDeployDeleteAPI = (req: { deployId: number }) => {
	return http.post<void>(`/website/app-deploy/delete`, req)
}

export const AppDeployTriggerAPI = (req: Website.AppDeployTriggerReq) => {
	return http.post<void>(`/website/app-deploy/trigger`, req)
}

export const AppDeploySnapshotAPI = (req: { websiteId: number }) => {
	return http.post<void>(`/website/app-deploy/snapshot`, req)
}

 
 

 
 
 
