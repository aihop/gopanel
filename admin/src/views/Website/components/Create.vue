<template>
	<!-- eslint-disable vue/no-v-model-argument -->
	<n-drawer v-model:show="visible" :width="502" :mask-closable="false">
		<n-drawer-content closable>
			<template #header>
				<div class="flex items-center gap-4">
					<n-button text @click="close">
						<template #icon>
							<Icon name="mdi:arrow-left" />
						</template>
						{{ $t("commons.button.back") }}
					</n-button>
					<n-divider vertical />
					<div>{{ title }}</div>
				</div>
			</template>

			<n-form ref="formRef" :model="form" :rules="rules" require-mark-placement="right-hanging">
				<n-form-item :label="$t('website.appType')" path="type">
					<n-select
						v-model:value="form.type"
						:disabled="actionType === 'update'"
						:options="[
							{ label: $t('website.staticSiteOption'), value: 'static' },
							{ label: $t('website.containerAppOption'), value: 'web_app' },
							{ label: $t('website.proxyOption'), value: 'proxy' },
							{ label: $t('website.redirectOption'), value: 'redirect' }
						]"
						:placeholder="$t('website.selectTypePlaceholder')"
					/>
				</n-form-item>
				<n-form-item :label="$t('website.primaryDomain')" path="primaryDomain">
					<n-input v-model:value="form.primaryDomain" :disabled="actionType === 'update'" :placeholder="$t('website.primaryDomainPlaceholder')" />
				</n-form-item>
				<n-form-item :label="$t('website.otherDomains')" path="otherDomains">
					<n-input type="textarea" v-model:value="form.otherDomains" :placeholder="$t('website.otherDomainsPlaceholder')" />
				</n-form-item>

				<n-checkbox class="mb-2" v-model:checked="form.redirectDomainsToPrimary" :disabled="!form.otherDomains">
					{{ $t("website.redirectDomainsToPrimary") }}
				</n-checkbox>

				<n-checkbox class="mb-6" v-model:checked="form.IPV6">{{ $t("website.ipv6") }}</n-checkbox>

				<n-form-item :label="$t('website.siteDirectory')" path="alias">
					<n-input v-model:value="form.alias" :disabled="actionType === 'update'" :placeholder="$t('website.siteDirPlaceholder')" />
				</n-form-item>

				<template v-if="form.type === 'web_app' || form.type === 'static'">
					<div
						v-if="actionType === 'update' && bindingRuntimeSummary"
						class="mb-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-600"
					>
						{{ bindingRuntimeSummary }}
					</div>
					<n-form-item :label="$t('website.codeSource')" path="codeSource">
						<n-radio-group v-model:value="form.codeSource">
							<n-space>
								<n-radio value="upload" v-if="form.type === 'static'">
									{{ $t("website.uploadLater") }}
								</n-radio>
								<n-radio value="git">{{ $t("website.customImageDeploy") }}</n-radio>
								<n-radio value="app_store">{{ $t("website.appStore") }}</n-radio>
							</n-space>
						</n-radio-group>
					</n-form-item>

					<n-form-item :label="$t('website.dockerImageAddress')" path="dockerImage" v-if="form.codeSource === 'git' || form.codeSource === 'docker'">
						<n-auto-complete
							v-model:value="form.dockerImage"
							:options="localImageOptions"
							:placeholder="$t('website.dockerImagePlaceholder')"
							:get-show="() => true"
							clearable
						/>
					</n-form-item>

					<n-form-item :label="$t('website.localCodePath')" path="codeDir" v-if="form.codeSource === 'upload'">
						<n-input v-model:value="form.codeDir" :placeholder="$t('website.localCodePathPlaceholder')" />
					</n-form-item>

					<div v-if="form.codeSource === 'upload'" class="mb-6 rounded border bg-slate-50 p-3 text-sm text-slate-500">
						{{ $t("website.uploadHint") }}
					</div>
				</template>

				<template v-if="form.type === 'proxy'">
					<n-form-item :label="$t('website.upstreams')" path="upstreams">
						<div class="w-full space-y-3">
							<div v-for="(item, index) in form.upstreams" :key="item.formKey" class="rounded-2xl border border-slate-200 bg-slate-50 p-3">
								<div class="flex items-center justify-between gap-3">
									<div class="text-xs font-medium text-slate-500">
										{{ `${$t("website.node")} ${index + 1}` }}
									</div>
									<n-button quaternary size="small" :disabled="form.upstreams.length === 1" @click="removeUpstream(index)">
										{{ $t("commons.button.delete") }}
									</n-button>
								</div>
								<div class="mt-3 flex flex-col gap-3 lg:flex-row">
									<n-select v-model:value="item.scheme" style="width: 120px" :options="upstreamSchemeOptions" />
									<n-input v-model:value="item.address" class="flex-1" :placeholder="$t('website.upstreamAddressPlaceholder')" />
								</div>
								<div class="mt-3">
									<n-checkbox v-model:checked="item.enabled">
										{{ $t("website.upstreamEnabled") }}
									</n-checkbox>
								</div>
							</div>
							<n-button dashed block @click="addUpstream">
								{{ $t("website.addUpstream") }}
							</n-button>
						</div>
					</n-form-item>

					<n-form-item :label="$t('website.healthCheck')">
						<n-grid cols="1 s:3" responsive="screen" :x-gap="12" :y-gap="12">
							<n-gi>
								<n-input v-model:value="form.upstreamHealthUri" :placeholder="$t('website.healthUriPlaceholder')" />
							</n-gi>
							<n-gi>
								<n-input v-model:value="form.upstreamHealthInterval" placeholder="10s" />
							</n-gi>
							<n-gi>
								<n-input v-model:value="form.upstreamHealthTimeout" placeholder="3s" />
							</n-gi>
						</n-grid>
					</n-form-item>
				</template>

				<n-form-item :label="$t('website.proxyTarget')" path="proxy" v-if="form.type === 'web_app'">
					<div class="w-full">
						<n-input v-model:value="form.proxy" :placeholder="$t('website.proxyTargetPlaceholder')" />
					</div>
				</n-form-item>

				<template v-if="form.type === 'redirect'">
					<n-form-item :label="$t('website.redirectTargetUrl')" path="proxy">
						<n-input v-model:value="form.proxy" :placeholder="$t('website.redirectTargetPlaceholder')" />
					</n-form-item>
					<n-form-item :label="$t('website.redirectType')">
						<n-radio-group v-model:value="form.redirectCode">
							<n-space>
								<n-radio :value="301">{{ $t("website.redirect301") }}</n-radio>
								<n-radio :value="302">{{ $t("website.redirect302") }}</n-radio>
								<n-radio :value="307">{{ $t("website.redirect307") }}</n-radio>
								<n-radio :value="308">{{ $t("website.redirect308") }}</n-radio>
							</n-space>
						</n-radio-group>
					</n-form-item>
				</template>

				<n-form-item :label="$t('website.selectInstalledApp')" path="appInstallId" v-if="form.codeSource === 'app_store'">
					<div class="w-full">
						<n-select
							v-model:value="form.appInstallId"
							:options="appInstallOptions"
							:placeholder="$t('website.selectInstalledAppPlaceholder')"
							@update:value="handleAppSelect"
						/>
						<div v-if="selectedAppRuntimeText" class="mt-2 text-xs text-slate-500">
							{{ selectedAppRuntimeText }}
						</div>
					</div>
				</n-form-item>

				<n-divider class="!my-6 text-slate-400">{{ $t("website.securityProtection") }}</n-divider>

				<div class="mb-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4">
					<div class="flex items-start justify-between gap-4">
						<div class="space-y-2">
							<div class="text-sm font-semibold text-slate-700">
								{{ $t("website.lightSecurityPolicy") }}
							</div>
							<div class="text-xs leading-6 text-slate-500">
								{{ $t("website.securityPolicyHint") }}
							</div>
						</div>
						<n-tag round :bordered="false" type="info">{{ $t("website.recommendedExtract") }}</n-tag>
					</div>
				</div>

				<n-form-item :label="$t('website.securityPolicy')">
					<n-radio-group v-model:value="securityPreset" @update:value="handleSecurityPresetChange">
						<n-space vertical>
							<n-radio value="off">{{ $t("website.protectionOff") }}</n-radio>
							<n-radio value="recommended">{{ $t("website.recommendedPolicy") }}</n-radio>
							<n-radio value="strict">{{ $t("website.strictPolicy") }}</n-radio>
						</n-space>
					</n-radio-group>
				</n-form-item>

				<div class="mb-4 flex flex-wrap gap-2">
					<n-tag v-for="item in securitySummary" :key="item" round :bordered="false" type="success">
						{{ item }}
					</n-tag>
					<span v-if="securitySummary.length === 0" class="text-xs text-slate-400">
						{{ $t("website.noSecurityEnabled") }}
					</span>
				</div>

				<n-divider class="!my-4" />

				<details class="mb-4 rounded-2xl border border-slate-200">
					<summary class="cursor-pointer select-none px-4 py-3 text-sm font-medium text-slate-600 hover:text-slate-800">
						{{ $t("website.customCaddyConfig") }}
						<span class="ml-2 text-xs text-slate-400">{{ $t("website.optional") }}</span>
					</summary>
					<div class="border-t border-slate-100 px-4 py-3">
						<n-alert type="info" :show-icon="true" class="mb-3 text-xs">
							{{ $t("website.caddyConfigHint") }}
						</n-alert>
						<div class="h-[240px] w-full overflow-hidden rounded-lg border border-slate-200">
							<vue-monaco-editor
								v-model:value="form.httpConfig"
								language="ini"
								theme="vs-dark"
								class="h-full w-full"
								:options="{
									automaticLayout: true,
									wordWrap: 'on',
									minimap: { enabled: false },
									scrollBeyondLastLine: false,
									padding: { top: 8, bottom: 8 },
									fontSize: 13,
									lineNumbersMinChars: 3
								}"
							/>
						</div>
					</div>
				</details>

				<n-form-item :label="$t('website.remark')" path="remark">
					<n-input type="textarea" v-model:value="form.remark" :placeholder="$t('website.remarkPlaceholder')" />
				</n-form-item>
			</n-form>

			<template #footer>
				<div class="flex justify-end gap-4">
					<n-button @click="close">{{ $t("commons.button.cancel") }}</n-button>
					<n-button type="primary" @click="onConfirm" :loading="loading">
						{{ $t("commons.button.confirm") }}
					</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>
