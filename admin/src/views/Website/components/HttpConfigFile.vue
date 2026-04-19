lu
<template>
  <FtEditor
    v-model="content"
    language="ini"
    height="400px"
  />
  <n-space class="mt-6">
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
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { NButton, useMessage } from "naive-ui"
import FtEditor from "@/components/FtEditor/index.vue"
import { httpDefaultConfigAPI, httpDefaultUpdateAPI } from "@/api/modules/http"

const message = useMessage()
const content = ref("")
const saving = ref(false)

const loadConfig = async () => {
	try {
		const res = await httpDefaultConfigAPI()
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
		await httpDefaultUpdateAPI({ content: content.value })
		message.success("保存成功")
	} catch {
		// message.error("保存失败")
	}
	saving.value = false
}

const onReload = () => {
	loadConfig()
	message.success("已重新加载")
}

onMounted(loadConfig)
</script>
