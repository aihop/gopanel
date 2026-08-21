<template>
	<SystemUpdateProvider :poll-interval-ms="120000" v-slot="u">
		<n-button
			v-if="u.effectiveNeedUpdate === true"
			quaternary
			class="update-entry-need !rounded-2xl !px-4"
			@click="drawerOpen = true"
		>
			<div class="flex items-center gap-2">
				<Icon :size="20" name="carbon:cloud-download" class="update-icon-need" />
				<div class="flex flex-col items-start leading-tight">
					<div class="text-sm font-semibold text-slate-900">
						{{ u.updateInfo.title || t("setting.versionUpdate") }}
					</div>
					<div class="text-xs text-slate-600">
						{{
							u.updateInfo.description ||
							t("setting.newVersionAvailable", { version: u.updateInfo.latestVersionName || "-" })
						}}
					</div>
				</div>
			</div>
		</n-button>

		<n-drawer
			v-if="u.effectiveNeedUpdate === true"
			:show="drawerOpen"
			width="460"
			placement="right"
			@update:show="val => handleDrawerChange(val, u)"
		>
			<n-drawer-content :title="t('setting.versionUpdate')" :closable="!u.isReading">
				<div class="space-y-4">
					<div class="rounded-[18px] border border-slate-200 bg-slate-50/80 p-4">
						<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
							{{ t("setting.systemCurrentVersion") }}
						</div>
						<div class="fg-base-100 mt-2 text-lg font-semibold">
							{{ u.versionInfo.versionName || "-" }}
							<span class="text-sm text-slate-500">
								{{ t("setting.versionCodeLabel") }}: {{ u.versionInfo.versionCode || "-" }}
								{{ t("setting.buildTime") }}: {{ formatTime(u.versionInfo.buildTime) || "-" }}
							</span>
						</div>
					</div>

					<div class="rounded-[18px] border border-slate-200 bg-slate-50/80 p-4">
						<div class="flex items-start justify-between gap-3">
							<div>
								<div class="font-semibold uppercase tracking-[0.16em] text-blue-600">
									{{ t("setting.latestVersion") }}
									<span class="text-xs uppercase">
										{{ t("setting.updateAvailable") }}
									</span>
								</div>
								<div class="mt-2 text-xs text-gray-500">
									{{ t("setting.version") }}{{ u.updateInfo.latestVersionName }}
									{{ t("setting.versionCodeLabel") }}-{{ u.updateInfo.latestVersionCode }}
								</div>
							</div>
							<div class="text-right">
								<div class="fg-base-100 mt-2 text-lg font-semibold">
									<n-tag type="warning" round :bordered="false">
										{{ t("setting.upgradable") }}
									</n-tag>
								</div>
							</div>
						</div>
						<n-descriptions
							class="mt-3"
							bordered
							size="small"
							:column="1"
							label-placement="left"
							:label-style="{ width: '90px' }"
						>
							<n-descriptions-item :label="t('setting.versionName')">
								{{ u.updateInfo.title || "-" }}
							</n-descriptions-item>
							<n-descriptions-item :label="t('setting.releaseTime')">
								{{ formatTime(u.updateInfo.createAt) || "-" }}
							</n-descriptions-item>
							<n-descriptions-item :label="t('setting.versionIntro')">
								{{ u.updateInfo.description || "-" }}
							</n-descriptions-item>
							<n-descriptions-item :label="t('setting.downloadUrl')">
								<a
									v-if="u.updateInfo.downloadUrl"
									class="break-all text-blue-600 hover:underline"
									:href="u.updateInfo.downloadUrl"
									target="_blank"
									rel="noreferrer"
								>
									{{ u.updateInfo.downloadUrl }}
								</a>
								<span v-else>-</span>
							</n-descriptions-item>
						</n-descriptions>
					</div>

					<n-space justify="end">
						<n-button ghost class="!rounded-[16px]" :loading="u.checkingUpdate" @click="u.checkUpdate">
							{{ t("setting.checkUpdate") }}
						</n-button>
						<n-button
							type="primary"
							class="!rounded-[16px]"
							:loading="u.updating"
							:disabled="u.effectiveNeedUpdate !== true"
							@click="handleUpgrade(u)"
						>
							{{ t("setting.upgradeNow") }}
						</n-button>
					</n-space>

					<div v-if="u.logVisible" class="space-y-3">
						<div
							class="flex items-center justify-between gap-3 rounded-[18px] border border-slate-200 bg-slate-50/80 px-4 py-3"
						>
							<div class="text-sm text-slate-500">
								{{ u.logStatusText }}
							</div>
							<n-tag :type="u.logStatusTag" round :bordered="false">
								{{ u.logStatusLabel }}
							</n-tag>
						</div>

						<div :ref="u.setTerminalRef" class="update-log-terminal">
							<div v-for="(line, idx) in u.streamLogs" :key="idx" class="whitespace-pre-wrap break-words">
								{{ line }}
							</div>
							<div v-if="u.streamLogs.length === 0" class="italic text-slate-500">
								{{ t("setting.connectingLogStream") }}
							</div>
						</div>
					</div>

					<div v-if="u.updateInfo.content" class="rounded-[18px] border border-slate-200 bg-white p-4">
						<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
							{{ t("setting.upgradeNotes") }}
						</div>
						<div
							class="update-html mt-3 whitespace-pre-wrap break-words"
							v-html="u.updateInfo.content"
						></div>
					</div>
				</div>
			</n-drawer-content>
		</n-drawer>
	</SystemUpdateProvider>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { NButton, NDrawer, NDrawerContent, NTag, NSpace, useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import SystemUpdateProvider from "@/components/system/SystemUpdateProvider.vue"
import { formatTime } from "@/utils/date"

const drawerOpen = ref(false)
const dialog = useDialog()
const message = useMessage()
const { t } = useI18n()

const handleDrawerChange = (val: boolean, u: any) => {
	if (!val) {
		u.setLogVisible(false)
	}
	drawerOpen.value = val
}

const handleUpgrade = (u: any) => {
	dialog.warning({
		title: t("setting.updateConfirm"),
		content: t("setting.updateConfirmContent", { version: u.updateInfo.latestVersionName }),
		positiveText: t("setting.confirmUpdate"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			const res = await u.startUpgrade({
				containerName: "gopanel",
				currentVersion: u.versionInfo.versionName,
				targetVersion: u.updateInfo.latestVersionName
			})
			if (res.ok) {
				u.setLogVisible(true)
				message.success(t("setting.updateStarted"))
			} else {
				message.error(res.msg || t("setting.updateFailed"))
			}
		}
	})
}
</script>

<style scoped>
.update-entry-need {
	border: 1px solid rgba(37, 99, 235, 0.35);
	background: linear-gradient(90deg, rgba(219, 234, 254, 0.85), rgba(191, 219, 254, 0.55));
}

.update-entry-ok {
	border: 1px solid rgba(148, 163, 184, 0.32);
	background: rgba(241, 245, 249, 0.55);
}

.update-icon-need {
	color: rgb(37, 99, 235);
}

.update-icon-ok {
	color: rgb(59, 130, 246);
}

.update-html :deep(h1),
.update-html :deep(h2),
.update-html :deep(h3) {
	color: rgb(15, 23, 42);
	font-weight: 700;
}

.update-html :deep(h1) {
	font-size: 18px;
	margin: 10px 0 6px;
}

.update-html :deep(h2) {
	font-size: 16px;
	margin: 10px 0 6px;
}

.update-html :deep(p) {
	color: rgb(71, 85, 105);
	line-height: 1.8;
	margin: 8px 0;
}

.update-html :deep(ul),
.update-html :deep(ol) {
	padding-left: 18px;
	color: rgb(71, 85, 105);
	line-height: 1.8;
	margin: 8px 0;
}

.update-html :deep(code) {
	background: rgba(15, 23, 42, 0.06);
	padding: 2px 6px;
	border-radius: 8px;
	color: rgb(15, 23, 42);
}

.update-html :deep(pre) {
	background: #0f172a;
	color: #dbeafe;
	padding: 12px 14px;
	border-radius: 14px;
	overflow: auto;
	line-height: 1.7;
	margin: 10px 0;
}

.update-html :deep(a) {
	color: rgb(37, 99, 235);
	text-decoration: underline;
}

.update-html :deep(hr) {
	border: none;
	border-top: 1px solid rgba(148, 163, 184, 0.35);
	margin: 14px 0;
}

.update-html :deep(blockquote) {
	margin: 10px 0;
	padding: 10px 12px;
	border-left: 3px solid rgba(37, 99, 235, 0.5);
	background: rgba(219, 234, 254, 0.35);
	border-radius: 12px;
	color: rgb(51, 65, 85);
}

.update-html :deep(table) {
	width: 100%;
	border-collapse: separate;
	border-spacing: 0;
	border: 1px solid rgba(226, 232, 240, 1);
	border-radius: 14px;
	overflow: hidden;
	margin: 12px 0;
}

.update-html :deep(th),
.update-html :deep(td) {
	padding: 10px 12px;
	border-bottom: 1px solid rgba(226, 232, 240, 1);
	color: rgb(51, 65, 85);
}

.update-html :deep(th) {
	background: rgba(248, 250, 252, 1);
	font-weight: 700;
}

.update-html :deep(tr:last-child td) {
	border-bottom: none;
}

.update-html :deep(img) {
	max-width: 100%;
	height: auto;
	border-radius: 14px;
	margin: 10px 0;
}

.update-log-terminal {
	height: 300px;
	overflow: auto;
	border-radius: 18px;
	background: #0f172a;
	color: #dbeafe;
	padding: 14px;
	font-family:
		ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New",
		monospace;
	font-size: 13px;
	line-height: 1.7;
}
</style>
