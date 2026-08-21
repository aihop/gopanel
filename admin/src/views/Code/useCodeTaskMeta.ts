import { useI18n } from "vue-i18n"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import { codeWorkspaceMessages } from "./codeWorkspaceMessages"

// 任务行的元信息格式化。工作台侧栏和开发面板都要渲染同一套
// 「时长 · token · git 状态 · diff」，逻辑放这里共用，避免两处各写一份走偏。
export function useCodeTaskMeta() {
	const { t, locale } = useI18n({ messages: codeWorkspaceMessages })

	const formatTaskDuration = (durationMs: number) => {
		const seconds = Math.max(1, Math.round(durationMs / 1000))
		if (seconds < 60) return t("code.taskDurationSeconds", { count: seconds })
		const minutes = Math.floor(seconds / 60)
		const remainingSeconds = seconds % 60
		if (minutes < 60)
			return remainingSeconds
				? t("code.taskDurationMinutesSeconds", { minutes, seconds: remainingSeconds })
				: t("code.taskDurationMinutes", { count: minutes })
		const hours = Math.floor(minutes / 60)
		const remainingMinutes = minutes % 60
		return remainingMinutes
			? t("code.taskDurationHoursMinutes", { hours, minutes: remainingMinutes })
			: t("code.taskDurationHours", { count: hours })
	}

	const formatTaskTokens = (tokens: number) => new Intl.NumberFormat(locale.value).format(tokens)

	// 百万级的 token 原样铺开就是一串没人会去数的数字。超过一万折成 "183.4万 / 1.8M"，
	// 精确值留给 title，行内只负责让人一眼有量级概念。
	const formatTaskTokensCompact = (tokens: number) =>
		tokens < 10000
			? formatTaskTokens(tokens)
			: new Intl.NumberFormat(locale.value, { notation: "compact", maximumFractionDigits: 1 }).format(tokens)

	const taskTokenStatus = (task: CodeTaskListItem) => {
		switch (task.summary.tokenUsageStatus) {
			case "recovered":
				return { label: t("code.taskTokensRecovered"), color: "text-emerald-600" }
			case "partial":
				return { label: t("code.taskTokensPartial"), color: "text-amber-600" }
			case "pending":
				return { label: t("code.taskTokensPending"), color: "text-slate-400" }
			case "unavailable":
				return { label: t("code.taskTokensUnavailable"), color: "text-slate-400" }
			default:
				return null
		}
	}

	const taskGitMeta = (task: CodeTaskListItem) =>
		({
			working: { icon: "mdi:source-branch", color: "text-slate-400" },
			committed: { icon: "mdi:source-commit", color: "text-blue-500" },
			merged: { icon: "mdi:source-merge", color: "text-emerald-600" },
			pushed: { icon: "mdi:cloud-check-outline", color: "text-emerald-600" },
			push_failed: { icon: "mdi:cloud-alert-outline", color: "text-red-500" },
			conflict: { icon: "mdi:source-branch-alert", color: "text-red-500" },
		})[task.summary.gitStatus || ""]

	const taskDeliveryMeta = (task: CodeTaskListItem) => {
		const status = task.summary.deliveryStatus
		if (!status) return null
		const appearances = {
			queued: { icon: "mdi:clock-outline", color: "text-amber-600" },
			running: { icon: "mdi:cloud-sync-outline", color: "text-blue-500" },
			completed: { icon: "mdi:cloud-check-outline", color: "text-emerald-600" },
			partial: { icon: "mdi:alert-circle-outline", color: "text-amber-600" },
			conflict: { icon: "mdi:source-branch-alert", color: "text-red-500" },
			failed: { icon: "mdi:cloud-alert-outline", color: "text-red-500" },
		}
		return {
			...appearances[status],
			label: t(`code.deliveryStatus_${status}`, {
				position: task.summary.deliveryQueuePosition,
				progress: task.summary.deliveryProgress,
			}),
		}
	}

	// 出错信息以前只在 title 里。让人 hover 才发现任务挂了是设计事故，所以提到行内。
	const taskError = (task: CodeTaskListItem) => task.summary.deliveryError || task.summary.gitError || ""

	// 会话阶段比任务 status 细一档。只在「还在动」的阶段显示；
	// completed / idle 不显示 —— 那些 status 徽标已经说清楚了，重复只会占地方。
	const meaningfulStages = ["executing", "awaiting_approval", "instruction_queued", "syncing_base"]
	const taskStage = (task: CodeTaskListItem) => {
		const stage = task.summary.stage || ""
		return meaningfulStages.includes(stage) ? t(`code.taskStage_${stage}`) : ""
	}

	const taskTooltip = (task: CodeTaskListItem) =>
		[
			task.summary.deliveryStatus
				? t(`code.deliveryStatus_${task.summary.deliveryStatus}`, {
						position: task.summary.deliveryQueuePosition,
						progress: task.summary.deliveryProgress,
					})
				: "",
			task.summary.deliveryStage ? t(`code.deliveryStage_${task.summary.deliveryStage}`) : "",
			task.summary.deliveryError,
			task.summary.gitStatus ? t(`code.taskGitStatus_${task.summary.gitStatus}`) : "",
			task.summary.gitError,
			task.summary.executor,
			task.summary.model,
			task.summary.totalTokens ? t("code.taskTokens", { count: formatTaskTokens(task.summary.totalTokens) }) : "",
			taskTokenStatus(task)?.label,
			task.summary.branch,
			new Date(task.createdAt).toLocaleString(),
		]
			.filter(Boolean)
			.join(" · ")

	return {
		t,
		formatTaskDuration,
		formatTaskTokens,
		formatTaskTokensCompact,
		taskTokenStatus,
		taskGitMeta,
		taskDeliveryMeta,
		taskTooltip,
		taskError,
		taskStage,
	}
}
