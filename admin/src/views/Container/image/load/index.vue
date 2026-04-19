<template>
  <n-drawer
    v-model:show="loadVisible"
    :destroy-on-close="true"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    size="30%"
  >
    <n-drawer-content>
      <template #header>
        <DrawerHeader
          :header="$t('container.importImage')"
          :back="handleClose"
        />
      </template>
      <n-form
        @submit.prevent
        v-loading="loading"
        ref="formRef"
        :model="form"
        label-position="top"
      >
        <n-row
          type="flex"
          justify="center"
        >
          <n-col :span="22">
            <n-form-item
              :label="$t('container.path')"
              :rules="Rules.requiredInput"
              prop="path"
            >
              <n-input v-model:value="form.path"></n-input>
            </n-form-item>
          </n-col>
        </n-row>
      </n-form>
      <template #footer>
        <span class="dialog-footer">
          <n-button
            :disabled="loading"
            @click="loadVisible = false"
          >
            {{ $t("commons.button.cancel") }}
          </n-button>
          <n-button
            :disabled="loading"
            type="primary"
            @click="onSubmit(formRef)"
          >
            {{ $t("commons.button.import") }}
          </n-button>
        </span>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue"
import { Rules } from "@/global/form-rules"
import { imageLoad } from "@/api/modules/container"
import { MsgSuccess } from "@/utils/message"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { t } from "@/i18n"

const loading = ref(false)

const loadVisible = ref(false)
const form = reactive({
	path: ""
})

const acceptParams = () => {
	loadVisible.value = true
	form.path = ""
}
const handleClose = () => {
	loadVisible.value = false
}

const emit = defineEmits<{ (e: "search"): void }>()

const formRef = ref<any>()

const onSubmit = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		loading.value = true
		await imageLoad(form)
			.then(() => {
				loading.value = false
				loadVisible.value = false
				emit("search")
				MsgSuccess(t("commons.msg.operationSuccess"))
			})
			.catch(() => {
				loading.value = false
			})
	})
}

const loadLoadDir = async (path: string) => {
	form.path = path
}

defineExpose({
	acceptParams
})
</script>
