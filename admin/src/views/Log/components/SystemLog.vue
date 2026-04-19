<template>
  <div class="system-log-wrap flex flex-col relative h-[calc(100vh-150px)]">
    <n-space class="mb-4 flex items-center">
      <span class="mr-4 ">日志文件:</span>
      <n-select
        v-model:value="currentFile"
        :options="fileOptions"
        placeholder="选择日志文件"
        class="w-64 mr-4"
        @update:value="loadSystemLog"
      />
      <n-button
        type="primary"
        @click="loadSystemLog"
        :loading="loading"
      >
        刷新
      </n-button>
    </n-space>

    <div
      class="flex-1 bg-[#1e1e1e] p-4 rounded-md text-[#d4d4d4] font-mono text-sm leading-relaxed overflow-auto"
      ref="terminalRef"
    >
      <div
        v-if="loading"
        class="text-gray-400"
      >正在读取日志...</div>
      <div
        v-else-if="!logContent"
        class="text-gray-400"
      >暂无日志内容</div>
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

const message = useMessage()
const loading = ref(false)
const currentFile = ref("")
const fileOptions = ref<{ label: string; value: string }[]>([])
const logContent = ref("")
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
	} catch (error) {
	}
}

const loadSystemLog = async () => {
	if (!currentFile.value) return
	loading.value = true
	try {
		const res = await getSystemLogs(currentFile.value)
		logContent.value = res.data || ""
		nextTick(() => {
			if (terminalRef.value) {
				terminalRef.value.scrollTop = terminalRef.value.scrollHeight
			}
		})
	} catch (error) {
		message.error("获取日志内容失败")
	} finally {
		loading.value = false
	}
}

onMounted(() => {
	loadFiles()
})
</script>
