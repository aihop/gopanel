<template>
	<SystemUpdateProvider v-slot="u" class="mt-4">
		<div class="update-page-root">
			<n-space vertical size="large">
				<div
					size="small"
					:bordered="false"
					class="bg-base-accent border-base-accent rounded-[28px] p-8 shadow-[0_18px_48px_rgba(15,23,42,0.08)]"
				>
					<div class="flex items-start justify-between gap-4">
						<div>
							<div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
								Version Center
							</div>
							<div class="fg-base-100 my-3 text-2xl font-semibold">{{ t("setting.versionInfo") }}</div>
						</div>
						<n-tag type="info" round :bordered="false">{{ t("setting.currentRuntimeEnv") }}</n-tag>
					</div>

					<n-descriptions :column="4" bordered class="mt-2">
						<n-descriptions-item :label="t('setting.currentVersionLabel')">
							{{ u.versionInfo.versionName || "-" }}
						</n-descriptions-item>
						<n-descriptions-item :label="t('setting.currentVersionCode')">
							{{ u.versionInfo.versionCode || "-" }}
						</n-descriptions-item>
						<n-descriptions-item :label="t('setting.buildTime')">
							{{ u.formatTime(u.versionInfo.buildTime) || "-" }}
						</n-descriptions-item>
						<n-descriptions-item :label="t('setting.installPath')">
							{{ u.versionInfo.installPath || "-" }}
						</n-descriptions-item>
					</n-descriptions>
				</div>

				<div
					v-if="u.effectiveNeedUpdate === false"
					class="rounded-[28px] p-8 shadow-[0_18px_48px_rgba(15,23,42,0.08)]"
				>
					<div class="flex items-start justify-between gap-4">
						<div>
							<div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
								Update Center
							</div>
							<div class="fg-base-100 my-3 text-2xl font-semibold">
								{{ t("setting.alreadyLatest") }}
							</div>
							<div class="my-3 text-sm leading-7 text-slate-500">
								{{ t("setting.alreadyLatestDesc") }}
							</div>
						</div>
						<n-tag type="success" round :bordered="false">
							{{ t("setting.latestTag") }}
						</n-tag>
					</div>
					<div class="flex justify-end">
						<n-button
							ghost
							class="!rounded-[18px]"
							:loading="u.checkingUpdate"
							@click="handleCheckUpdate(u)"
						>
							{{ t("setting.recheck") }}
						</n-button>
					</div>
				</div>

				<div v-else class="rounded-[28px] p-8 shadow-[0_18px_48px_rgba(15,23,42,0.08)]">
					<div class="flex items-start justify-between gap-4">
						<div>
							<div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
								Update Center
							</div>
							<div class="fg-base-100 my-3 text-2xl font-semibold">
								{{ u.updateInfo.title || t("setting.systemUpdate") }}
							</div>
							<div class="my-3 text-sm leading-7 text-slate-500">
								{{ u.updateInfo.description || t("setting.updateDefaultDesc") }}
							</div>
						</div>
						<n-tag
							v-if="u.effectiveNeedUpdate !== undefined"
							:type="u.effectiveNeedUpdate === true ? 'warning' : 'info'"
							round
							:bordered="false"
						>
							{{ u.effectiveNeedUpdate === true ? t("setting.upgradable") : t("setting.checking") }}
						</n-tag>
					</div>

					<div class="update-section">
						<div class="update-info">
							<div class="grid gap-4 sm:grid-cols-3">
								<div class="rounded-[22px] border border-slate-200 bg-slate-50/80 p-5">
									<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
										{{ t("setting.latestVersion") }}
									</div>
									<div class="fg-base-100 mt-3 text-xl font-semibold">
										{{ u.updateInfo.latestVersionName || t("setting.checkingEllipsis") }}
									</div>
								</div>
								<div class="rounded-[22px] border border-slate-200 bg-slate-50/80 p-5">
									<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
										{{ t("setting.versionCode") }}
									</div>
									<div class="fg-base-100 mt-3 text-xl font-semibold">
										{{ u.updateInfo.latestVersionCode || t("setting.checkingEllipsis") }}
									</div>
								</div>
								<div class="rounded-[22px] border border-slate-200 bg-slate-50/80 p-5">
									<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
										{{ t("setting.updateStatus") }}
									</div>
									<div class="fg-base-100 mt-3 text-xl font-semibold">
										{{
											u.effectiveNeedUpdate === true
												? t("setting.suggestedUpgrade")
												: t("setting.checkingEllipsis")
										}}
									</div>
								</div>
							</div>

							<n-descriptions class="mt-5" bordered :column="3" label-placement="top">
								<n-descriptions-item :label="t('setting.currentVersionLabel')">
									{{ u.versionInfo.versionName || u.updateInfo.curVersion || "-" }}
								</n-descriptions-item>
								<n-descriptions-item :label="t('setting.releaseTime')">
									{{ u.formatTime(u.updateInfo.createAt || "") || "-" }}
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
						<div class="update-actions">
							<n-button
								ghost
								class="!rounded-[18px]"
								:loading="u.checkingUpdate"
								@click="handleCheckUpdate(u)"
							>
								{{ t("setting.recheck") }}
							</n-button>
							<n-button
								type="primary"
								class="!rounded-[18px] shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
								:loading="u.updating"
								:disabled="u.effectiveNeedUpdate !== true"
								@click="handleUpdate(u)"
							>
								{{ t("setting.upgradeNow") }}
							</n-button>
						</div>
					</div>

					<div
						v-if="u.effectiveNeedUpdate === true && u.updateInfo.content"
						class="mt-6 rounded-[22px] border border-slate-200 bg-white p-5"
					>
						<div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
							{{ t("setting.upgradeNotes") }}
						</div>
						<div
							class="update-html mt-3 whitespace-pre-wrap break-words"
							v-html="u.updateInfo.content"
						></div>
					</div>
				</div>
			</n-space>

			<n-modal
				:show="u.logVisible"
				:mask-closable="!u.isReading"
				:closable="!u.isReading"
				preset="card"
				:title="t('setting.updateLog')"
				style="width: 80%; max-width: 1000px"
				:show-close="!u.isReading"
				@update:show="val => u.setLogVisible(val)"
			>
				<div class="space-y-4">
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
				<template #footer>
					<div class="flex justify-end gap-2">
						<n-button v-if="!u.isReading" type="primary" @click="u.setLogVisible(false)">
							{{ t("commons.button.close") }}
						</n-button>
						<n-button v-else disabled>{{ t("setting.upgrading") }}</n-button>
					</div>
				</template>
			</n-modal>
		</div>
	</SystemUpdateProvider>
