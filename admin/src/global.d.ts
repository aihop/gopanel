export {}

import type { MessageApi } from "naive-ui"
import type { Router } from "vue-router"

declare global {
	interface Document {
		startViewTransition?: (callback: () => Promise<void> | void) => ViewTransition
	}

	interface ViewTransition {
		readonly ready: Promise<undefined>
		readonly finished: Promise<undefined>
		readonly updateCallbackDone: Promise<undefined>
		skipTransition: () => void
	}

	interface CSSStyleDeclaration {
		viewTransitionName: string
	}

	interface Window {
		$message: MessageApi
		$bus: {
			on: (event: string, handler: (...args: any[]) => void) => void
			off: (event: string, handler?: (...args: any[]) => void) => void
			emit: (event: string, ...args: any[]) => void
		}
	}
}

declare module "@vue/runtime-core" {
	interface ComponentCustomProperties {
		$t: (key: string, ...args: any[]) => string
		$gettext: (message: string, params?: Record<string, any>) => string
		$router: Router
	}
}
