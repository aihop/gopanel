<template>
  <n-drawer
    v-model:show="open"
    :mask-closable="false"
    :destroy-on-close="true"
    width="675"
  >
    <n-drawer-content>
      <template #header>
        <DrawerHeader
          :header="title"
          :back="handleClose"
        />
      </template>

      <div style="padding: 16px">
        <n-form
          ref="fileForm"
          :model="addForm"
          :rules="rules"
          label-placement="top"
        >
          <n-form-item
            :label="$t('file.path')"
            path="newPath"
          >
            <n-input v-model:value="addForm.newPath">
              <template #prefix></template>
            </n-input>
          </n-form-item>

          <div v-if="changeName">
            <n-form-item
              :label="$t('commons.table.name')"
              path="name"
            >
              <n-input
                v-model:value="addForm.name"
                :disabled="addForm.cover"
              />
            </n-form-item>

            <n-radio-group
              v-model:value="addForm.cover"
              @update:value="changeType"
            >
              <n-space>
                <n-radio :value="true">{{ $t("file.replace") }}</n-radio>
                <n-radio :value="false">{{ $t("file.rename") }}</n-radio>
              </n-space>
            </n-radio-group>
          </div>

          <div
            v-if="existFiles.length > 0 && !changeName"
            style="text-align: center; margin-top: 12px"
          >
            <n-alert
              :title="$t('file.existFileDirHelper')"
              type="warning"
              :show-icon="true"
            />
            <n-transfer
              style="margin-top: 12px; display: block"
              :data="transferData"
              v-model:value="skipFiles"
              :titles="[$t('commons.button.cover'), $t('commons.button.skip')]"
              keys-field="key"
              label-field="label"
            />
          </div>
        </n-form>
      </div>

      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px; padding: 12px">
          <n-button
            @click="handleClose(false)"
            :disabled="loading"
          >
            {{ $t("commons.button.cancel") }}
          </n-button>
          <n-button
            type="primary"
            @click="submit"
            :loading="loading"
          >
            {{ $t("commons.button.confirm") }}
          </n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import { ref, reactive, computed } from "vue"
import { useI18n } from "vue-i18n"
import { NDrawer, NForm, NFormItem, NInput, NRadioGroup, NRadio, NAlert, NTransfer, NButton, NSpace } from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { BatchCheckFiles, CheckFile, MoveFile } from "@/api/modules/file"
import { Rules } from "@/global/form-rules"
import { MsgSuccess } from "@/utils/message"
import { getDateStr } from "@/utils/util"
import type { FormRules } from "naive-ui"

interface MoveProps {
	oldPaths: Array<string>
	allNames: Array<string>
	type: string
	path: string
	name: string
	isDir: boolean
}

const { t } = useI18n()

const fileForm = ref<any | null>(null)
const loading = ref(false)
const open = ref(false)
const type = ref<"cut" | "copy">("cut")
const changeName = ref(false)
const oldName = ref("")
const existFiles = ref<any[]>([])
const skipFiles = ref<string[]>([])
const transferData = ref<Array<{ key: string; label: string }>>([])

const title = computed(() => (type.value === "cut" ? t("file.move") : t("file.copy")) as string)

const addForm = reactive({
	oldPaths: [] as string[],
	newPath: "",
	type: "",
	name: "",
	allNames: [] as string[],
	isDir: false,
	cover: false
})

const rules = reactive<FormRules>({
	newPath: [Rules.requiredInput],
	name: [Rules.requiredInput]
})

const em = defineEmits(["close"])

const handleClose = (search = true) => {
	open.value = false
	fileForm.value?.resetValidation?.()
	em("close", search)
}

const getFileName = (filePath: string) => {
	let p = filePath
	if (p.endsWith("/")) p = p.slice(0, -1)
	return p.split("/").pop() || ""
}

const coverFiles = computed(() =>
	addForm.oldPaths.filter(item => !skipFiles.value.includes(getFileName(item))).map(item => item)
)

const getPath = (path: string) => {
	addForm.newPath = path
}

const changeType = () => {
	if (addForm.cover) {
		addForm.name = oldName.value
	} else {
		addForm.name = renameFileWithSuffix(oldName.value, addForm.isDir)
	}
}

const mvFile = async () => {
	loading.value = true
	try {
		await MoveFile({ ...addForm })
		if (type.value === "cut") {
			MsgSuccess(t("file.moveSuccess") as string)
		} else {
			MsgSuccess(t("file.copySuccess") as string)
		}
		handleClose(true)
	} finally {
		loading.value = false
	}
}

const submit = async () => {
	try {
		await fileForm.value?.validate?.()
	} catch {
		return
	}
	loading.value = true
	addForm.oldPaths = coverFiles.value
	await mvFile()
}

const getCompleteExtension = (filename: string): string => {
	const compoundExtensions = [
		".tar.gz",
		".tar.bz2",
		".tar.xz",
		".tar.lzma",
		".tar.Z",
		".tar.zst",
		".tar.lzo",
		".tar.sz",
		".tgz",
		".tbz2",
		".txz",
		".tzst"
	]
	const foundExtension = compoundExtensions.find(ext => filename.endsWith(ext))
	if (foundExtension) return foundExtension
	const match = filename.match(/\.[a-zA-Z0-9]+$/)
	return match ? match[0] : ""
}

const renameFileWithSuffix = (fileName: string, isDir: boolean): string => {
	const insertStr = "-" + getDateStr()
	const completeExt = isDir ? "" : getCompleteExtension(fileName)
	if (!completeExt) {
		return `${fileName}${insertStr}`
	} else {
		const baseName = fileName.slice(0, fileName.length - completeExt.length)
		return `${baseName}${insertStr}${completeExt}`
	}
}

const handleFilePaths = async (fileNames: string[], newPath: string) => {
	const uniqueFiles = [...new Set(fileNames)]
	const fileNamesWithPath = uniqueFiles.map(file => newPath + "/" + file)
	const existData = await BatchCheckFiles(fileNamesWithPath)
	existFiles.value = existData.data || []
	transferData.value = existFiles.value.map((file: any) => ({
		key: file.name,
		label: file.name
	}))
	// 默认不跳过任何，skipFiles 表示右侧选中（skip），保持为空表示全都 cover
	skipFiles.value = []
}

const acceptParams = async (props: MoveProps) => {
	changeName.value = false
	addForm.oldPaths = props.oldPaths
	addForm.type = props.type
	addForm.newPath = props.path
	addForm.isDir = props.isDir
	addForm.name = ""
	addForm.allNames = props.allNames
	type.value = props.type as any

	if (props.name && props.name !== "") {
		oldName.value = props.name
		const res = await CheckFile(props.path + "/" + props.name)
		if (res.data) {
			changeName.value = true
			addForm.cover = false
			addForm.name = renameFileWithSuffix(props.name, addForm.isDir)
			open.value = true
			return
		} else {
			// no conflict, move directly
			await mvFile()
			return
		}
	} else if (props.allNames && props.allNames.length > 0) {
		await handleFilePaths(addForm.allNames, addForm.newPath)
		if (existFiles.value.length > 0) {
			changeName.value = false
			open.value = true
			return
		} else {
			await mvFile()
			return
		}
	} else {
		await mvFile()
	}
}

defineExpose({ acceptParams })
</script>
