<template>
  <n-drawer
    v-model:show="visible"
    :width="502"
  >
    <n-drawer-content :title="title">
      <template #header>
        <div class="flex items-center gap-2">
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
          {{ title }}
        </div>
      </template>
      <n-form-item :label="title">
        <n-input
          v-if="inputKey === 'registryMirrors'"
          v-model:value="inputVal"
          type="textarea"
          placeholder="当存在多个加速器时，需要换行显示，例：
  http://xxxxxx.m.daocloud.io
  https://xxxxxx.mirror.aliyuncs.com"
          :autosize="{ minRows: 5, maxRows: 10 }"
        />
      </n-form-item>
      <template #footer>
        <n-button @click="close">取消</n-button>
        <n-button
          type="primary"
          :loading="loading"
          style="margin-left: 10px"
          @click="save"
        >确认</n-button>
      </template>
    </n-drawer-content>
  </n-drawer>
  <RebootAlert
    ref="RebootAlertModel"
    @confirm="alertConfirm"
  />
</template>

<script setup lang="ts">
import { updateDaemonUpdate } from "@/api/modules/container"
import RebootAlert from "@/components/RebootAlert.vue"
import { MsgSuccess } from "@/utils/message"
import { ref } from "vue"

const emit = defineEmits(["save"])

const visible = ref(false)
const title = ref("")
const inputVal = ref("")
const inputKey = ref("")
const loading = ref(false)
const titles = {
	registryMirrors: "镜像加速",
	insecureRegistries: "私有仓库"
}
function open(data: string[] | string | undefined, key: "registryMirrors" | "insecureRegistries") {
	visible.value = true
	inputKey.value = key
	title.value = titles[key]

	// 兼容多种 data 形式：数组 / 逗号分隔字符串 / 单一字符串 / undefined
	if (Array.isArray(data)) {
		inputVal.value = data.join("\n")
	} else if (typeof data === "string") {
		// 如果是逗号分隔的值，拆成多行；否则直接显示
		if ((data as string).includes(",")) {
			inputVal.value = (data as string)
				.split(",")
				.map(v => v.trim())
				.filter(Boolean)
				.join("\n")
		} else {
			inputVal.value = data
		}
	} else {
		inputVal.value = ""
	}
}
function close() {
	visible.value = false
}
defineExpose({ open, close })
const RebootAlertModel = ref()

function save() {
	RebootAlertModel.value.open({
		title: `配置修改`,
		input: "立即重启",
		msg: `修改配置后需要重启生效。`
	})
}

async function alertConfirm() {
	RebootAlertModel.value.close()
	loading.value = true
	try {
		let value = inputVal.value
		if (inputKey.value === "registryMirrors" || inputKey.value === "insecureRegistries") {
			value = value
				.split("\n")
				.map(v => v.trim())
				.filter(v => v)
				.join(",")
		}
		const keyMap = {
			registryMirrors: "Mirrors",
			insecureRegistries: "Registries"
		}
		const res = await updateDaemonUpdate(keyMap[inputKey.value], value)
		if (res && res.code === 0) {
			MsgSuccess("保存成功，Docker正在重启...")
			visible.value = false
			emit("save") // 通知父组件刷新
		} else {
			MsgSuccess(res.msg || "保存失败")
		}
	} catch (e) {
		MsgSuccess("保存异常")
	} finally {
		loading.value = false
	}
}
</script>
