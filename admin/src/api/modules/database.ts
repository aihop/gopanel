import http from "@/api"

export const databaseListAPI = (params: any) => {
	return http.post(`/database/list`, params)
}

export const databaseCountAPI = (params: any) => {
	return http.post(`/database/count`, params)
}

export const databaseCommentAPI = (params: any) => {
	return http.post(`/database/comment`, params)
}

export const databaseCreateAPI = (params: any) => {
	return http.post(`/database/create`, params)
}

export const databaseDeleteAPI = (params: any) => {
	return http.post(`/database/delete`, params)
}

export const databaseServerListAPI = (params: any) => {
	return http.post(`/database/server/list`, params)
}

export const databaseServerCountAPI = (params: any) => {
	return http.post(`/database/server/count`, params)
}

export const databaseServerCreateAPI = (params: any) => {
	return http.post(`/database/server/create`, params)
}

export const databaseServerUpdateAPI = (params: any) => {
	return http.post(`/database/server/update`, params)
}

export const databaseServerGetAPI = (params: any) => {
	return http.post(`/database/server/get`, params)
}

export const databaseServerDeleteAPI = (params: any) => {
	return http.post(`/database/server/delete`, params)
}

export const databaseServerSyncAPI = (params: any) => {
	return http.post(`/database/server/sync`, params)
}

export const databaseUserListAPI = (params: any) => {
	return http.post(`/database/user/list`, params)
}

export const databaseUserCountAPI = (params: any) => {
	return http.post(`/database/user/count`, params)
}

export const databaseUserCreateAPI = (params: any) => {
	return http.post(`/database/user/create`, params)
}

export const databaseUserUpdateAPI = (params: any) => {
	return http.post(`/database/user/update`, params)
}

export const databaseUserDeleteAPI = (params: any) => {
	return http.post(`/database/user/delete`, params)
}

// DB Manager API
export const getDBManagerTablesAPI = (data: { serverId: number; databaseName: string }) => {
	return http.post<string[]>(`/database/manager/tables`, data)
}

export const getDBManagerTableListAPI = (data: { serverId: number; databaseName: string; page: number; limit: number; keyword?: string }) => {
	return http.post<any>(`/database/manager/table-list`, data)
}

export const getDBManagerTableDataAPI = (data: { serverId: number; databaseName: string; tableName: string; page: number; limit: number; searchColumn?: string; searchValue?: string; advancedSearch?: any[] }) => {
	return http.post<any>(`/database/manager/data`, data)
}

export const execDBManagerSqlAPI = (data: { serverId: number; databaseName: string; sql: string }) => {
	return http.post<any>(`/database/manager/exec`, data)
}

export const insertDBManagerRecordAPI = (data: any) => {
	return http.post(`/database/manager/insert`, data)
}

export const updateDBManagerRecordAPI = (data: any) => {
	return http.post(`/database/manager/update`, data)
}

export const deleteDBManagerRecordAPI = (data: any) => {
	return http.post(`/database/manager/delete`, data)
}

export const databaseUserGetAPI = (params: any) => {
	return http.post(`/database/user/get`, params)
}

export const databaseUserRemarkAPI = (params: any) => {
	return http.post(`/database/user/remark`, params)
}