</template>

<script setup lang="ts">
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import SystemUpdateProvider from "@/components/system/SystemUpdateProvider.vue"

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const handleCheckUpdate = async (u: any) => {
	const checked = await u.checkUpdate()
	if (!checked) {
		message.error(t("setting.updateCheckFailed"))
		return
	}
	if (u.effectiveNeedUpdate === true) {
		message.warning(t("setting.updateAvailable"))
	} else {
		message.success(t("setting.noUpgrade"))
	}
}

const handleUpdate = async (u: any) => {
	if (u.effectiveNeedUpdate !== true) {
		message.warning(t("setting.noUpgrade"))
		return
	}

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
				message.success(t("setting.updateStarted"))
			} else {
				message.error(res.msg || t("setting.updateFailed"))
			}
		}
	})
}
</script>

<style scoped>
.update-section {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	gap: 24px;
	flex-wrap: wrap;
}

.update-info {
	flex: 1;
}

.update-actions {
	display: flex;
	gap: 10px;
	flex-shrink: 0;
	align-self: flex-end;
}

.update-log-terminal {
	height: 400px;
	overflow: auto;
	border-radius: 18px;
	background: #0f172a;
	color: #dbeafe;
	padding: 16px;
	font-family:
		ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New",
		monospace;
	font-size: 13px;
	line-height: 1.7;
}

.update-html :deep(h1),
.update-html :deep(h2),
.update-html :deep(h3) {
	color: var(--fg-default-color);
	font-weight: 700;
}

.update-html :deep(h1) {
	font-size: 20px;
	margin: 12px 0 8px;
}

.update-html :deep(h2) {
	font-size: 16px;
	margin: 12px 0 8px;
}

.update-html :deep(p) {
	color: var(--fg-secondary-color);
	line-height: 1.8;
	margin: 10px 0;
}

.update-html :deep(ul),
.update-html :deep(ol) {
	padding-left: 18px;
	color: var(--fg-secondary-color);
	line-height: 1.8;
	margin: 10px 0;
}

.update-html :deep(code) {
	background: color-mix(in srgb, var(--fg-default-color) 6%, transparent);
	padding: 2px 6px;
	border-radius: 8px;
	color: var(--fg-default-color);
}

.update-html :deep(pre) {
	background: #0f172a;
	color: #dbeafe;
	padding: 12px 14px;
	border-radius: 14px;
	overflow: auto;
	line-height: 1.7;
	margin: 12px 0;
}

.update-html :deep(a) {
	color: var(--primary-color);
	text-decoration: underline;
}

.update-html :deep(hr) {
	border: none;
	border-top: 1px solid rgba(148, 163, 184, 0.35);
	margin: 16px 0;
}

.update-html :deep(blockquote) {
	margin: 12px 0;
	padding: 12px 14px;
	border-left: 3px solid color-mix(in srgb, var(--primary-color) 50%, transparent);
	background: color-mix(in srgb, var(--primary-color) 12%, var(--bg-secondary-color));
	border-radius: 14px;
	color: var(--fg-secondary-color);
}

.update-html :deep(table) {
	width: 100%;
	border-collapse: separate;
	border-spacing: 0;
	border: 1px solid var(--border-color);
	border-radius: 14px;
	overflow: hidden;
	margin: 14px 0;
}

.update-html :deep(th),
.update-html :deep(td) {
	padding: 10px 12px;
	border-bottom: 1px solid var(--border-color);
	color: var(--fg-secondary-color);
}

.update-html :deep(th) {
	background: var(--bg-secondary-color);
	font-weight: 700;
}

.update-html :deep(tr:last-child td) {
	border-bottom: none;
}

.update-html :deep(img) {
	max-width: 100%;
	height: auto;
	border-radius: 14px;
	margin: 12px 0;
}

:deep(.n-descriptions) {
	border-radius: 22px;
	overflow: hidden;
}

@media (max-width: 768px) {
	.update-actions {
		width: 100%;
		justify-content: flex-end;
	}
}
</style>

<style>
.theme-dark .update-page-root .text-slate-500 {
	color: var(--fg-secondary-color) !important;
}
.theme-dark .update-page-root .text-slate-400 {
	color: var(--fg-secondary-color) !important;
}
.theme-dark .update-page-root .border-slate-200 {
	border-color: color-mix(in srgb, var(--border-color) 80%, transparent) !important;
}
.theme-dark .update-page-root .bg-slate-50\/80 {
	background-color: color-mix(in srgb, var(--bg-default-color) 80%, transparent) !important;
}
.theme-dark .update-page-root .bg-white {
	background-color: var(--bg-default-color) !important;
}
</style>