<script setup lang="ts">
import type { FormInst } from "naive-ui"
import type { Website } from "@/api/interface/website"
import type { App } from "@/api/interface/apps"
import type { Container } from "@/api/interface/container"
import { computed, ref, watch, onMounted } from "vue"
import { websiteCreateAPI, websiteUpdateAPI } from "@/api/modules/website"
import { VueMonacoEditor } from "@guolao/vue-monaco-editor"
import { ListAppInstalled } from "@/api/modules/apps"
import { listAllImage } from "@/api/modules/container"
import { buildRuntimeBadgeText, buildRuntimeDetailText as formatRuntimeDetailText, getRunUserLabel } from "@/utils/runtime"
import { getWebsiteIpv6Value, normalizeWebsiteProtocol as normalizeWebsiteProtocolValue } from "@/utils/websiteRuntime"
import { t } from "@/i18n"

const visible = ref(false)
const loading = ref(false)
const emit = defineEmits(["confirm"])
const formRef = ref<FormInst | null>(null)
const actionType = ref("add")

type RuntimeOption = { label: string; value: number }
type ImageOption = { label: string; value: string }
type WebsiteDomainValue = string | { domain?: string }
type SecurityPreset = "off" | "recommended" | "strict"
type WebsiteFormUpstream = {
	formKey: number
	address: string
	scheme: string
	enabled: boolean
}
type WebsiteFormState = {
	id?: number
	type: string
	primaryDomain: string
	protocol: string
	alias: string
	otherDomains: string
	proxy: string
	IPV6: boolean
	appInstallId?: number
	remark: string
	codeSource: string
	dockerImage: string
	codeDir: string
	antiCrawler: boolean
	antiLeech: boolean
	rateLimitMode: string
	wafEnable: boolean
	blockSensitive: boolean
	ipAllowlist: string
	ipBlocklist: string
	securityHeader: boolean
	hstsEnabled: boolean
	httpConfig: string
	redirectCode: number
	redirectDomainsToPrimary: boolean
	upstreams: WebsiteFormUpstream[]
	upstreamHealthUri: string
	upstreamHealthInterval: string
	upstreamHealthTimeout: string
}
type WebsiteFormRecord = Website.WebsiteDTO & {
	domains?: WebsiteDomainValue[]
	runtimeBindingSummary?: string
	engineEnv?: string
	status?: string | boolean
	rateLimitMode?: string
	wafEnable?: boolean
	blockSensitive?: boolean
	ipAllowlist?: string
	ipBlocklist?: string
	securityHeader?: boolean
	hstsEnabled?: boolean
}

