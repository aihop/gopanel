export function useFormat() {
	const internalInstance = getCurrentInstance()
	if (!internalInstance) {
		throw new Error("useFormatDate() must be called within a setup function")
	}

	return {
		dd: internalInstance.appContext.config.globalProperties.$dd,
		nn: internalInstance.appContext.config.globalProperties.$nn
	}
}
