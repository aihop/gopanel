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

export const getDBManagerTableListAPI = (data: { serverId: number; databaseName: string; page: number; limit: number; keyword?: string; sortField?: string; sortOrder?: string }) => {
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

export const exportDBManagerTableAPI = (data: { serverId: number; databaseName: string; tableName: string; format: string }) => {
	return http.post(`/database/manager/export`, data, { responseType: 'text' })
}

export const importDBManagerTableAPI = (data: { serverId: number; databaseName: string; tableName: string; format: string; content: string }) => {
	return http.post(`/database/manager/import`, data)
}

export const uploadDBManagerImportAPI = (data: { serverId: number; databaseName: string; tableName: string; format: string; file: File }) => {
	const formData = new FormData()
	formData.append('serverId', String(data.serverId))
	formData.append('databaseName', data.databaseName)
	formData.append('tableName', data.tableName)
	formData.append('format', data.format)
	formData.append('file', data.file)
	return http.post(`/database/manager/upload`, formData, {
		headers: { 'Content-Type': 'multipart/form-data' },
		timeout: 300000 // 5 min for large files
	})
}

export const databaseUserGetAPI = (params: any) => {
	return http.post(`/database/user/get`, params)
}

export const databaseUserRemarkAPI = (params: any) => {
	return http.post(`/database/user/remark`, params)
}

// === 新增 DB Manager API (P0) ===

export const createDBManagerDatabaseAPI = (data: {
	serverId: number
	databaseName: string
	charset?: string
	collation?: string
}) => {
	return http.post(`/database/manager/create-database`, data)
}

export const dropDBManagerDatabaseAPI = (data: {
	serverId: number
	databaseName: string
}) => {
	return http.post(`/database/manager/drop-database`, data)
}

export const getDBManagerTableInfoAPI = (data: {
	serverId: number
	databaseName: string
	tableName: string
}) => {
	return http.post<any>(`/database/manager/table-info`, data)
}

export const getDBManagerDatabaseInfoAPI = (data: {
	serverId: number
	databaseName: string
}) => {
	return http.post<any>(`/database/manager/database-info`, data)
}

export const createDBManagerTableAPI = (data: {
	serverId: number
	databaseName: string
	tableName: string
	engine?: string
	charset?: string
	collation?: string
	comment?: string
	columns: Array<{
		name: string
		type: string
		length?: string
		nullable: boolean
		defaultValue?: string
		autoIncrement: boolean
		comment?: string
		isPrimary: boolean
	}>
}) => {
	return http.post(`/database/manager/create-table`, data)
}
