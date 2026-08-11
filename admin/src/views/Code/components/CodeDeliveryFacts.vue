<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeDeliveryFact } from "@/api/interface/codeGit"
import Icon from "@/components/common/Icon.vue"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

const props = defineProps<{ facts?: CodeDeliveryFact[]; jobStatus?: string }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
// 后端 fact 的数量会随交付语义演进，这里只依赖「有没有」，
// 写死具体条数会让新增 fact 时整个区块静默消失。
const visible = computed(() => (props.facts?.length ?? 0) > 0)

const icon = (status: CodeDeliveryFact["status"]) => {
	if (status === "completed") return "mdi:check-circle"
	if (status === "partial") return "mdi:progress-alert"
	if (status === "skipped") return "mdi:minus-circle-outline"
	return "mdi:circle-outline"
}
const color = (status: CodeDeliveryFact["status"]) => {
	if (status === "completed") return "text-emerald-600"
	if (status === "partial") return "text-amber-600"
	return "text-slate-400"
}
const detail = (fact: CodeDeliveryFact) => {
	if (props.jobStatus === "conflict" && fact.status === "pending" && fact.key !== "snapshot") {
		return t("code.gitDeliveryFactStatus_blockedByConflict")
	}
	if (fact.total && fact.total > 1) {
		return t("code.gitDeliveryFactCount", { count: fact.count || 0, total: fact.total })
	}
	return t(`code.gitDeliveryFactStatus_${fact.status}`)
}
</script>

<template>
	<div v-if="visible" class="mt-3 grid grid-cols-2 gap-2">
		<div
			v-for="fact in facts"
			:key="fact.key"
			class="flex items-start gap-2 rounded-lg bg-black/5 p-2 dark:bg-white/5"
		>
			<Icon :name="icon(fact.status)" :size="16" :class="color(fact.status)" />
			<div class="min-w-0">
				<div class="text-xs font-medium">{{ t(`code.gitDeliveryFact_${fact.key}`) }}</div>
				<div class="mt-0.5 text-[11px] text-slate-500">{{ detail(fact) }}</div>
			</div>
		</div>
	</div>
</template>
