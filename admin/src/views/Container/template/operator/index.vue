<template>
  <n-drawer
    v-model:show="drawerVisible"
    :destroy-on-close="true"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    size="50%"
  >
    <n-drawer-content>
      <template #header>
        <DrawerHeader
          :header="title + $t('container.composeTemplate').toLowerCase()"
          :hide-resource="dialogData.title === 'create'"
          :resource="dialogData.rowData?.name"
          :back="handleClose"
        />
      </template>
      <n-form
        ref="formRef"
        v-loading="loading"
        label-position="top"
        :model="dialogData.rowData"
        :rules="rules"
        label-width="80px"
      >
        <n-row
          type="flex"
          justify="center"
        >
          <n-col :span="22">
            <n-form-item
              :label="$t('commons.table.name')"
              prop="name"
            >
              <n-input
                v-model.trim="dialogData.rowData!.name"
                :disabled="dialogData.title === 'edit'"
              ></n-input>
            </n-form-item>
            <n-form-item :label="$t('container.description')">
              <n-input v-model:value="dialogData.rowData!.description"></n-input>
            </n-form-item>
            <n-form-item>
              <FtEditor
                v-model="dialogData.rowData!.content"
                language="yaml"
                height="calc(100vh - 351px)"
              />
            </n-form-item>
          </n-col>
        </n-row>
      </n-form>
      <template #footer>
        <span class="dialog-footer">
          <n-button
            :disabled="loading"
            @click="drawerVisible = false"
          >
            {{ $t("commons.button.cancel") }}
          </n-button>
          <n-button
            :disabled="loading"
            type="primary"
            @click="onSubmit(formRef)"
          >
            {{ $t("commons.button.confirm") }}
          </n-button>
        </span>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import type { Container } from "@/api/interface/container"
import { createComposeTemplate, updateComposeTemplate } from "@/api/modules/container"
import FtEditor from "@/components/FtEditor/index.vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { Rules } from "@/global/form-rules"
import { i18n } from "@/i18n"
import { MsgSuccess } from "@/utils/message"
import { reactive, ref } from "vue"
import { t } from "@/i18n"

const emit = defineEmits<{ (e: "search"): void }>()

const loading = ref(false)

interface DialogProps {
	title: string
	rowData?: Container.TemplateInfo
	getTableList?: () => Promise<any>
}
const title = ref<string>("")
const drawerVisible = ref(false)
const dialogData = ref<DialogProps>({
	title: ""
})
function acceptParams(params: DialogProps): void {
	dialogData.value = params
	title.value = t(`commons.button.${dialogData.value.title}`)
	drawerVisible.value = true
}
function handleClose() {
	drawerVisible.value = false
}

const rules = reactive({
	name: [Rules.requiredInput, Rules.name],
	content: [Rules.requiredInput]
})

const formRef = ref<any>()

async function onSubmit(formEl: any | undefined) {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		loading.value = true
		if (dialogData.value.title === "create") {
			await createComposeTemplate(dialogData.value.rowData!)
				.then(() => {
					loading.value = false
					MsgSuccess(t("commons.msg.operationSuccess"))
					emit("search")
					drawerVisible.value = false
				})
				.catch(() => {
					loading.value = false
				})
			return
		}
		await updateComposeTemplate(dialogData.value.rowData!)
			.then(() => {
				loading.value = false
				MsgSuccess(t("commons.msg.operationSuccess"))
				emit("search")
				drawerVisible.value = false
			})
			.catch(() => {
				loading.value = false
			})
	})
}

defineExpose({
	acceptParams
})
</script>
