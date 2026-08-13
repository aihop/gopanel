<template>
  <div class="system-log-wrap flex flex-col relative h-[calc(100vh-150px)]">
    <n-space class="mb-4 flex items-center">
	  <span class="mr-4">{{ t("securityMonitoring.systemLogFile") }}:</span>
      <n-select
        v-model:value="currentFile"
        :options="fileOptions"
		:placeholder="t('securityMonitoring.selectSystemLog')"
        class="w-64 mr-4"
        @update:value="loadSystemLog"
      />
      <n-button
        type="primary"
        @click="loadSystemLog"
        :loading="loading"
      >
		{{ t("securityMonitoring.refresh") }}
      </n-button>
    </n-space>

    <div
      class="flex-1 bg-[#1e1e1e] p-4 rounded-md text-[#d4d4d4] font-mono text-sm leading-relaxed overflow-auto"
      ref="terminalRef"
    >
	  <div v-if="truncated" class="mb-3 rounded bg-amber-900/40 px-3 py-2 text-amber-200">
		{{ t("securityMonitoring.logTailNotice", { returned: returnedBytes, size: fileSize }) }}
	  </div>
      <div
        v-if="loading"
        class="text-gray-400"
	  >{{ t("securityMonitoring.readingLog") }}</div>
      <div
        v-else-if="!logContent"
        class="text-gray-400"
	  >{{ t("securityMonitoring.emptyLog") }}</div>
      <div
        v-else
        class="whitespace-pre-wrap break-all"
      >{{ logContent }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from "vue"
import { getSystemFiles, getSystemLogs } from "@/api/modules/log"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"

const message = useMessage()
const { t } = useI18n()
const loading = ref(false)
const currentFile = ref("")
const fileOptions = ref<{ label: string; value: string }[]>([])
const logContent = ref("")
const truncated = ref(false)
const fileSize = ref(0)
const returnedBytes = ref(0)
const terminalRef = ref<HTMLElement | null>(null)

const loadFiles = async () => {
	try {
		const res = await getSystemFiles()
		const files = res.data || []
		fileOptions.value = files.map((f: string) => ({ label: f, value: f }))
		if (files.length > 0) {
			currentFile.value = files[0]
			loadSystemLog()
		}
	} catch {
		message.error(t("securityMonitoring.logFilesLoadFailed"))
	}
}

const loadSystemLog = async () => {
	if (!currentFile.value) return
	loading.value = true
	try {
		const res = await getSystemLogs(currentFile.value)
		logContent.value = res.data?.content || ""
		truncated.value = !!res.data?.truncated
		fileSize.value = res.data?.size || 0
		returnedBytes.value = res.data?.returnedBytes || 0
		nextTick(() => {
			if (terminalRef.value) {
				terminalRef.value.scrollTop = terminalRef.value.scrollHeight
			}
		})
	} catch {
		message.error(t("securityMonitoring.logLoadFailed"))
	} finally {
		loading.value = false
	}
}

onMounted(() => {
	loadFiles()
})
</script>
