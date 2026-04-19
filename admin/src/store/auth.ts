import type { User } from "@/types/api/auth"
import { userInfoAPI } from "@/api/modules/user"

import { acceptHMRUpdate, defineStore } from "pinia"

export const useAuthStore = defineStore("auth", {
	state: () => ({
		logged: false,
		auth: null as string | null,
		role: null as string | null,
		user: null as User | null
	}),
	actions: {
		setLogged(payload: { auth: string; user: User }) {
			this.logged = true
			this.auth = payload.auth
			this.user = payload.user
			this.role = payload.user.role
		},
		setLogout() {
			this.logged = false
			this.auth = null
			this.role = null
			this.user = null
		},
		async updateUser() {
			const user = await userInfoAPI()
			this.user = user
			this.role = user.role
		},
		getAuth(): string | null {
			return this.auth
		}
	},
	getters: {
		isLogged(state) {
			return state.logged
		},
		isAuthLeak(state) {
			return !state.user
		},
		userRole(state) {
			return state.role
		},
		userAvatar(state) {
			return state.user?.avatar
				? state.user.avatar
				: `${import.meta.env.BASE_URL !== "" ? import.meta.env.BASE_URL : "/"}images/avatar.svg`
		},
		nickName(state) {
			return state.user?.nickName ? state.user.nickName : "User"
		},
		userMenus(state): string[] {
			if (state.role === "ADMIN" || state.role === "SUPER") {
				return ["ALL"]
			}
			return state.user?.menus ? state.user.menus.split(",").filter(m => m.trim() !== "") : []
		}
	},
	persist: true
})

if (import.meta.hot) {
	import.meta.hot.accept(acceptHMRUpdate(useAuthStore, import.meta.hot))
}
