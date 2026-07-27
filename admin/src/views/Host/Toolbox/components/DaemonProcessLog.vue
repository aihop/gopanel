<template>
  <NDrawer
    v-model:show="logDrawerVisible"
    :width="700"
    :mask-closable="false"
  >
    <NDrawerContent
      :title="logTitle"
      closable
    >
      <template #header>
        <div class="flex items-center gap-4">
          <n-button
            text
            @click="close"
          >
            <template #icon>
              <Icon name="mdi:arrow-left" />
            </template>
            返回
          </n-button>
          <n-divider vertical />
          <div>{{ logTitle }}</div>
          <div class="ml-12"><n-button @click="cleanLog">清空日志</n-button></div>
        </div>
      </template>
      <FtEditor
        v-model="logContent"
        :loading="logLoading"
        style="font-family: monospace"
        placeholder="暂无日志内容"
        height="calc(100vh - 150px)"
      />
      <div class="mt-4 flex justify-center">
        <n-pagination
          v-if="logTotal > logPageSize"
          :page="logPage"
          :page-size="logPageSize"
          :page-count="Math.ceil(logTotal / logPageSize)"
          @update:page="
						p => {
							logPage = p
							fetchLogPage()
						}
					"
          simple
        />
      </div>
    </NDrawerContent>
  </NDrawer>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { NButton, NTag, NDrawer, NDrawerContent, NInput, useMessage } from "naive-ui"
import { DaemonProcessLog, DaemonProcessLogClearAPI } from "@/api/modules/daemon"
import FtEditor from "@/components/FtEditor/index.vue"

const message = useMessage()
const logDrawerVisible = ref(false)
const logContent = ref("")
const logLoading = ref(false)
const logTitle = ref("")
const logPage = ref(1)
const logPageSize = 10240 // 10KB
const logTotal = ref(0)
const logName = ref("")

const handleShowLog = async (row: any) => {
	logDrawerVisible.value = true
	logTitle.value = row.name + " 日志"
	logContent.value = ""
	logPage.value = 1
	logName.value = row.name
	await fetchLogPage()
}

const fetchLogPage = async () => {
	logLoading.value = true
	try {
		const offset = (logPage.value - 1) * logPageSize
		const res = await DaemonProcessLog({ name: logName.value, offset, length: logPageSize })
		if (res.data && typeof res.data.logData === "string") {
			logContent.value = res.data.logData
			logTotal.value = res.data.logSize || 0 // 每次都刷新logTotal
		} else {
			logContent.value = "暂无日志内容"
			logTotal.value = 0
		}
	} catch {
		logContent.value = "日志加载失败"
		logTotal.value = 0
	}
	logLoading.value = false
}

const open = (record?: any) => {
	logDrawerVisible.value = true
	logTitle.value = record.name + " 日志"
	logContent.value = ""
	logPage.value = 1
	logName.value = record.name
	fetchLogPage()
}

const close = () => {
	logDrawerVisible.value = false
}

const cleanLog = async () => {
	try {
		await DaemonProcessLogClearAPI({ name: logName.value })
		logContent.value = ""
		logTotal.value = 0
		logPage.value = 1
		message.success("日志已清空")
		// 回源确认服务端已清空（避免本地清了但服务端没清的假象）
		await fetchLogPage()
	} catch {
		message.error("日志清空失败")
	}
}
defineExpose({
	open,
	close
})
</script>
