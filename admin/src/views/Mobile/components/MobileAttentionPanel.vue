<script setup lang="ts">
import { executeMobileAttentionAction, getMobileAttention } from "@/api/modules/mobileAttention"
import type { MobileAttentionAction, MobileAttentionItem } from "@/api/interface/mobileControlPlane"
import Icon from "@/components/common/Icon.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import { onMounted, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"

const emit = defineEmits<{ openSession: [sessionId: number] }>()
const { t } = useI18n({ messages: mobileMessages })
const dialog = useDialog()
const message = useMessage()
const loading = ref(false)
const loadError = ref("")
const actionKey = ref("")
const items = ref<MobileAttentionItem[]>([])

function itemTitle(item: MobileAttentionItem) {
	return t(`mobile.attentionType_${item.type}`)
}

function actionLabel(action: MobileAttentionAction) {
	return t(`mobile.attentionAction_${action.type}`)
}

async function load() {
	loading.value = true
	try {
		items.value = (await getMobileAttention()).items
		loadError.value = ""
	} catch (error) {
		loadError.value = error instanceof Error ? error.message : t("mobile.attentionLoadFailed")
	} finally {
		loading.value = false
	}
}

function run(item: MobileAttentionItem, action: MobileAttentionAction) {
	if (action.type === "open_session") {
		emit("openSession", item.sessionId)
		return
	}
	const execute = async () => {
		actionKey.value = `${item.id}:${action.type}`
		try {
			await executeMobileAttentionAction(action)
			message.success(t("mobile.attentionActionSuccess"))
			await load()
		} catch (error) {
			// 错误提示由请求拦截器统一处理
		} finally {
			actionKey.value = ""
		}
	}
	if (!action.requiresConfirmation) {
		void execute()
		return
	}
	dialog.warning({
		title: actionLabel(action),
		content: t("mobile.attentionActionConfirm", { action: actionLabel(action) }),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: execute
	})
}

onMounted(load)
</script>

<template>
	<section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
		<div class="mb-3 flex items-center justify-between gap-3">
			<div>
				<div class="flex items-center gap-2 font-bold text-slate-900">
					<Icon name="mdi:alert-circle-outline" class="text-rose-500" />
					{{ t("mobile.attentionTitle") }}
				</div>
				<div class="mt-1 text-xs text-slate-500">{{ t("mobile.attentionHint") }}</div>
			</div>
			<n-button circle quaternary size="small" :loading="loading" @click="load">
				<template #icon><Icon name="mdi:refresh" /></template>
			</n-button>
		</div>
		<n-alert v-if="loadError" type="error" :title="t('mobile.attentionLoadFailed')">
			<div class="flex items-center justify-between gap-3">
				<span>{{ loadError }}</span>
				<n-button text type="primary" @click="load">{{ t("mobile.retry") }}</n-button>
			</div>
		</n-alert>
		<n-spin v-else :show="loading">
			<n-empty v-if="!loading && !items.length" size="small" :description="t('mobile.attentionEmpty')" />
			<div v-else class="space-y-3">
				<article v-for="item in items" :key="item.id" class="rounded-xl bg-slate-50 p-3">
					<div class="font-semibold text-slate-900">{{ itemTitle(item) }}</div>
					<div v-if="item.summary" class="mt-1 whitespace-pre-wrap text-sm text-slate-600">
						{{ item.summary }}
					</div>
					<div class="mt-3 flex flex-wrap gap-2">
						<n-button
							v-for="action in item.actions"
							:key="action.type"
							size="small"
							:type="action.type === 'reject' ? 'error' : 'primary'"
							secondary
							:loading="actionKey === `${item.id}:${action.type}`"
							@click="run(item, action)"
						>
							{{ actionLabel(action) }}
						</n-button>
					</div>
				</article>
			</div>
		</n-spin>
	</section>
</template>
