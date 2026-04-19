import { DeviceType } from "@/enums/app"

export interface ThemeConfigProp {
	panelName: string
	primary: string
	theme: string // dark | bright ｜ auto
	footer: boolean

	title: string
	logo: string
	logoWithText: string
	favicon: string
	themeColor: string
}

export interface GlobalState {
	isLoading: boolean
	loadingText: string
	csrfToken: string
	isLogin: boolean
	entrance: string
	language: string // zh | en | tw
	themeConfig: ThemeConfigProp
	isFullScreen: boolean
	openMenuTabs: boolean
	isOnRestart: boolean
	agreeLicense: boolean
	device: DeviceType.Desktop | DeviceType.Mobile
	hasNewVersion: boolean
	ignoreCaptcha: boolean
	lastFilePath: string
	currentDB: string
	currentRedisDB: string
	showEntranceWarn: boolean
	defaultNetwork: string

	isProductPro: boolean
	isIntl: boolean
	isTrial: boolean
	productProExpires: number
	licenseVerify: string

	errStatus: string
}
