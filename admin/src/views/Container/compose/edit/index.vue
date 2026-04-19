<template>
  <n-drawer
    v-model:show="composeVisible"
    :width="800"
    :mask-closable="false"
    placement="right"
  >
    <n-drawer-content
      title="编辑编排"
      closable
    >
      <n-form
        ref="formRef"
        :model="form"
      >
        <n-form-item
          label="Compose 内容"
          path="content"
        >
          <FtEditor
            v-model="form.content"
            language="yaml"
            height="calc(100vh - 200px)"
          />
        </n-form-item>
        <n-form-item
          v-if="form.createdBy === 'GoPanel'"
          label="环境变量"
          path="environmentStr"
        >
          <n-input
            v-model:value="form.environmentStr"
            type="textarea"
            placeholder="一行一个, 例: key1=value1"
            :autosize="{ minRows: 3, maxRows: 8 }"
          />
        </n-form-item>
        <n-form-item
          v-if="form.createdBy === 'GoPanel'"
          label="env_file 内容"
        >
          <FtEditor
            v-model="form.envFileContent"
            language="yaml"
            :readonly="true"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="composeVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="loading"
            @click="onSubmitEdit"
          >确定</n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { ref, reactive } from "vue"
import { NDrawer, NDrawerContent, NForm, NFormItem, NInput, NButton, useMessage } from "naive-ui"
import FtEditor from "@/components/FtEditor/index.vue"
import { composeUpdate, loadContainerLog } from "@/api/modules/container"

const emit = defineEmits<{ (e: "search"): void }>()
const message = useMessage()
const loading = ref(false)
const composeVisible = ref(false)
const formRef = ref()
const form = reactive({
	name: "",
	path: "",
	content: "",
	environmentStr: "",
	createdBy: "",
	envFileContent: "env_file:\n  - gopanel.env"
})

const onSubmitEdit = async () => {
	const param: any = {
		name: form.name,
		path: form.path,
		content: form.content,
		createdBy: form.createdBy
	}
	if (form.environmentStr !== undefined && form.createdBy === "GoPanel") {
		param.env = form.environmentStr.split("\n")
	}
	loading.value = true
	try {
		await composeUpdate(param)
		loading.value = false
		message.success("操作成功")
		composeVisible.value = false
		emit("search")
	} catch {
		loading.value = false
	}
}

interface DialogProps {
	name: string
	path: string
	content: string
	env: Array<string>
	envStr: string
	createdBy: string
}

const acceptParams = async (props: DialogProps): Promise<void> => {
	composeVisible.value = true
	form.name = props.name
	form.path = props.path
	form.createdBy = props.createdBy
	form.environmentStr = (props.env || []).join("\n")
	// 获取compose内容
	try {
		const res = await loadContainerLog("compose-detail", props.name)
		if (res.data) {
			form.content = res.data
		} else {
			form.content = props.content
		}
	} catch {
		form.content = props.content
	}
}
const handleClose = () => {
	composeVisible.value = false
}

defineExpose({ acceptParams })
</script>
