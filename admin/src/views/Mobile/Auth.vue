<script setup lang="ts">
import { exchangeMobilePairing, loginMobileDevice } from "@/api/modules/mobile"
import LoginCaptcha from "@/components/auth/LoginCaptcha.vue"
import { mobileMessages } from "@/i18n/locales/mobile"
import Logo from "@/layouts/common/Logo.vue"
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { useRoute, useRouter } from "vue-router"

const route = useRoute()
const router = useRouter()
const { t } = useI18n({ messages: mobileMessages })
const message = useMessage()
const status = ref<"login" | "loading" | "error">("login")
const errorMessage = ref("")
const email = ref("")
const password = ref("")
const captchaRequired = ref(false)
const captchaToken = ref("")
const captchaRef = ref<{ show: () => void } | null>(null)
const pendingSubmitAfterCaptcha = ref(false)
const canSubmit = computed(() => Boolean(email.value.trim() && password.value && status.value !== "loading"))
const deviceName = navigator.platform || navigator.userAgent || t("mobile.defaultDeviceName")

async function authorizePairing() {
	const code = typeof route.query.code === "string" ? route.query.code.trim() : ""
	if (!code) return
	status.value = "loading"
	try {
		await exchangeMobilePairing(code, deviceName)
		await router.replace("/mobile")
	} catch (error) {
		status.value = "error"
		errorMessage.value = error instanceof Error ? error.message : t("mobile.authorizeFailed")
	}
}

async function signIn() {
	if (!canSubmit.value) return
	if (captchaRequired.value && !captchaToken.value) {
		pendingSubmitAfterCaptcha.value = true
		captchaRef.value?.show()
		return
	}
	status.value = "loading"
	errorMessage.value = ""
	try {
		await loginMobileDevice({
			email: email.value.trim(),
			password: password.value,
			captchaToken: captchaToken.value || undefined,
			deviceName
		})
		await router.replace("/mobile")
	} catch (error) {
		status.value = "login"
		captchaRequired.value = true
		captchaToken.value = ""
		pendingSubmitAfterCaptcha.value = false
		errorMessage.value = error instanceof Error ? error.message : t("mobile.loginFailed")
		message.error(errorMessage.value)
	}
}

function handleCaptchaSuccess(token: string) {
	captchaToken.value = token
	if (pendingSubmitAfterCaptcha.value) {
		pendingSubmitAfterCaptcha.value = false
		void signIn()
	}
}

function showLogin() {
	status.value = "login"
	errorMessage.value = ""
	void router.replace({ path: "/mobile/auth" })
}

onMounted(authorizePairing)
</script>

<template>
	<div class="flex min-h-dvh w-full items-center justify-center bg-slate-100 px-5 py-8 text-slate-900">
		<div class="w-full max-w-sm rounded-[28px] border border-slate-200 bg-white p-7 shadow-xl">
			<Logo :dark="false" max-height="34px" class="mb-7" />
			<div v-if="status === 'loading'" class="py-12 text-center">
				<n-spin size="large" />
				<div class="mt-5 text-sm text-slate-500">{{ route.query.code ? t("mobile.authorizing") : t("mobile.loggingIn") }}</div>
			</div>
			<div v-else-if="status === 'error'">
				<n-alert type="error" :title="t('mobile.authorizeFailed')">{{ errorMessage }}</n-alert>
				<n-button type="primary" block class="mt-5" @click="showLogin">{{ t("mobile.accountLogin") }}</n-button>
			</div>
			<div v-else>
				<h1 class="text-2xl font-bold">{{ t("mobile.accountLogin") }}</h1>
				<p class="mt-2 text-sm leading-6 text-slate-500">{{ t("mobile.loginHint") }}</p>
				<n-form class="mt-6" label-placement="top" @submit.prevent="signIn">
					<n-form-item :label="t('mobile.loginAccount')">
						<n-input v-model:value="email" size="large" autocomplete="username" :placeholder="t('mobile.loginAccountPlaceholder')" @keydown.enter="signIn" />
					</n-form-item>
					<n-form-item :label="t('mobile.loginPassword')">
						<n-input v-model:value="password" size="large" type="password" show-password-on="click" autocomplete="current-password" :placeholder="t('mobile.loginPasswordPlaceholder')" @keydown.enter="signIn" />
					</n-form-item>
					<n-alert v-if="captchaRequired" type="warning" :show-icon="false" class="mb-5">{{ t("mobile.captchaRequired") }}</n-alert>
					<n-button attr-type="submit" type="primary" size="large" block :disabled="!canSubmit">{{ t("mobile.login") }}</n-button>
				</n-form>
				<div class="mt-5 text-center text-xs leading-5 text-slate-400">{{ t("mobile.loginSecurityHint") }}</div>
			</div>
		</div>
		<LoginCaptcha ref="captchaRef" @success="handleCaptchaSuccess" />
	</div>
</template>
