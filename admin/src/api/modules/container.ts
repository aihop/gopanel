import http from "@/api"
import type { ResPage, SearchWithPage } from "../interface"
import type { Container } from "../interface/container"
import { TimeoutEnum } from "@/enums/http-enum"
import type { Response } from "@/types/api/response"

export const containerListAPI = (params: Container.ContainerSearch) => {
	return http.post<ResPage<Container.ContainerInfo>>(`/container/list`, params, TimeoutEnum.T_40S)
}
export const containerAllAPI = () => {
	return http.post<Array<string>>(`/container/all`, {})
}
export const loadResourceLimit = () => {
	return http.get<Container.ResourceLimit>(`/container/limit`)
}
export const createContainer = (params: Container.ContainerHelper) => {
	return http.post(`/container/create`, params, TimeoutEnum.T_10M)
}
export const updateContainer = (params: Container.ContainerHelper) => {
	return http.post(`/container/update`, params, TimeoutEnum.T_10M)
}
export const upgradeContainer = (name: string, image: string, forcePull: boolean) => {
	return http.post(`/container/upgrade`, { name: name, image: image, forcePull: forcePull }, TimeoutEnum.T_10M)
}
export const commitContainer = (params: Container.ContainerCommit) => {
	return http.post(`/container/commit`, params)
}
export const loadContainerInfo = (name: string, runtimeHost?: string) => {
	return http.post<Container.ContainerHelper>(`/container/info`, { name: name, runtimeHost: runtimeHost || "" })
}
export const containerCleanLogsAPI = (containerName: string) => {
	return http.post(`/container/clean/logs`, { name: containerName })
}
export const loadContainerLogs = (type: string, name: string) => {
	return http.post<string>(`/container/load/logs`, { type: type, name: name })
}
export const containerStatsAPI = () => {
	return http.get<Array<Container.ContainerListStats>>(`/container/stats`)
}
export const containerStatsGetAPI = (id: string) => {
	return http.get<Container.ContainerStats>(`/container/stats/${id}`)
}
export const containerRename = (params: Container.ContainerRename) => {
	return http.post(`/container/rename`, params)
}
export const containerOperator = (params: Container.ContainerOperate) => {
	return http.post(`/container/operate`, params, TimeoutEnum.T_60S)
}
export const containerPrune = (params: Container.ContainerPrune) => {
	return http.post<Container.ContainerPruneReport>(`/container/prune`, params)
}
export const inspect = (params: Container.ContainerInspect) => {
	return http.post<string>(`/container/inspect`, params)
}

export const DownloadFile = (params: Container.ContainerLogInfo) => {
	return http.download<BlobPart>("/container/download/logs", params, {
		responseType: "blob",
		timeout: TimeoutEnum.T_40S
	})
}

// image
export const containerImageListAPI = (params: SearchWithPage) => {
	return http.post<ResPage<Container.ImageInfo>>(`/container/image/list`, params)
}

export const listAllImage = () => {
	return http.get<Array<Container.ImageInfo>>(`/container/image/all`)
}
export const listImage = () => {
	return http.get<Array<Container.Options>>(`/container/image`)
}
export const imageBuild = (params: Container.ImageBuild) => {
	return http.post<string>(`/container/image/build`, params)
}
export const imagePull = (params: Container.ImagePull) => {
	return http.post<string>(`/container/image/pull`, params)
}
export const imagePush = (params: Container.ImagePush) => {
	return http.post<string>(`/container/image/push`, params)
}
export const imageLoad = (params: Container.ImageLoad) => {
	return http.post(`/container/image/load`, params, TimeoutEnum.T_10M)
}
export const imageSave = (params: Container.ImageSave) => {
	return http.post(`/container/image/save`, params, TimeoutEnum.T_10M)
}
export const imageTag = (params: Container.ImageTag) => {
	return http.post(`/container/image/tag`, params)
}
export const imageRemove = (params: Container.BatchDelete) => {
	return http.post(`/container/image/remove`, params)
}

// network
export const containerNetworkListAPI = (params: SearchWithPage) => {
	return http.post<ResPage<Container.NetworkInfo>>(`/container/network/list`, params)
}
export const listNetwork = () => {
	return http.get<Array<Container.Options>>(`/container/network`)
}
export const deleteNetwork = (params: Container.BatchDelete) => {
	return http.post(`/container/network/del`, params)
}
export const createNetwork = (params: Container.NetworkCreate) => {
	return http.post(`/container/network`, params)
}

