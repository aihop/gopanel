import { computed, unref } from "vue"
import type { ComputedRef, Ref } from "vue"
import type { CodeDeliveryJob } from "@/api/interface/codeGit"

/**
 * 交付按钮在桌面端和移动端曾各写一套状态判定，改一次交付语义要改两处。
 * 这里收口共用的判定，两端只负责把 phase 映射到各自的文案与图标。
 */
export type CodeDeliveryPhase =
	| "idle" // 可以发起交付
	| "queued" // 已排队
	| "quality_check" // 正在跑质量门禁
	| "running" // 交付执行中
	| "needs_save" // 有未提交改动，需先保存
	| "deliverable" // 已交付过，但仍有可交付的新提交
	| "delivered" // 已交付且没有待交付内容
	| "retry" // 上次交付失败，可重试

type MaybeRef<T> = Ref<T> | ComputedRef<T> | T

export interface UseCodeDeliveryOptions {
	job: MaybeRef<CodeDeliveryJob | null | undefined>
	/** 会话是否具备隔离 Worktree，没有就不显示交付入口。 */
	available: MaybeRef<boolean>
	/** 各端额外的「正在交付」信号，例如乐观标记或会话状态。 */
	extraDelivering?: MaybeRef<boolean>
	/** 各端额外的「已交付」信号，例如会话状态。 */
	extraDelivered?: MaybeRef<boolean>
	/** 工作区里尚未保存的改动，移动端会据此把按钮切成「保存」。 */
	hasLocalChanges?: MaybeRef<boolean>
	/**
	 * 待交付提交是否只在交付完成后才算数。
	 * 桌面端沿用 completed 才判定，移动端不要求，保持两端既有行为。
	 */
	pendingRequiresCompleted?: boolean
}

const failedStatuses = ["failed", "conflict", "partial"]

export function useCodeDelivery(options: UseCodeDeliveryOptions) {
	const job = computed(() => unref(options.job) ?? null)
	const available = computed(() => Boolean(unref(options.available)))
	const status = computed(() => job.value?.status || "")

	const completed = computed(() => status.value === "completed")
	const delivering = computed(
		() => ["queued", "running"].includes(status.value) || Boolean(unref(options.extraDelivering))
	)
	const delivered = computed(() => completed.value || Boolean(unref(options.extraDelivered)))

	const pendingGate = computed(() => (options.pendingRequiresCompleted ? completed.value : true))
	const hasPendingCommits = computed(() => pendingGate.value && job.value?.hasPendingCommits === true)
	const hasUncommittedChanges = computed(() => pendingGate.value && job.value?.hasUncommittedChanges === true)
	const hasLocalChanges = computed(() => Boolean(unref(options.hasLocalChanges)) || hasUncommittedChanges.value)

	/** 已交付之后是否还有能再交付的内容。 */
	const canDeliverPending = computed(() => hasPendingCommits.value && !hasUncommittedChanges.value)
	const canDeliver = computed(() => !completed.value || canDeliverPending.value)
	const failed = computed(() => failedStatuses.includes(status.value))

	const phase = computed<CodeDeliveryPhase>(() => {
		if (status.value === "queued") return "queued"
		if (delivering.value && job.value?.stage === "quality_check") return "quality_check"
		if (delivering.value) return "running"
		if (hasLocalChanges.value) return "needs_save"
		if (canDeliverPending.value) return "deliverable"
		if (delivered.value) return "delivered"
		if (failed.value) return "retry"
		return "idle"
	})

	const busy = computed(() => phase.value === "queued" || phase.value === "quality_check" || phase.value === "running")

	return {
		available,
		phase,
		busy,
		delivering,
		delivered,
		completed,
		failed,
		hasPendingCommits,
		hasUncommittedChanges,
		hasLocalChanges,
		canDeliver,
		canDeliverPending,
		progress: computed(() => job.value?.progress ?? 0)
	}
}

/** 按 phase 给出按钮配色，两端一致。 */
export function codeDeliveryPhaseType(phase: CodeDeliveryPhase): "success" | "info" | "primary" {
	if (phase === "delivered") return "success"
	if (phase === "queued" || phase === "quality_check" || phase === "running") return "info"
	return "primary"
}

/** 按 phase 给出按钮图标，两端一致。 */
export function codeDeliveryPhaseIcon(phase: CodeDeliveryPhase): string {
	if (phase === "needs_save") return "mdi:content-save-outline"
	if (phase === "delivered") return "mdi:cloud-check-outline"
	if (phase === "queued" || phase === "quality_check" || phase === "running") return "mdi:cloud-sync-outline"
	return "mdi:source-merge"
}
