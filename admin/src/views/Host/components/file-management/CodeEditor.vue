<template>
  <!-- preset="card" -->
  <n-modal
    v-model:show="showModal"
    :mask-closable="false"
    :on-esc="handleBeforeClose"
  >
    <n-card :title="'编辑- ' + fileEdit?.path">
      <template #header-extra>
        <n-button
          @click="handleBeforeClose"
          quaternary
          size="small"
        >
          <template #icon>
            <n-icon>
              <Icon name="mdi:close" />
            </n-icon>
          </template>
        </n-button>
      </template>

      <FtEditor
        v-if="fileEdit"
        v-model="fileEdit.content"
        :language="langValue"
        height="400px"
        :key="langValue"
      />

      <template #footer>
        <div class="flex justify-end">
          <n-space>
            <n-button
              quaternary
              @click="handleBeforeClose"
            >取消</n-button>
            <n-button
              type="info"
              quaternary
              @click="saveFileWithoutParam"
            >保存</n-button>
          </n-space>
        </div>
      </template>
    </n-card>
  </n-modal>

  <n-modal
    v-model:show="showConfirmModal"
    preset="dialog"
    title="提示"
    positive-text="保存"
    negative-text="不保存"
    :on-positive-click="handleSaveAndClose"
    :on-negative-click="handleDiscardChanges"
  >
    文件已被修改，是否保存更改？
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { GetFileContent, SaveFileContent } from "@/api/modules/file"
import FtEditor from "@/components/FtEditor/index.vue"
import { Languages } from "@/global/mimetype"
import { MsgError, MsgSuccess } from "@/utils/message"
import { NButton, NSelect, NSpace, NModal } from "naive-ui"
import type { SelectOption } from "naive-ui"

const emit = defineEmits(["close"])

const showModal = ref(false)
const showConfirmModal = ref(false)
const originalContent = ref("")

const langValue = ref("json")
const langOptions = computed<Array<SelectOption>>(() => {
	return Languages.map(lang => ({
		label: lang.label,
		value: lang.label // 直接使用语言标签作为值
	}))
})

const openModal = () => {
	showModal.value = true
}

const closeModal = () => {
	showModal.value = false
	emit("close")
}

const handleBeforeClose = () => {
	if (fileEdit.value && fileEdit.value.content !== originalContent.value) {
		// 文件已被修改，显示确认对话框
		showConfirmModal.value = true
	} else {
		// 文件未修改，直接关闭
		closeModal()
	}
}

const handleDiscardChanges = () => {
	showConfirmModal.value = false
	closeModal()
}

const handleSaveAndClose = () => {
	showConfirmModal.value = false
	saveFile(true)
}

const path = ref<string>("")

type EditorFileParams = {
	path: string
	extension: string
}

const acceptParams = (row: EditorFileParams) => {
	path.value = row.path
	openCodeEditor(path.value, row.extension)
}

const fileEdit = ref<{
	path: string
	content: string
	name: string
	extension: string
} | null>(null)
const openCodeEditor = (path: string, extension: string) => {
	if (extension != "") {
		Languages.forEach(lang => {
			const ext = extension.substring(1)
			if (lang.value.indexOf(ext) > -1) {
				langValue.value = lang.label
			}
		})
	}

	const req = {
		path: path,
		expand: true,
		page: 1,
		limit: 10
	}

	GetFileContent(req)
		.then(res => {
			console.log(res)
			fileEdit.value = res.data
			// 保存原始内容用于比较
			originalContent.value = res.data.content
		})
		.catch(() => {
			closeModal()
		})
}

const saveFile = (closeAfterSave = false) => {
	// 保存文件逻辑
	if (!fileEdit.value) {
		MsgError("没有文件内容可保存")
		return
	}

	SaveFileContent({
		path: fileEdit.value.path,
		content: fileEdit.value.content
	})
		.then(() => {
			MsgSuccess("保存成功")
			// 更新原始内容
			originalContent.value = fileEdit.value!.content
			if (closeAfterSave) {
				closeModal()
			}
		})
		.catch(() => {
		})
}

const saveFileWithoutParam = () => {
	saveFile(false)
}

const handleKeydown = (event: KeyboardEvent) => {
	if (!showModal.value) {
		return
	}
	const isSave = event.key.toLowerCase() === "s" && (event.metaKey || event.ctrlKey)
	if (!isSave) {
		return
	}
	event.preventDefault()
	saveFileWithoutParam()
}

onMounted(() => {
	window.addEventListener("keydown", handleKeydown)
})

onBeforeUnmount(() => {
	window.removeEventListener("keydown", handleKeydown)
})

// 导出函数
defineExpose({
	openModal,
	closeModal,
	acceptParams
})
</script>
