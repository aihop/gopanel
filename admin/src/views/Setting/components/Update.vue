<template>
  <SystemUpdateProvider
    v-slot="u"
    class="mt-4"
  >
    <div class="update-page-root">
      <n-space
        vertical
        size="large"
      >
        <div
          size="small"
          :bordered="false"
          class="rounded-[28px] bg-base-accent border-base-accent p-8 shadow-[0_18px_48px_rgba(15,23,42,0.08)]"
        >
          <div class="flex items-start justify-between gap-4">
            <div>
              <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
                Version Center
              </div>
              <div class="my-3 text-2xl font-semibold fg-base-100">版本信息</div>
            </div>
            <n-tag
              type="info"
              round
              :bordered="false"
            >当前运行环境</n-tag>
          </div>

          <n-descriptions
            :column="4"
            bordered
            class="mt-2"
          >
            <n-descriptions-item label="当前版本">
              {{ u.versionInfo.versionName || "-" }}
            </n-descriptions-item>
            <n-descriptions-item label="当前版本代码">
              {{ u.versionInfo.versionCode || "-" }}
            </n-descriptions-item>
            <n-descriptions-item label="构建时间">
              {{ u.formatTime(u.versionInfo.buildTime) || "-" }}
            </n-descriptions-item>
            <n-descriptions-item label="安装路径">
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
              <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Update Center</div>
              <div class="my-3 text-2xl font-semibold fg-base-100">
                当前已是最新版本
              </div>
              <div class="my-3 text-sm leading-7 text-slate-500">
                当前版本代号不小于最新版本代号，无需更新。如需确认可手动重新检查。
              </div>
            </div>
            <n-tag
              type="success"
              round
              :bordered="false"
            >
              已最新
            </n-tag>
          </div>
          <div class="flex justify-end">
            <n-button
              ghost
              class="!rounded-[18px]"
              :loading="u.checkingUpdate"
              @click="handleCheckUpdate(u)"
            >
              重新检查
            </n-button>
          </div>
        </div>

        <div
          v-else
          class="rounded-[28px] p-8 shadow-[0_18px_48px_rgba(15,23,42,0.08)]"
        >
          <div class="flex items-start justify-between gap-4">
            <div>
              <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Update Center</div>
              <div class="my-3 text-2xl font-semibold fg-base-100">
                {{ u.updateInfo.title || "系统更新" }}
              </div>
              <div class="my-3 text-sm leading-7 text-slate-500">
                {{ u.updateInfo.description || "快速判断是否存在新版本，并在升级时查看实时日志" }}
              </div>
            </div>
            <n-tag
              v-if="u.effectiveNeedUpdate !== undefined"
              :type="u.effectiveNeedUpdate === true ? 'warning' : 'info'"
              round
              :bordered="false"
            >
              {{ u.effectiveNeedUpdate === true ? "可升级" : "检查中" }}
            </n-tag>
          </div>

          <div class="update-section">
            <div class="update-info">
              <div class="grid gap-4 sm:grid-cols-3">
                <div class="rounded-[22px] border border-slate-200 bg-slate-50/80 p-5">
                  <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
                    最新版本
                  </div>
                  <div class="mt-3 text-xl font-semibold fg-base-100">
                    {{ u.updateInfo.latestVersionName || "检查中..." }}
                  </div>
                </div>
                <div class="rounded-[22px] border border-slate-200 bg-slate-50/80 p-5">
                  <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
                    版本代码
                  </div>
                  <div class="mt-3 text-xl font-semibold fg-base-100">
                    {{ u.updateInfo.latestVersionCode || "检查中..." }}
                  </div>
                </div>
                <div class="rounded-[22px] border border-slate-200 bg-slate-50/80 p-5">
                  <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
                    更新状态
                  </div>
                  <div class="mt-3 text-xl font-semibold fg-base-100">
                    {{ u.effectiveNeedUpdate === true ? "建议升级" : "检查中..." }}
                  </div>
                </div>
              </div>

              <n-descriptions
                class="mt-5"
                bordered
                :column="3"
                label-placement="top"
              >
                <n-descriptions-item label="当前版本">
                  {{ u.versionInfo.versionName || u.updateInfo.curVersion || "-" }}
                </n-descriptions-item>
                <n-descriptions-item label="发布时间">
                  {{ u.formatTime(u.updateInfo.createAt || "") || "-" }}
                </n-descriptions-item>
                <n-descriptions-item label="下载地址">
                  <a
                    v-if="u.updateInfo.downloadUrl"
                    class="text-blue-600 hover:underline break-all"
                    :href="u.updateInfo.downloadUrl"
                    target="_blank"
                    rel="noreferrer"
                  >{{ u.updateInfo.downloadUrl }}</a>
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
                重新检查
              </n-button>
              <n-button
                type="primary"
                class="!rounded-[18px] shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
                :loading="u.updating"
                :disabled="u.effectiveNeedUpdate !== true"
                @click="handleUpdate(u)"
              >
                立即更新
              </n-button>
            </div>
          </div>

          <div
            v-if="u.effectiveNeedUpdate === true && u.updateInfo.content"
            class="mt-6 rounded-[22px] border border-slate-200 bg-white p-5"
          >
            <div class="text-xs font-semibold uppercase tracking-[0.16em] text-slate-400">
              更新内容
            </div>
            <div
              class="update-html mt-3"
              v-html="u.updateInfo.content"
            />
          </div>
        </div>
      </n-space>

      <n-modal
        :show="u.logVisible"
        :mask-closable="!u.isReading"
        :closable="!u.isReading"
        preset="card"
        title="更新日志"
        style="width: 80%; max-width: 1000px"
        :show-close="!u.isReading"
        @update:show="(val) => u.setLogVisible(val)"
      >
        <div class="space-y-4">
          <div class="flex items-center justify-between gap-3 rounded-[18px] border border-slate-200 bg-slate-50/80 px-4 py-3">
            <div class="text-sm text-slate-500">
              {{ u.logStatusText }}
            </div>
            <n-tag
              :type="u.logStatusTag"
              round
              :bordered="false"
            >
              {{ u.logStatusLabel }}
            </n-tag>
          </div>
          <div
            :ref="u.setTerminalRef"
            class="update-log-terminal"
          >
            <div
              v-for="(line, idx) in u.streamLogs"
              :key="idx"
              class="whitespace-pre-wrap break-words"
            >
              {{ line }}
            </div>
            <div
              v-if="u.streamLogs.length === 0"
              class="text-slate-500 italic"
            >
              正在连接更新日志流...
            </div>
          </div>
        </div>
        <template #footer>
          <div class="flex justify-end gap-2">
            <n-button
              v-if="!u.isReading"
              type="primary"
              @click="u.setLogVisible(false)"
            >关闭</n-button>
            <n-button
              v-else
              disabled
            >更新中，请稍候...</n-button>
          </div>
        </template>
      </n-modal>
    </div>
  </SystemUpdateProvider>
</template>

<script setup lang="ts">
import { useDialog, useMessage } from "naive-ui"
import SystemUpdateProvider from "@/components/system/SystemUpdateProvider.vue"

const message = useMessage()
const dialog = useDialog()

const handleCheckUpdate = async (u: any) => {
	await u.checkUpdate()
	if (u.updateInfo.needUpdate) {
		message.warning("发现新版本，建议及时更新")
	} else {
		message.success("当前已是最新版本")
	}
}

const handleUpdate = async (u: any) => {
	if (!u.updateInfo.needUpdate) {
		message.warning("当前已是最新版本")
		return
	}

	dialog.warning({
		title: "更新确认",
		content: `确定要更新到版本 ${u.updateInfo.latestVersionName} 吗？更新过程中系统可能会短暂不可用。`,
		positiveText: "确定更新",
		negativeText: "取消",
		onPositiveClick: async () => {
			const res = await u.startUpgrade({
				containerName: "gopanel",
				currentVersion: u.versionInfo.versionName,
				targetVersion: u.updateInfo.latestVersionName
			})
			if (res.ok) {
				message.success("更新已开始，请稍候...")
			} else {
				message.error(res.msg || "更新失败")
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
	font-family: ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
		"Courier New", monospace;
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
