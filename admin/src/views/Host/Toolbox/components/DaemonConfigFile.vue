<template>
  <n-card
    :bordered="false"
    size="large"
    class="shadow"
    title="守护进程配置文件"
  >
    <FtEditor
      v-model="content"
      language="ini"
      height="400px"
    />

    <n-space class="mt-4 flex justify-end">
      <n-button
        type="primary"
        :loading="saving"
        @click="onSave"
      >保存</n-button>
      <n-button
        class="ml-2"
        @click="onReload"
      >重新加载</n-button>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { NCard, NForm, NFormItem, NButton, useMessage } from "naive-ui"
import FtEditor from "@/components/FtEditor/index.vue"
import { DaemonConfigFileLoad, DaemonConfigFileUpdate, DaemonReload } from "@/api/modules/daemon"

const message = useMessage()
const content = ref("")
const saving = ref(false)

const loadConfig = async () => {
	try {
		const res = await DaemonConfigFileLoad()
		if (res.data) {
			content.value = res.data
		}
	} catch {
		message.error("加载配置失败")
	}
}

const onSave = async () => {
	saving.value = true
	try {
		await DaemonConfigFileUpdate({ content: content.value })
		await DaemonReload()
		message.success("保存成功")
	} catch {
		 
	}
	saving.value = false
}

const onReload = () => {
	loadConfig()
	message.success("已重新加载")
}

onMounted(loadConfig)
</script>
