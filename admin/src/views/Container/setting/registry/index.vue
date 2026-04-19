<template>
  <div>
    <n-drawer
      v-model:show="drawerVisible"
      :destroy-on-close="true"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      size="30%"
    >
      <template #header>
        <DrawerHeader
          :header="$t('container.registries')"
          :back="handleClose"
        />
      </template>
      <n-form
        ref="formRef"
        label-position="top"
        :model="form"
        :rules="rules"
        @submit.prevent
        v-loading="loading"
      >
        <n-row
          type="flex"
          justify="center"
        >
          <n-col :span="22">
            <n-form-item
              :label="$t('container.registries')"
              prop="registries"
            >
              <n-input
                type="textarea"
                :placeholder="$t('container.registrieHelper')"
                :rows="5"
                v-model:value="form.registries"
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
            @click="onSave"
          >
            {{ $t("commons.button.confirm") }}
          </n-button>
        </span>
      </template>
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
	registries: string
}
const drawerVisible = ref()
const loading = ref()

const form = reactive({
	registries: ""
})
const formRef = ref<any>()
const rules = reactive({
	registries: [{ validator: checkRegistries, trigger: "blur" }]
})

function checkRegistries(rule: any, value: any, callback: any) {
	if (form.registries !== "") {
		const reg = /^[a-zA-Z0-9]{1}[a-z:A-Z0-9_/.-]{0,150}$/
		let regis = form.registries.split("\n")
		for (const item of regis) {
			if (item === "") {
				continue
			}
			if (!reg.test(item)) {
				return callback(new Error(t("commons.rule.imageName")))
			}
		}
	}
	callback()
}

const acceptParams = (params: DialogProps): void => {
	form.registries = params.registries || params.registries.replaceAll(",", "\n")
	drawerVisible.value = true
}

const onSave = async () => {
	let params = {
		header: t("database.confChange"),
		operationInfo: t("database.restartNowHelper"),
		submitInputInfo: t("database.restartNow")
	}
	confirmDialogRef.value!.acceptParams(params)
}

const onSubmit = async () => {
	loading.value = true
	await updateDaemonUpdate("Registries", emptyLineFilter(form.registries, "\n").replaceAll("\n", ","))
		.then(() => {
			loading.value = false
			handleClose()
			emit("search")
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