let upstreamFormKey = 0
function createDefaultUpstream(values: Partial<Omit<WebsiteFormUpstream, "formKey">> = {}): WebsiteFormUpstream {
	return {
		formKey: ++upstreamFormKey,
		address: values.address || "",
		scheme: values.scheme || "http",
		enabled: values.enabled !== false
	}
}

function buildProxyFromUpstream(item?: { address?: string; scheme?: string } | null): string {
	if (!item) return ""
	const address = String(item.address || "").trim()
	if (!address) return ""
	return item.scheme === "https" ? `https://${address}` : `http://${address}`
}

function normalizeUpstreamForm(record?: Website.WebsiteUpstream[] | null, proxy?: string): WebsiteFormUpstream[] {
	if (record && record.length > 0) {
		return record.map(item =>
			createDefaultUpstream({
				address: item.address || "",
				scheme: item.scheme || "http",
				enabled: item.enabled !== false
			})
		)
	}
	const fallback = String(proxy || "").trim()
	if (!fallback) {
		return [createDefaultUpstream()]
	}
	if (fallback.startsWith("https://")) {
		return [createDefaultUpstream({ address: fallback.replace(/^https:\/\//, ""), scheme: "https" })]
	}
	if (fallback.startsWith("http://")) {
		return [createDefaultUpstream({ address: fallback.replace(/^http:\/\//, ""), scheme: "http" })]
	}
	return [createDefaultUpstream({ address: fallback })]
}

function pickUpstreamHealth(record?: Website.WebsiteUpstream[] | null) {
	const first = record?.find(item => item.enabled !== false) || record?.[0]
	return {
		uri: first?.healthUri || "",
		interval: first?.healthInterval || "",
		timeout: first?.healthTimeout || ""
	}
}

function createDefaultForm(): WebsiteFormState {
	return {
		id: undefined,
		type: "static",
		primaryDomain: "",
		protocol: "HTTPS",
		alias: "",
		otherDomains: "",
		proxy: "",
		IPV6: false,
		appInstallId: undefined,
		remark: "",
		codeSource: "upload",
		dockerImage: "",
		codeDir: "",
		antiCrawler: false,
		antiLeech: false,
		rateLimitMode: "none",
		wafEnable: false,
		blockSensitive: false,
		ipAllowlist: "",
		ipBlocklist: "",
		securityHeader: false,
		hstsEnabled: false,
		httpConfig: "",
		redirectCode: 301,
		redirectDomainsToPrimary: true,
		upstreams: [createDefaultUpstream()],
		upstreamHealthUri: "",
		upstreamHealthInterval: "",
		upstreamHealthTimeout: ""
	}
}

function sanitizeOtherDomains(raw: string, primary: string): string {
	return raw
		.split("\n")
		.map(s => s.trim())
		.filter(Boolean)
		.filter(s => s.toLowerCase() !== (primary || "").trim().toLowerCase())
		.join("\n")
}

const title = ref(t("website.addDomainTitle"))
const securityPreset = ref<SecurityPreset>("recommended")
const bindingRuntimeSummary = ref("")

const appInstallOptions = ref<RuntimeOption[]>([])
const rawAppList = ref<App.AppInstalledInfo[]>([])

const localImageOptions = ref<ImageOption[]>([])
const upstreamSchemeOptions = [
	{ label: "HTTP", value: "http" },
	{ label: "HTTPS", value: "https" }
]

const selectedAppRuntimeText = computed(() => {
	const item = rawAppList.value.find(app => app.id === form.value.appInstallId)
	if (!item) return ""
	return formatRuntimeDetailText(item, {
		prefix: item.containerName ? `${t("website.containerPrefix")}${item.containerName}` : "",
		kindFallback: "Runtime",
		userFallback: t("website.imageDefault"),
		runtimePrefix: t("website.runtimePrefix"),
		runUserPrefix: t("website.userPrefix")
	})
})

const securitySummary = computed(() => {
	const list: string[] = []
	if (form.value.antiCrawler) list.push(t("website.secAntiCrawler"))
	if (form.value.antiLeech) list.push(t("website.secAntiLeech"))
	if (form.value.rateLimitMode === "normal") list.push(t("website.secRateLimitNormal"))
	if (form.value.rateLimitMode === "strict") list.push(t("website.secRateLimitStrict"))
	if (form.value.wafEnable) list.push(t("website.secLightweightWaf"))
	if (form.value.blockSensitive) list.push(t("website.secSensitiveFileProtection"))
	if (form.value.securityHeader) list.push(t("website.secSecurityHeaders"))
	if (form.value.hstsEnabled) list.push(t("website.secHsts"))
	return list
})

function applySecurityPreset(preset: "off" | "recommended" | "strict") {
	if (preset === "off") {
		form.value.antiCrawler = false
		form.value.antiLeech = false
		form.value.rateLimitMode = "none"
		form.value.wafEnable = false
		form.value.blockSensitive = false
		form.value.securityHeader = false
		form.value.hstsEnabled = false
		return
	}
	if (preset === "strict") {
		form.value.antiCrawler = true
		form.value.antiLeech = true
		form.value.rateLimitMode = "strict"
		form.value.wafEnable = true
		form.value.blockSensitive = true
		form.value.securityHeader = true
		form.value.hstsEnabled = normalizeWebsiteProtocolValue(form.value.protocol) === "HTTPS"
		return
	}
	form.value.antiCrawler = true
	form.value.antiLeech = true
	form.value.rateLimitMode = "normal"
	form.value.wafEnable = true
	form.value.blockSensitive = true
	form.value.securityHeader = true
	form.value.hstsEnabled = normalizeWebsiteProtocolValue(form.value.protocol) === "HTTPS"
}

function syncSecurityPreset() {
	if (
		!form.value.antiCrawler &&
		!form.value.antiLeech &&
		form.value.rateLimitMode === "none" &&
		!form.value.wafEnable &&
		!form.value.blockSensitive &&
		!form.value.securityHeader &&
		!form.value.hstsEnabled
	) {
		securityPreset.value = "off"
		return
	}
	if (
		form.value.antiCrawler &&
		form.value.antiLeech &&
		form.value.rateLimitMode === "strict" &&
		form.value.wafEnable &&
		form.value.blockSensitive &&
		form.value.securityHeader
	) {
		securityPreset.value = "strict"
		return
	}
	securityPreset.value = "recommended"
}

function handleSecurityPresetChange(value: "off" | "recommended" | "strict") {
	securityPreset.value = value
	applySecurityPreset(value)
}

function syncSourceSpecificFields(source: string) {
	if (source !== "app_store") {
		form.value.appInstallId = undefined
	}
	if (source !== "git") {
		form.value.dockerImage = ""
	}
	if (source !== "upload") {
		form.value.codeDir = ""
	}
}

function buildProxyUpstreamsPayload(): Website.WebsiteUpstreamInput[] {
	return form.value.upstreams
		.map(item => ({
			address: String(item.address || "").trim(),
			scheme: item.scheme || "http",
			enabled: item.enabled,
			healthUri: form.value.upstreamHealthUri.trim(),
			healthInterval: form.value.upstreamHealthInterval.trim(),
			healthTimeout: form.value.upstreamHealthTimeout.trim()
		}))
		.filter(item => item.address)
}

function applyProxyTarget(target: string) {
	form.value.proxy = target
	if (form.value.type !== "proxy") {
		return
	}
	form.value.upstreams = normalizeUpstreamForm(undefined, target)
}

function addUpstream() {
	form.value.upstreams.push(createDefaultUpstream())
}

function removeUpstream(index: number) {
	if (form.value.upstreams.length <= 1) return
	form.value.upstreams.splice(index, 1)
	if (form.value.upstreams.length === 0) {
		form.value.upstreams = [createDefaultUpstream()]
	}
}

function buildCreatePayload(): Website.WebSiteCreateReq {
	const source = form.value.codeSource
	const upstreams = form.value.type === "proxy" ? buildProxyUpstreamsPayload() : undefined
	return {
		type: form.value.type,
		primaryDomain: form.value.primaryDomain,
		protocol: normalizeWebsiteProtocolValue(form.value.protocol) || "HTTPS",
		alias: form.value.alias,
		otherDomains: sanitizeOtherDomains(form.value.otherDomains, form.value.primaryDomain),
		proxy: form.value.type === "proxy" ? buildProxyFromUpstream(upstreams?.find(item => item.enabled !== false)) : form.value.proxy,
		IPV6: form.value.IPV6,
		appInstallId: source === "app_store" ? form.value.appInstallId : undefined,
		remark: form.value.remark,
		codeSource: source,
		gitRepo: source === "git" ? form.value.dockerImage : "",
		codeDir: source === "upload" ? form.value.codeDir : "",
		antiCrawler: form.value.antiCrawler,
		antiLeech: form.value.antiLeech,
		rateLimitMode: form.value.rateLimitMode,
		wafEnable: form.value.wafEnable,
		blockSensitive: form.value.blockSensitive,
		ipAllowlist: form.value.ipAllowlist,
		ipBlocklist: form.value.ipBlocklist,
		securityHeader: form.value.securityHeader,
		hstsEnabled: form.value.hstsEnabled,
		httpConfig: form.value.httpConfig,
		redirectCode: form.value.redirectCode,
		redirectDomainsToPrimary: form.value.redirectDomainsToPrimary,
		upstreams
	}
}

function buildUpdatePayload(): Website.WebSiteUpdateReq {
	const source = form.value.codeSource
	const upstreams = form.value.type === "proxy" ? buildProxyUpstreamsPayload() : undefined
	return {
		id: form.value.id || 0,
		primaryDomain: form.value.primaryDomain,
		protocol: normalizeWebsiteProtocolValue(form.value.protocol),
		otherDomains: sanitizeOtherDomains(form.value.otherDomains, form.value.primaryDomain),
		proxy: form.value.type === "proxy" ? buildProxyFromUpstream(upstreams?.find(item => item.enabled !== false)) : form.value.proxy,
		codeSource: source,
		IPV6: form.value.IPV6,
		remark: form.value.remark,
		antiCrawler: form.value.antiCrawler,
		antiLeech: form.value.antiLeech,
		rateLimitMode: form.value.rateLimitMode,
		wafEnable: form.value.wafEnable,
		blockSensitive: form.value.blockSensitive,
		ipAllowlist: form.value.ipAllowlist,
		ipBlocklist: form.value.ipBlocklist,
		securityHeader: form.value.securityHeader,
		hstsEnabled: form.value.hstsEnabled,
		httpConfig: form.value.httpConfig,
		redirectCode: form.value.redirectCode,
		redirectDomainsToPrimary: form.value.redirectDomainsToPrimary,
		upstreams
	}
}

const fetchApps = async () => {
	try {
		const res = await ListAppInstalled()
		if (res.data) {
			rawAppList.value = res.data as App.AppInstalledInfo[]
			appInstallOptions.value = rawAppList.value.map(app => ({
				label: buildAppOptionLabel(app),
				value: app.id
			}))
		}
	} catch (e) {
		console.error("Failed to fetch installed apps", e)
	}
}

const fetchLocalImages = async () => {
	try {
		const res = await listAllImage()
		if (res.data) {
			localImageOptions.value = (res.data as Container.ImageInfo[]).map(item => ({
				label: item.tags && item.tags.length > 0 ? item.tags[0] : item.name,
				value: item.tags && item.tags.length > 0 ? item.tags[0] : item.name
			}))
		}
	} catch (e) {
		console.error("Failed to fetch local images", e)
	}
}

const handleAppSelect = (val: number) => {
	if (!val) return
	const app = rawAppList.value.find(a => a.id === val)
	if (app && app.httpPort) {
		applyProxyTarget(`http://127.0.0.1:${app.httpPort}`)
	} else if (app && app.httpsPort) {
		applyProxyTarget(`https://127.0.0.1:${app.httpsPort}`)
	} else {
		applyProxyTarget("")
	}
}

function buildAppOptionLabel(app: App.AppInstalledInfo) {
	const port = app.httpPort ? `${t("website.portLabel")}${app.httpPort}` : app.httpsPort ? `HTTPS:${app.httpsPort}` : t("website.noPort")
	return `${app.name} · ${port} · ${buildRuntimeBadgeText(app, { kindFallback: "Runtime" })} · ${t("website.userPrefix")}${getRunUserLabel(app, { userFallback: t("website.imageDefault") })}`
}

onMounted(() => {
	fetchApps()
	fetchLocalImages()
})

const form = ref<WebsiteFormState>(createDefaultForm())

const rules = {
	type: {
		required: true,
		message: t("website.typeRequired"),
		trigger: "blur"
	},
	primaryDomain: {
		required: true,
		message: t("website.primaryDomainRequired"),
		trigger: "blur"
	},
	alias: {
		required: true,
		message: t("website.aliasRequired"),
		trigger: "blur"
	},
	otherDomains: {
		validator(_rule: unknown, value: string) {
			if (!value) return true
			const lines = value
				.split("\n")
				.map(s => s.trim())
				.filter(Boolean)
			const primary = (form.value.primaryDomain || "").trim().toLowerCase()
			if (!primary) return true
			for (const line of lines) {
				if (line.toLowerCase() === primary) {
					return new Error(t("website.otherDomainsDuplicate"))
				}
			}
			return true
		},
		trigger: "blur"
	},
	upstreams: {
		required: true,
		validator() {
			if (form.value.type !== "proxy") {
				return true
			}
			const items = form.value.upstreams.filter(item => String(item.address || "").trim())
			if (items.length === 0) {
				return new Error(t("website.upstreamRequired"))
			}
			if (!items.some(item => item.enabled)) {
				return new Error(t("website.upstreamEnabledRequired"))
			}
			return true
		},
		trigger: "blur"
	},
	proxy: {
		required: true,
		validator(_rule: unknown, value: string) {
			if (form.value.type === "web_app" && !value) {
				return new Error(t("website.proxyTargetRequired"))
			}
			return true
		},
		trigger: "blur"
	},
	appInstallId: {
		required: true,
		validator(_rule: unknown, value: number) {
			if (form.value.codeSource === "app_store" && !value) {
				return new Error(t("website.appInstallRequired"))
			}
			return true
		},
		trigger: "blur"
	},
	dockerImage: {
		required: true,
		validator(_rule: unknown, value: string) {
			if (form.value.codeSource === "git" && !String(value || "").trim()) {
				return new Error(t("website.dockerImageRequired"))
			}
			return true
		},
		trigger: "blur"
	}
}

const onConfirm = async () => {
	try {
		await formRef.value?.validate()
	} catch (errors) {
		return
	}
	try {
		loading.value = true
		const createPayload = buildCreatePayload()
		const updatePayload = buildUpdatePayload()
		let res = await (actionType.value === "add" ? websiteCreateAPI(createPayload) : websiteUpdateAPI(updatePayload))
		emit("confirm", res, loading)
	} catch (error) {
		console.error(error)
	} finally {
		loading.value = false
	}
}

const open = (record?: WebsiteFormRecord, action: "add" | "update" = "add") => {
	visible.value = true
	if (action === "add") {
		actionType.value = "add"
		title.value = t("website.addWebsite")
		bindingRuntimeSummary.value = ""
		form.value = {
			...createDefaultForm(),
			antiCrawler: true,
			antiLeech: true,
			rateLimitMode: "normal",
			wafEnable: true,
			blockSensitive: true,
			securityHeader: true,
			upstreams: [createDefaultUpstream()]
		}
		securityPreset.value = "recommended"
	} else {
		if (!record) return
		actionType.value = "update"
		title.value = t("website.updateWebsite")
		bindingRuntimeSummary.value = record.runtimeBindingSummary || ""
		const editableRecord: WebsiteFormRecord = { ...record }
		if (editableRecord.domains && editableRecord.domains.length > 0) {
			const domains = editableRecord.domains as WebsiteDomainValue[]
			let otherDomain = ""
			for (let i = 0; i < domains.length; i++) {
				const domainValue = domains[i]
				const domain = typeof domainValue === "string" ? domainValue : domainValue?.domain || ""
				if (!domain || domain === editableRecord.primaryDomain) {
					continue
				}
				if (i == 0) {
					otherDomain += domain
				} else {
					otherDomain += "\n" + domain
				}
			}
			editableRecord.otherDomains = otherDomain
		}
		// Keep editableRecord.otherDomains from API response when domains array is not available
		const health = pickUpstreamHealth(editableRecord.upstreams)
		form.value = {
			...createDefaultForm(),
			...editableRecord,
			protocol: normalizeWebsiteProtocolValue(editableRecord.protocol) || "HTTPS",
			IPV6: getWebsiteIpv6Value(editableRecord),
			dockerImage: editableRecord.engineEnv || "",
			rateLimitMode: editableRecord.rateLimitMode || "none",
			wafEnable: editableRecord.wafEnable || false,
			blockSensitive: editableRecord.blockSensitive || false,
			ipAllowlist: editableRecord.ipAllowlist || "",
			ipBlocklist: editableRecord.ipBlocklist || "",
			securityHeader: !!editableRecord.securityHeader,
			hstsEnabled: !!editableRecord.hstsEnabled,
			upstreams: normalizeUpstreamForm(editableRecord.upstreams, editableRecord.proxy),
			upstreamHealthUri: health.uri,
			upstreamHealthInterval: health.interval,
			upstreamHealthTimeout: health.timeout
		}
		syncSecurityPreset()
	}
}
watch(
	() => form.value.primaryDomain,
	val => {
		form.value.alias = val || ""
	}
)
watch(
	() => form.value.alias,
	val => {
		if (val.startsWith("http://") || val.startsWith("https://")) {
			form.value.alias = val.replace("http://", "").replace("https://", "")
		}
	}
)
watch(
	() => form.value.codeSource,
	val => {
		syncSourceSpecificFields(val)
	}
)
watch(
	() => form.value.type,
	val => {
		if (val === "web_app" && form.value.codeSource === "upload") {
			form.value.codeSource = "git" // Default to custom image when switching away from static
		}
		if (val === "proxy" && form.value.upstreams.length === 0) {
			form.value.upstreams = [createDefaultUpstream()]
		}
	}
)
const close = () => {
	visible.value = false
	loading.value = false
}
defineExpose({
	open,
	close
})
</script>
