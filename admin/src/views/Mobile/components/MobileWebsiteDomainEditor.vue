<script setup lang="ts">
import { updateMobileWebsiteDomainBindings, type MobileWebsite } from "@/api/modules/mobile"
import { mobileResourceMessages } from "@/i18n/locales/mobileResources"
import { reactive, ref, watch } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"

const props = defineProps<{ show: boolean; website: MobileWebsite | null }>()
const emit = defineEmits<{ "update:show": [value: boolean]; saved: [] }>()
const { t } = useI18n({ messages: mobileResourceMessages })
const dialog = useDialog()
const message = useMessage()
const saving = ref(false)
const saveError = ref("")
const form = reactive({
	primaryDomain: "",
	otherDomains: "",
	redirectDomainsToPrimary: false,
})

watch(
	() => [props.show, props.website] as const,
	() => {
		if (!props.show || !props.website) return
		form.primaryDomain = props.website.primaryDomain || ""
		form.otherDomains = (props.website.otherDomains || "").replaceAll(",", "\n")
		form.redirectDomainsToPrimary = Boolean(props.website.redirectDomainsToPrimary)
		saveError.value = ""
	},
	{ immediate: true },
)

function close(value = false) {
	if (!saving.value) emit("update:show", value)
}

function confirmSave() {
	if (!form.primaryDomain.trim()) {
		message.error(t("mobile.websiteDomainRequired"))
		return
	}
	dialog.warning({
		title: t("mobile.websiteDomainConfirmTitle"),
		content: t("mobile.websiteDomainConfirm"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: save,
	})
}

async function save() {
	if (!props.website) return
	saving.value = true
	saveError.value = ""
	try {
		await updateMobileWebsiteDomainBindings({
			websiteId: props.website.id,
			primaryDomain: form.primaryDomain.trim(),
			otherDomains: form.otherDomains,
			redirectDomainsToPrimary: form.redirectDomainsToPrimary,
		})
		message.success(t("mobile.websiteDomainSaved"))
		emit("saved")
		emit("update:show", false)
	} catch (error) {
		saveError.value = error instanceof Error ? error.message : t("mobile.websiteDomainSaveFailed")
	} finally {
		saving.value = false
	}
}
</script>

<template>
	<n-drawer :show="show" placement="bottom" height="min(620px, 82dvh)" :mask-closable="!saving" @update:show="close">
		<n-drawer-content :title="t('mobile.websiteDomainBindings')" :closable="!saving" body-content-style="padding: 16px;">
			<div class="space-y-4">
				<n-alert type="info" :bordered="false">{{ t("mobile.websiteDomainDnsHint") }}</n-alert>
				<n-alert v-if="saveError" type="error">{{ saveError }}</n-alert>
				<n-form label-placement="top">
					<n-form-item :label="t('mobile.websitePrimaryDomain')" required>
						<n-input v-model:value="form.primaryDomain" :placeholder="t('mobile.websitePrimaryDomainPlaceholder')" :disabled="saving" />
					</n-form-item>
					<n-form-item :label="t('mobile.websiteOtherDomains')">
						<n-input
							v-model:value="form.otherDomains"
							type="textarea"
							:autosize="{ minRows: 4, maxRows: 8 }"
							:placeholder="t('mobile.websiteOtherDomainsPlaceholder')"
							:disabled="saving"
						/>
					</n-form-item>
					<div class="flex items-center justify-between gap-4 rounded-xl bg-slate-50 p-3">
						<span class="text-sm text-slate-700">{{ t("mobile.websiteRedirectToPrimary") }}</span>
						<n-switch v-model:value="form.redirectDomainsToPrimary" :disabled="saving" />
					</div>
				</n-form>
				<n-button type="primary" block size="large" :loading="saving" @click="confirmSave">
					{{ t("mobile.websiteDomainSave") }}
				</n-button>
			</div>
		</n-drawer-content>
	</n-drawer>
</template>
