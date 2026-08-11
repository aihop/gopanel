<script setup lang="ts">
import { computed, ref } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeDeliveryFact } from "@/api/interface/codeGit"
import Icon from "@/components/common/Icon.vue"
import { codeDeliveryFactsMessages } from "../codeDeliveryFactsMessages"

const props = defineProps<{ facts?: CodeDeliveryFact[]; jobStatus?: string }>()
const { t } = useI18n({ messages: codeDeliveryFactsMessages })
const expanded = ref(false)
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
	if (props.jobStatus === "conflict" && fact.key === "merge" && fact.status !== "completed") {
		return t("code.gitDeliveryFactStatus_mergeConflict")
	}
	if (props.jobStatus === "conflict" && fact.status === "pending" && fact.key !== "snapshot") {
		return t("code.gitDeliveryFactStatus_stoppedAfterConflict")
	}
	if (fact.total && fact.total > 1) {
		return t("code.gitDeliveryFactCount", { count: fact.count || 0, total: fact.total })
	}
	return t(`code.gitDeliveryFactStatus_${fact.status}`)
}
</script>

<template>
	<div v-if="visible" class="mt-2">
		<n-button text size="tiny" @click="expanded = !expanded">
			<template #icon>
				<Icon :name="expanded ? 'mdi:chevron-up' : 'mdi:chevron-down'" :size="14" />
			</template>
			{{ t(expanded ? "code.gitDeliveryDetailsHide" : "code.gitDeliveryDetailsShow") }}
		</n-button>
		<div v-if="expanded" class="mt-2 divide-y divide-slate-200/70 rounded-lg bg-black/5 px-2 dark:divide-white/10 dark:bg-white/5">
			<div
				v-for="fact in facts"
				:key="fact.key"
				class="flex items-center justify-between gap-3 py-1.5 text-xs"
			>
				<div class="flex min-w-0 items-center gap-1.5">
					<Icon :name="icon(fact.status)" :size="14" :class="color(fact.status)" />
					<span class="font-medium">{{ t(`code.gitDeliveryFact_${fact.key}`) }}</span>
				</div>
				<span class="shrink-0 text-[11px] text-slate-500">{{ detail(fact) }}</span>
			</div>
		</div>
	</div>
</template>
