import type { GlobalState, ThemeConfigProp } from "../interface"
import piniaPersistConfig from "@/config/pinia-persist"
import { DeviceType } from "@/enums/app"
import { t } from "@/i18n"
import { defineStore } from "pinia"

const GlobalStore = defineStore("GlobalState", {
	state: (): GlobalState => ({
		isLoading: false,
		loadingText: "",
		isLogin: false,
		entrance: "",
		language: "",
		themeConfig: {
			panelName: "",
			primary: "#005eeb",
			theme: "auto",
			footer: true,
			themeColor: "",
			title: "",
			logo: "",
			logoWithText: "",
			favicon: ""
		},
		openMenuTabs: false,
		isFullScreen: false,
		isOnRestart: false,
		agreeLicense: false,
		device: DeviceType.Desktop,
		hasNewVersion: false,
		ignoreCaptcha: true,
		lastFilePath: "",
		currentDB: "",
		currentRedisDB: "",
		showEntranceWarn: true,
		defaultNetwork: "all",
		csrfToken: "",

		isProductPro: false,
		isIntl: false,
		isTrial: false,
		productProExpires: 0,
		licenseVerify: "",

		errStatus: ""
	}),
	getters: {
		isDarkTheme: state =>
			state.themeConfig.theme === "dark" ||
			(state.themeConfig.theme === "auto" && window.matchMedia("(prefers-color-scheme: dark)").matches),
		isDarkGoldTheme: state => state.themeConfig.primary === "#F0BE96" && state.isProductPro,
		docsUrl: state => (state.isIntl ? "https://gopanel.cn/docs" : "https://gopanel.com/docs")
	},
	actions: {
		setOpenMenuTabs(openMenuTabs: boolean) {
			this.openMenuTabs = openMenuTabs
		},
		setScreenFull() {
			this.isFullScreen = !this.isFullScreen
		},
		setLogStatus(login: boolean) {
			this.isLogin = login
		},
		setEntrance(entrance: string) {
			this.entrance = entrance
		},
		setGlobalLoading(loading: boolean) {
			this.isLoading = loading
		},
		setLoadingText(text: string) {
			this.loadingText = t(`commons.loadingText.${text}`)
		},
		setCsrfToken(token: string) {
			this.csrfToken = token
		},
		updateLanguage(language: any) {
			this.language = language
			localStorage.setItem("lang", language)
		},
		setThemeConfig(themeConfig: ThemeConfigProp) {
			this.themeConfig = themeConfig
		},
		setAgreeLicense(agree: boolean) {
			this.agreeLicense = agree
		},
		toggleDevice(value: DeviceType) {
			this.device = value
		},
		isMobile() {
			return this.device === DeviceType.Mobile
		},
		setLastFilePath(path: string) {
			this.lastFilePath = path
		},
		setCurrentDB(name: string) {
			this.currentDB = name
		},
		setCurrentRedisDB(name: string) {
			this.currentRedisDB = name
		},
		setShowEntranceWarn(show: boolean) {
			this.showEntranceWarn = show
		},
		setDefaultNetwork(net: string) {
			this.defaultNetwork = net
		}
	},
	persist: piniaPersistConfig("GlobalState")
})

export default GlobalStore
