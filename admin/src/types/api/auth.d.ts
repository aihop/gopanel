export type Role = "all" | "ADMIN" | "SUPER" | "SUB_ADMIN"
export type Roles = role | role[]

export interface RouteMetaAuth {
	checkAuth?: boolean
	authRedirect?: string
	auth?: boolean
	roles?: Roles
}

export interface User {
	id: number
	isdCode: string
	email: string
	mobile: string
	nickName: string
	avatar: string
	role: Role
	fileBaseDir?: string
	menus?: string
}

export interface AuthData {
	xAuth: string
	userInfo: User
}
