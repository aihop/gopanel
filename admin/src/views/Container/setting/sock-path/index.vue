<template>
  <div>
    <n-drawer
      v-model:show="drawerVisible"
      :destroy-on-close="true"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      size="30%"
    >
      <n-drawer-content>
        <template #header>
          <DrawerHeader
            :header="$t('container.sockPath')"
            :back="handleClose"
          />
        </template>
        <n-form
          ref="formRef"
          v-loading="loading"
          label-position="top"
          :model="form"
          :rules="rules"
          @submit.prevent
        >
          <n-row
            type="flex"
            justify="center"
          >
            <n-col :span="22">
              <n-form-item
                :label="$t('container.sockPath')"
                prop="dockerSockPath"
              >
                <n-input v-model:value="form.dockerSockPath">
                  <template #prefix>unix://</template>
                  <template #suffix></template>
                </n-input>
                <span class="input-help">{{ $t("container.sockPathHelper1") }}</span>
              </n-form-item>
            </n-col>
          </n-row>
        </n-form>
        <template #footer>
          <span class="dialog-footer">
            <n-button @click="drawerVisible = false">{{ $t("commons.button.cancel") }}</n-button>
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
  </div>
</template>

<script lang="ts" setup>
import { updateSetting } from "@/api/modules/setting"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { t } from "@/i18n"
import { MsgSuccess } from "@/utils/message"
import { useDialog, useMessage } from "naive-ui"
import { reactive, ref } from "vue"

const emit = defineEmits<{ (e: "search"): void }>()

const message = useMessage()
const dialog = useDialog()

interface DialogProps {
	dockerSockPath: string
}
const drawerVisible = ref()
const loading = ref()

const form = reactive({
	dockerSockPath: "",
	currentPath: ""
})
const formRef = ref<any>()
const rules = reactive({
	dockerSockPath: [{ required: true, validator: checkSockPath, trigger: "blur" }]
})

function checkSockPath(rule: any, value: any, callback: any) {
	if (!value.endsWith(".sock")) {
		return callback(new Error(t("container.sockPathErr")))
	}
	callback()
}

function acceptParams(params: DialogProps): void {
	form.dockerSockPath = params.dockerSockPath.replaceAll("unix://", "")
	form.currentPath = params.dockerSockPath.replaceAll("unix://", "")
	drawerVisible.value = true
}

async function loadBuildDir(path: string) {
	form.dockerSockPath = path
}

async function onSubmit(formEl: any | undefined) {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return

		  await dialog.warning({
				title: "确认删除",
				content: `确定要删除 ${form.currentPath}`,
				positiveText: "确定",
				negativeText: "取消",
				onPositiveClick: async () => {
				try {
				loading.value = true
						let params = {
							key: "DockerSockPath",
							value: form.dockerSockPath.startsWith("unix://") ? form.dockerSockPath : `unix://${form.dockerSockPath}`
						}
						await updateSetting(params)
							.then(() => {
								loading.value = false
								handleClose()
								emit("search")
								MsgSuccess(t("commons.msg.operationSuccess"))
							})
							.catch(() => {
								loading.value = false
							})
				} catch (error: any) {
					message.error(error.message || "删除失败")
				}
				}
			})

 
	})
}

function handleClose() {
	drawerVisible.value = false
}

defineExpose({
	acceptParams
})
</script>
