/// <reference types="vite/client" />
/// <reference types="vite-svg-loader" />

declare const __APP_ENV__: any
declare const __APP_BRAND__: string

interface ImportMetaEnv {
  readonly VITE_APP_TITLE: string
  readonly VITE_APP_BRAND: string
  // more env variables...
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module "*.vue" {
	import type { DefineComponent } from "vue"

	const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, any>
	export default component
}

declare module "*.svg" {
	import type { DefineComponent } from "vue"

	const component: DefineComponent
	export default component
}
