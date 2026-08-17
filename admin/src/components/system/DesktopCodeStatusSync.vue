<script setup lang="ts">
import { getCodeDesktopSummary } from "@/api/modules/code"
import type { CodeDesktopSummary } from "@/api/interface/codeDesktop"
import { useEventListener, useIntervalFn } from "@vueuse/core"
import { onBeforeUnmount, onMounted } from "vue"

interface WailsRuntime {
	EventsEmit?: (eventName: string, ...data: unknown[]) => void
}

const emptySummary: CodeDesktopSummary = { attention: 0, running: 0, queued: 0 }
let active = false

function desktopRuntime() {
	return (window as Window & { runtime?: WailsRuntime }).runtime
}

function emitSummary(summary: CodeDesktopSummary) {
	desktopRuntime()?.EventsEmit?.("gopanel:code-summary", summary)
}

async function syncSummary() {
	if (!desktopRuntime()?.EventsEmit) return
	try {
		const response = await getCodeDesktopSummary()
		if (active) emitSummary(response.data)
	} catch {
		return
	}
}

const { pause, resume } = useIntervalFn(syncSummary, 15000, { immediate: false })

useEventListener(document, "visibilitychange", () => {
	if (document.visibilityState === "visible") void syncSummary()
})

onMounted(() => {
	if (!desktopRuntime()?.EventsEmit) return
	active = true
	void syncSummary()
	resume()
})

onBeforeUnmount(() => {
	active = false
	pause()
	emitSummary(emptySummary)
})
</script>

<template></template>
