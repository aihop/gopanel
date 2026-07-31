<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import { issueMobilePairing, getMobileDevices, revokeMobileDevice, type MobileDevice } from "@/api/modules/mobile"
import { mobileMessages } from "@/i18n/locales/mobile"
import { useI18n } from "vue-i18n"
import { computed, ref, watch } from "vue"
import { useMessage } from "naive-ui"
import useClipboard from "vue-clipboard3"

const { t } = useI18n({ messages: mobileMessages })
const { toClipboard } = useClipboard()
const message = useMessage()
const showModal = ref(false)
const loading = ref(false)
const pairingCode = ref("")
const desktopMobileUrl = (window as Window & { __GOPANEL_DESKTOP_MOBILE_URL__?: string }).__GOPANEL_DESKTOP_MOBILE_URL__
const publicUrl = ref(desktopMobileUrl || new URL("/mobile", window.location.origin).toString())
const deviceTtlDays = ref(30)
const devices = ref<MobileDevice[]>([])
let pairingRequestId = 0

const durationOptions = computed(() => [1, 7, 30, 90, 365].map(days => ({
	label: days === 365 ? t("mobile.durationYear") : t("mobile.durationDays", { days }),
	value: days
})))

const qrCodeUrl = computed(() => {
	if (!pairingCode.value) return ""
	try {
		const url = new URL("/mobile/auth", normalizedPublicUrl())
		url.searchParams.set("code", pairingCode.value)
		return url.toString()
	} catch {
		return ""
	}
})
const isHttpAddress = computed(() => /^http:\/\//i.test(qrCodeUrl.value))

function normalizedPublicUrl() {
	let value = publicUrl.value.trim().replace(/\/+$/, "")
	if (!/^https?:\/\//i.test(value)) value = `http://${value}`
	const url = new URL(value)
	url.pathname = "/mobile"
	url.search = ""
	url.hash = ""
	return url.toString()
}

function normalizePublicUrl() {
	try {
		publicUrl.value = normalizedPublicUrl()
	} catch {}
}

async function copyPublicUrl() {
	try {
		const url = normalizedPublicUrl()
		publicUrl.value = url
		await toClipboard(url)
		message.success(t("mobile.publicUrlCopied"))
	} catch {
		message.error(t("mobile.copyFailed"))
	}
}

async function loadDevices() {
	try {
		const result = await getMobileDevices()
		devices.value = result.items || []
	} catch (error) {
		devices.value = []
		message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
	}
}

async function generatePairing() {
	const requestId = ++pairingRequestId
	const requestedTTLDays = deviceTtlDays.value
	loading.value = true
	try {
		const result = await issueMobilePairing(requestedTTLDays)
		if (requestId !== pairingRequestId) return
		pairingCode.value = result.code
	} catch (error) {
		if (requestId !== pairingRequestId) return
		message.error(error instanceof Error ? error.message : t("mobile.pairFailed"))
	} finally {
		if (requestId === pairingRequestId) loading.value = false
	}
}

async function revokeDevice(device: MobileDevice) {
	try {
		await revokeMobileDevice(device.id)
		message.success(t("mobile.revokeSuccess"))
		await loadDevices()
	} catch (error) {
		message.error(error instanceof Error ? error.message : t("mobile.loadFailed"))
	}
}

function formatDeviceExpiry(value: string) {
	return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value))
}

watch(showModal, opened => {
	if (!opened) return
	void Promise.all([generatePairing(), loadDevices()])
})
</script>

<template>
	<button class="flex h-5 w-5 items-center justify-center border-0 bg-transparent p-0" :aria-label="t('mobile.title')" @click="showModal = true">
		<Icon name="ion:phone-portrait-outline" :size="20" />
	</button>

	<n-modal v-model:show="showModal" preset="card" style="width: 420px" :title="t('mobile.title')">
		<div class="flex flex-col gap-4">
			<n-alert v-if="isHttpAddress" type="warning" :show-icon="false">{{ t("mobile.httpWarning") }}</n-alert>
			<div>
				<div class="mb-2 text-sm font-medium">{{ t("mobile.publicUrl") }}</div>
				<n-input v-model:value="publicUrl" :placeholder="t('mobile.publicUrlHint')" @blur="normalizePublicUrl">
					<template #suffix>
						<n-button quaternary circle size="tiny" :title="t('mobile.copyPublicUrl')" :aria-label="t('mobile.copyPublicUrl')" @click="copyPublicUrl">
							<template #icon><Icon name="mdi:content-copy" :size="16" /></template>
						</n-button>
					</template>
				</n-input>
				<div class="mt-1 text-xs text-[var(--n-text-color-3)]">{{ t("mobile.publicUrlHint") }}</div>
			</div>
			<div>
				<div class="mb-2 text-sm font-medium">{{ t("mobile.authDuration") }}</div>
				<n-select v-model:value="deviceTtlDays" :options="durationOptions" @update:value="generatePairing" />
			</div>
			<div class="flex min-h-[240px] flex-col items-center justify-center rounded-2xl border border-[var(--n-border-color)] bg-[var(--n-color-embedded)] p-4">
				<n-spin v-if="loading" />
				<n-qr-code v-else-if="qrCodeUrl" :value="qrCodeUrl" :size="200" />
				<n-empty v-else :description="t('mobile.pairFailed')" />
				<div class="mt-3 text-center text-xs text-[var(--n-text-color-3)]">{{ t("mobile.pairExpires", { days: deviceTtlDays }) }}</div>
			</div>
			<n-button secondary :loading="loading" @click="generatePairing">{{ t("mobile.refresh") }}</n-button>
			<div>
				<div class="mb-2 text-sm font-semibold">{{ t("mobile.devices") }}</div>
				<n-empty v-if="devices.length === 0" size="small" :description="t('mobile.noDevices')" />
				<div v-else class="flex max-h-40 flex-col gap-2 overflow-y-auto">
					<div v-for="device in devices" :key="device.id" class="flex items-center justify-between gap-3 rounded-xl bg-[var(--n-color-embedded)] px-3 py-2">
						<div class="min-w-0">
							<div class="truncate text-sm font-medium">{{ device.name }}</div>
							<div class="truncate text-xs text-[var(--n-text-color-3)]">{{ device.lastIp || '-' }}</div>
							<div class="truncate text-xs text-[var(--n-text-color-3)]">{{ t("mobile.deviceExpires", { time: formatDeviceExpiry(device.expiresAt) }) }}</div>
						</div>
						<n-button size="tiny" type="error" secondary :disabled="!!device.revokedAt" @click="revokeDevice(device)">{{ t("mobile.revoke") }}</n-button>
					</div>
				</div>
			</div>
		</div>
	</n-modal>
</template>
