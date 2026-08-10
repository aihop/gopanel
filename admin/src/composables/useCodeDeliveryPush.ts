import { computed, ref, unref } from "vue"
import type { ComputedRef, Ref } from "vue"
import type { CodeDeliveryPushResult } from "@/api/interface/codeGit"

/**
 * 推送已交付提交的状态。桌面端与移动端走各自的 http 客户端，
 * 所以这里只收口状态与动作，由调用方注入 load / push。
 */
export interface UseCodeDeliveryPushOptions {
	sessionId: Ref<number | null> | ComputedRef<number | null>
	load: (sessionId: number) => Promise<CodeDeliveryPushResult>
	push: (sessionId: number) => Promise<CodeDeliveryPushResult>
}

export function useCodeDeliveryPush(options: UseCodeDeliveryPushOptions) {
	const result = ref<CodeDeliveryPushResult | null>(null)
	const loading = ref(false)
	const pushing = ref(false)
	const loadError = ref("")

	const repositories = computed(() => result.value?.repositories || [])
	const pushed = computed(() => result.value?.status === "pushed")
	const pendingCount = computed(() => repositories.value.filter(item => item.status !== "pushed").length)
	const destinations = computed(
		() => repositories.value.map(item => `${item.remote}/${item.branch}`).join(", ") || "-"
	)

	// 本地主仓未同步只是降级提示：交付提交已保留，推送远端不受影响。
	const localSyncPending = computed(() =>
		repositories.value.filter(item => !item.localSynced && item.localSyncError)
	)
	const localSyncCommands = computed(() =>
		localSyncPending.value
			.map(item => item.localSyncCommand || "")
			.filter(Boolean)
			.join("\n")
	)

	const visible = computed(() => loading.value || Boolean(loadError.value) || repositories.value.length > 0)

	async function loadStatus() {
		const sessionId = unref(options.sessionId)
		if (sessionId === null) {
			result.value = null
			return
		}
		loading.value = true
		loadError.value = ""
		try {
			result.value = await options.load(sessionId)
		} catch (error) {
			loadError.value = error instanceof Error ? error.message : String(error)
		} finally {
			loading.value = false
		}
	}

	/** 返回是否推送成功，由调用方决定如何提示。 */
	async function runPush(): Promise<boolean> {
		const sessionId = unref(options.sessionId)
		if (sessionId === null || !result.value?.available || pushing.value) return false
		pushing.value = true
		try {
			result.value = await options.push(sessionId)
			return true
		} catch (error) {
			await loadStatus()
			throw error
		} finally {
			pushing.value = false
		}
	}

	return {
		result,
		repositories,
		loading,
		pushing,
		loadError,
		pushed,
		pendingCount,
		destinations,
		localSyncPending,
		localSyncCommands,
		visible,
		loadStatus,
		runPush
	}
}
