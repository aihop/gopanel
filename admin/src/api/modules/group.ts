import type { Group } from "../interface/group"
import http from "@/api"

export const GetGroupList = (params: Group.GroupSearch) => {
	return http.post<Array<Group.GroupInfo>>(`/group/search`, params)
}
export const CreateGroup = (params: Group.GroupCreate) => {
	return http.post<Group.GroupCreate>(`/group/create`, params)
}
export const UpdateGroup = (params: Group.GroupUpdate) => {
	return http.post(`/group/update`, params)
}
export const DeleteGroup = (id: number) => {
	return http.post(`/group/del`, { id: id })
}
