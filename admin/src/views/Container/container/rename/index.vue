<template>
  <n-modal
    v-model:show="newNameVisible"
    preset="card"
    :title="$t('container.rename')"
    style="width: 300px"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="newNameVisible = false"
  >

    <n-form
      @submit.prevent
      ref="newNameRef"
      v-loading="loading"
      :model="renameForm"
      label-position="top"
    >
      <n-row
        type="flex"
        justify="center"
      >
        <n-col :span="22">
          <n-form-item
            :label="$t('container.newName')"
            :rules="[Rules.containerName, Rules.requiredInput]"
            prop="newName"
          >
            <n-input v-model:value="renameForm.newName"></n-input>
          </n-form-item>
        </n-col>
      </n-row>
    </n-form>
    <template #footer>
      <n-space>
        <n-button
          :disabled="loading"
          @click="newNameVisible = false"
        >
          {{ $t("commons.button.cancel") }}
        </n-button>
        <n-button
          :disabled="loading"
          type="primary"
          @click="onSubmitName(newNameRef)"
        >
          {{ $t("commons.button.confirm") }}
        </n-button>
      </n-space>
    </template>

  </n-modal>
</template>

<script lang="ts" setup>
import { containerRename } from "@/api/modules/container"
import { Rules } from "@/global/form-rules"
import { MsgSuccess } from "@/utils/message"
import { reactive, ref } from "vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { t } from "@/i18n"

const loading = ref(false)

const renameForm = reactive({
	name: "",
	newName: ""
})

const newNameRef = ref<any>()

const newNameVisible = ref<boolean>(false)

const emit = defineEmits<{ (e: "search"): void }>()

const onSubmitName = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		loading.value = true
		await containerRename(renameForm)
			.then(() => {
				loading.value = false
				emit("search")
				newNameVisible.value = false
				MsgSuccess(t("commons.msg.operationSuccess"))
			})
			.catch(() => {
				loading.value = false
			})
	})
}

interface DialogProps {
	container: string
}

const acceptParams = (props: DialogProps): void => {
	renameForm.name = props.container
	renameForm.newName = ""
	newNameVisible.value = true
}

const handleClose = async () => {
	newNameVisible.value = false
	emit("search")
}

defineExpose({
	acceptParams
})
</script>
