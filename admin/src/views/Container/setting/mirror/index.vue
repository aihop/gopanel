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
            :header="$t('container.mirrors')"
            :back="handleClose"
          />
        </template>
        <n-form
          ref="formRef"
          label-position="top"
          :model="form"
          @submit.prevent
          :rules="rules"
          v-loading="loading"
        >
          <n-row
            type="flex"
            justify="center"
          >
            <n-col :span="22">
              <n-form-item
                :label="$t('container.mirrors')"
                prop="mirrors"
              >
                <n-input
                  type="textarea"
                  :placeholder="$t('container.mirrorHelper')"
                  :rows="5"
                  v-model:value="form.mirrors"
                />
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
              @click="onSave(formRef)"
            >
              {{ $t("commons.button.confirm") }}
            </n-button>
          </span>
        </template>
      </n-drawer-content>
    </n-drawer>
    <ConfirmDialog
      ref="confirmDialogRef"
      @confirm="onSubmit"
    />
  </div>
</template>
<script lang="ts" setup>
import { reactive, ref } from "vue"
import { MsgSuccess } from "@/utils/message"
import ConfirmDialog from "@/components/confirm-dialog/index.vue"
import { updateDaemonUpdate } from "@/api/modules/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { emptyLineFilter } from "@/utils/util"
import { t } from "@/i18n"

const emit = defineEmits<{ (e: "search"): void }>()

const confirmDialogRef = ref()

interface DialogProps {
	mirrors: string
}
const drawerVisible = ref()
const loading = ref()

const form = reactive({
	mirrors: ""
})
const formRef = ref<any>()
const rules = reactive({
	mirrors: [{ validator: checkMirrors, trigger: "blur" }]
})

function checkMirrors(rule: any, value: any, callback: any) {
	if (form.mirrors !== "") {
		const reg = /^https?:\/\/[a-zA-Z0-9.-]+(:[0-9]{1,5})?(\/[a-zA-Z0-9./-]*)?$/
		let mirrors = form.mirrors.split("\n")
		for (const item of mirrors) {
			if (item === "") {
				continue
			}
			if (!reg.test(item)) {
				return callback(new Error(t("commons.rule.mirror")))
			}
		}
	}
	callback()
}

const acceptParams = (params: DialogProps): void => {
	form.mirrors = params.mirrors || params.mirrors.replaceAll(",", "\n")
	drawerVisible.value = true
}

const onSave = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async (valid: boolean) => {
		if (!valid) return
		let params = {
			header: t("database.confChange"),
			operationInfo: t("database.restartNowHelper"),
			submitInputInfo: t("database.restartNow")
		}
		confirmDialogRef.value!.acceptParams(params)
	})
}

const onSubmit = async () => {
	loading.value = true
	await updateDaemonUpdate("Mirrors", emptyLineFilter(form.mirrors, "\n").replaceAll("\n", ","))
		.then(() => {
			loading.value = false
			emit("search")
			handleClose()
			MsgSuccess(t("commons.msg.operationSuccess"))
		})
		.catch(() => {
			loading.value = false
		})
}

const handleClose = () => {
	drawerVisible.value = false
}

defineExpose({
	acceptParams
})
</script>