// volume
export const containerVolumeListAPI = (params: SearchWithPage) => {
	return http.post<ResPage<Container.VolumeInfo>>(`/container/volume/list`, params)
}
export const listVolume = () => {
	return http.get<Array<Container.Options>>(`/container/volume`)
}
export const deleteVolume = (params: Container.BatchDelete) => {
	return http.post(`/container/volume/del`, params)
}
export const createVolume = (params: Container.VolumeCreate) => {
	return http.post(`/container/volume`, params)
}

// repo
export const checkRepoStatus = (id: number) => {
	return http.post(`/container/repo/status`, { id: id }, TimeoutEnum.T_40S)
}
export const containerRepoListAPI = (params: SearchWithPage) => {
	return http.post<ResPage<Container.RepoInfo>>(`/container/repo/list`, params)
}
export const listImageRepo = () => {
	return http.get<Container.RepoOptions>(`/container/repo`)
}
export const createImageRepo = (params: Container.RepoCreate) => {
	return http.post(`/container/repo`, params, TimeoutEnum.T_40S)
}
export const updateImageRepo = (params: Container.RepoUpdate) => {
	return http.post(`/container/repo/update`, params, TimeoutEnum.T_40S)
}
export const deleteImageRepo = (params: Container.RepoDelete) => {
	return http.post(`/container/repo/del`, params, TimeoutEnum.T_40S)
}

// composeTemplate
export const searchComposeTemplate = (params: SearchWithPage) => {
	return http.post<ResPage<Container.TemplateInfo>>(`/container/template/search`, params)
}
export const listComposeTemplate = () => {
	return http.get<Container.TemplateInfo>(`/container/template`)
}
export const deleteComposeTemplate = (params: { ids: number[] }) => {
	return http.post(`/container/template/del`, params)
}
export const createComposeTemplate = (params: Container.TemplateCreate) => {
	return http.post(`/container/template`, params)
}
export const updateComposeTemplate = (params: Container.TemplateUpdate) => {
	return http.post(`/container/template/update`, params)
}

// compose
export const containerComposeListAPI = (params: SearchWithPage) => {
	return http.post<ResPage<Container.ComposeInfo>>(`/container/compose/list`, params)
}
export const upCompose = (params: Container.ComposeCreate) => {
	return http.post<string>(`/container/compose`, params)
}
export const testCompose = (params: Container.ComposeCreate) => {
	return http.post<boolean>(`/container/compose/test`, params)
}
export const composeOperator = (params: Container.ComposeOperation) => {
	return http.post(`/container/compose/operate`, params)
}
export const composeUpdate = (params: Container.ComposeUpdate) => {
	return http.post(`/container/compose/update`, params, TimeoutEnum.T_10M)
}

// docker
export const dockerOperate = (operation: string) => {
	return http.post(`/container/docker/operate`, { operation: operation })
}
 
export const loadDaemonFile = () => {
	return http.get<string>(`/container/engine/file`)
}
export const loadInstanceStatus = () => {
	return http.get<string>(`/container/engine/status`)
}

export const containerValidateAPI = () => {
	return http.get<any>(`/container/engine/validate`)
}

export const repairPodmanSocketAPI = (group?: string) => {
	return http.post(`/container/repair/podman-socket`, group ? { group } : {})
}

export const repairSystemdLingerAPI = () => {
	return http.post(`/container/repair/linger`, {})
}
export const updateDaemonUpdate = (key: string, value: string) => {
	return http.post(`/container/engine/update`, { key: key, value: value }, TimeoutEnum.T_60S)
}
export const updateLogOption = (maxSize: string, maxFile: string) => {
	return http.post(`/container/options/log`, { logMaxSize: maxSize, logMaxFile: maxFile }, TimeoutEnum.T_60S)
}
export const updateIpv6Option = (fixedCidrV6: string, ip6Tables: boolean, experimental: boolean) => {
	return http.post(
		`/container/options/ipv6`,
		{ fixedCidrV6: fixedCidrV6, ip6Tables: ip6Tables, experimental: experimental },
		TimeoutEnum.T_60S
	)
}
export const updateDaemonByfile = (params: Container.DaemonJsonUpdateByFile) => {
	return http.post(`/container/engine/update-file`, params)
}

export const containerDaemonConfigAPI = () => {
	return http.get<Response<any>>(`/container/engine/config`)
}
 

export async function containerInstanceOperateAPI(data: any) {
	return http.post<Response<any>>(`/container/engine/operate`, data)
}
