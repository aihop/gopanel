<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { getCodeDeliveryPushStatus, pushCodeSessionDelivery } from "@/api/modules/codeGit"
import type { CodeDeliveryPushResult } from "@/api/interface/codeGit"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

const props = defineProps<{ sessionId: number; refreshKey: number }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const dialog = useDialog()
const message = useMessage()
const loading = ref(false)
const pushing = ref(false)
const loadError = ref("")
const result = ref<CodeDeliveryPushResult | null>(null)

const pendingCount = computed(() => result.value?.repositories.filter(repository => repository.status !== "pushed").length || 0)
const destinations = computed(() => result.value?.repositories.map(repository => `${repository.remote}/${repository.branch}`).join(", ") || "-")

const loadStatus = async () => {
	loading.value = true
	loadError.value = ""
	try {
		result.value = (await getCodeDeliveryPushStatus(props.sessionId)).data
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("code.gitPushStatusFailed")
	} finally {
		loading.value = false
	}
}

const pushDelivery = () => {
	if (!result.value?.available || pushing.value) return
	dialog.warning({
		title: t("code.gitPushTitle"),
		content: t("code.gitPushConfirm", { count: pendingCount.value, destinations: destinations.value }),
		positiveText: t("code.gitPush"),
		negativeText: t("code.gitCancel"),
		onPositiveClick: async () => {
			pushing.value = true
			try {
				result.value = (await pushCodeSessionDelivery(props.sessionId)).data
				message.success(t("code.gitPushSuccess"))
			} catch (error) {
				message.error(error instanceof Error ? error.message : t("code.gitPushFailed"))
				await loadStatus()
			} finally {
				pushing.value = false
			}
		}
	})
}

watch(() => [props.sessionId, props.refreshKey], () => void loadStatus())
onMounted(() => void loadStatus())
</script>

<template>
	<div v-if="loading || loadError || result?.repositories.length" class="border-b border-slate-200 p-3">
		<n-spin :show="loading">
			<n-alert v-if="loadError" type="error" :title="t('code.gitPushStatusFailed')">
				<div class="space-y-2">
					<p>{{ loadError }}</p>
					<n-button size="tiny" @click="loadStatus">{{ t("code.gitRetry") }}</n-button>
				</div>
			</n-alert>
			<div v-else-if="result?.status === 'pushed'" class="text-xs text-emerald-600">
				{{ t("code.gitPushCompleted", { count: result.repositories.length }) }}
			</div>
			<div v-else-if="result" class="space-y-2">
				<p class="text-xs text-slate-500">
					{{ result.available ? t("code.gitPushReady", { count: pendingCount }) : t("code.gitPushUnavailable") }}
				</p>
				<p v-if="result.available" class="truncate text-[11px] text-slate-400" :title="destinations">{{ destinations }}</p>
				<n-button v-if="result.available" size="small" type="warning" secondary block :loading="pushing" @click="pushDelivery">
					{{ t("code.gitPush") }}
				</n-button>
			</div>
		</n-spin>
	</div>
</template>
