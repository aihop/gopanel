<template>
  <n-drawer
    v-model:show="open"
    placement="right"
    width="40vw"
    :mask-closable="false"
    :close-on-esc="false"
    @close="handleClose"
  >
    <n-drawer-content>
      <template #header>
        <DrawerHeader
          :header="$t('commons.button.create')"
          :back="handleClose"
        />
      </template>
      <n-grid :cols="1">
        <n-grid-item>
          <n-form
            ref="fileForm"
            :model="addForm"
            :rules="rules"
            label-placement="top"
            :show-require-mark="true"
            :loading="loading"
          >
            <n-form-item
              :label="$t('commons.table.name')"
              path="name"
            >
              <n-input v-model:value="addForm.name" />
            </n-form-item>
            <n-form-item v-if="!addForm.isDir">
              <n-checkbox v-model="addForm.isLink">
                {{ $t("file.link") }}
              </n-checkbox>
            </n-form-item>
            <n-form-item
              v-if="addForm.isLink"
              :label="$t('file.linkType')"
              path="linkType"
            >
              <n-radio-group v-model="addForm.isSymlink">
                <n-radio :value="true">
                  {{ $t("file.softLink") }}
                </n-radio>
                <n-radio :value="false">
                  {{ $t("file.hardLink") }}
                </n-radio>
              </n-radio-group>
            </n-form-item>
            <n-form-item
              v-if="addForm.isLink"
              :label="$t('file.linkPath')"
              path="linkPath"
            >
              <n-input v-model:value="addForm.linkPath">
                <template #prefix></template>
              </n-input>
            </n-form-item>
            <n-form-item>
              <n-checkbox
                v-if="addForm.isDir"
                v-model="setRole"
              >
                {{ $t("file.editPermissions") }}
              </n-checkbox>
            </n-form-item>
          </n-form>
        </n-grid-item>
      </n-grid>
      <template #footer>
        <n-space>
          <n-button @click="handleClose">{{ $t("commons.button.cancel") }}</n-button>
          <n-button
            type="primary"
            @click="submit(fileForm)"
          >{{ $t("commons.button.confirm") }}</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import DrawerHeader from "@/components/DrawerHeader.vue"
import { ref, reactive, computed } from "vue"
import type { File } from "@/api/interface/file"
import { CreateFile } from "@/api/modules/file"
import { Rules } from "@/global/form-rules"
import { MsgSuccess, MsgWarning } from "@/utils/message"
import { useI18n } from "vue-i18n"
const { t } = useI18n()

const fileForm = ref()
let loading = ref(false)
let setRole = ref(false)

interface CreateProps {
	file: Partial<File.FileCreate>
}
const propData = ref<CreateProps>({
	file: {}
})

let addForm = reactive({ path: "", name: "", isDir: false, mode: 0o755, isLink: false, isSymlink: true, linkPath: "" })
let open = ref(false)
const em = defineEmits(["close"])
const handleClose = () => {
	open.value = false
	em("close", open)
}

const rules = reactive({
	name: [Rules.requiredInput, Rules.linuxName],
	path: [Rules.requiredInput],
	isSymlink: [Rules.requiredInput],
	linkPath: [Rules.requiredInput]
})

const getMode = (val: number) => {
	addForm.mode = val
}

let getPath = computed(() => {
	if (addForm.path.endsWith("/")) {
		return addForm.path + addForm.name.trim()
	} else {
		return addForm.path + "/" + addForm.name.trim()
	}
})

const getLinkPath = (path: string) => {
	addForm.linkPath = path
}

const submit = async (formEl: any) => {
	if (!formEl) return
	try {
		await formEl.validate()
	} catch (_error: unknown) {
		return
	}
	if (getPath.value.indexOf(".gopanel_clash") > -1) {
		MsgWarning(t("file.clashDitNotSupport"))
		return
	}

	const addItem: Partial<File.FileCreate> = {
		...addForm,
		path: getPath.value,
		name: addForm.name.trim()
	}
	loading.value = true
	if (!setRole.value) {
		addItem.mode = undefined
	}
	CreateFile(addItem as File.FileCreate)
		.then(() => {
			MsgSuccess(t("commons.msg.createSuccess"))
			handleClose()
		})
		.finally(() => {
			loading.value = false
		})
}

const acceptParams = (create: File.FileCreate) => {
	propData.value.file = create
	open.value = true
	addForm.isDir = create.isDir
	addForm.path = create.path
	addForm.name = ""
	addForm.isLink = false
	init()
}

const init = () => {
	setRole.value = false
}

defineExpose({ acceptParams })
</script>
