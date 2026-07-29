<script setup lang="ts">
import Logo from "@/layouts/common/Logo.vue"
import { exchangeMobilePairing } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"
import { useI18n } from "vue-i18n"
import { onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"

const route = useRoute()
const router = useRouter()
const { t } = useI18n({ messages: mobileMessages })
const status = ref<"loading" | "error">("loading")
const errorMessage = ref("")

async function authorize() {
	const code = typeof route.query.code === "string" ? route.query.code.trim() : ""
	if (!code) {
		status.value = "error"
		errorMessage.value = t("mobile.missingCode")
		return
	}
	status.value = "loading"
	try {
		await exchangeMobilePairing(code, navigator.platform || "手机浏览器")
		await router.replace("/mobile")
	} catch (error) {
		status.value = "error"
		errorMessage.value = error instanceof Error ? error.message : t("mobile.authorizeFailed")
	}
}

onMounted(authorize)
</script>

<template>
	<div class="flex min-h-dvh w-full items-center justify-center bg-slate-950 px-5 text-white">
		<div class="w-full max-w-sm rounded-[28px] border border-white/10 bg-white/10 p-8 text-center shadow-2xl backdrop-blur">
			<Logo :mini="true" class="mx-auto mb-5" />
			<n-spin v-if="status === 'loading'" size="large" />
			<div v-if="status === 'loading'" class="mt-5 text-sm text-slate-300">{{ t("mobile.authorizing") }}</div>
			<n-alert v-else type="error" :title="t('mobile.authorizeFailed')" class="text-left">{{ errorMessage }}</n-alert>
			<n-button v-if="status === 'error'" type="primary" block class="mt-5" @click="authorize">{{ t("mobile.retry") }}</n-button>
		</div>
	</div>
</template>
