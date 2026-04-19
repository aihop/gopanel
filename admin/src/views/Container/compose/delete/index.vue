<template>
  <n-modal
    v-model:show="dialogVisible"
    :title="$t('commons.button.delete') + ' - ' + composeName"
    width="30%"
    :close-on-click-modal="false"
  >
    <n-form
      ref="deleteForm"
      v-loading="loading"
    >
      <n-form-item>
        <n-checkbox
          v-model="deleteFile"
          :label="$t('container.deleteFile')"
        />
        <span class="input-help whitespace-break-spaces">
          {{ $t("container.deleteComposeHelper") }}
        </span>
      </n-form-item>
      <n-form-item>
        <div class="font">
          <span>{{ $t("database.delete") }}</span>
          <span class="warning">{{ composeName }}</span>
          <span>{{ $t("container.deleteCompose") }}</span>
        </div>
        <n-input
          :value="deleteInfo"
          :placeholder="composeName"
        ></n-input>
      </n-form-item>
    </n-form>
    <template #footer>
      <span class="dialog-footer">
        <n-button
          @click="dialogVisible = false"
          :disabled="loading"
        >
          {{ $t("commons.button.cancel") }}
        </n-button>
        <n-button
          type="primary"
          @click="submit"
          :disabled="deleteInfo != composeName || loading"
        >
          {{ $t("commons.button.confirm") }}
        </n-button>
      </span>
    </template>
  </n-modal>
</template>
<script lang="ts" setup>
import { ref } from "vue"
import { t } from "@/i18n"
import { MsgSuccess } from "@/utils/message"
import { composeOperator } from "@/api/modules/container"

let dialogVisible = ref(false)
let loading = ref(false)
let deleteInfo = ref("")

const deleteFile = ref()
const composeName = ref()
const composePath = ref()

interface DialogProps {
	name: string
	path: string
}
const emit = defineEmits<{ (e: "search"): void }>()

const acceptParams = async (prop: DialogProps) => {
	deleteFile.value = false
	composeName.value = prop.name
	composePath.value = prop.path
	deleteInfo.value = ""
	dialogVisible.value = true
}

const submit = async () => {
	loading.value = true
	let params = {
		name: composeName.value,
		path: composePath.value,
		operation: "delete",
		withFile: deleteFile.value
	}
	await composeOperator(params)
		.then(() => {
			loading.value = false
			emit("search")
			MsgSuccess(t("commons.msg.deleteSuccess"))
			dialogVisible.value = false
		})
		.catch(() => {
			loading.value = false
		})
}

defineExpose({
	acceptParams
})
</script>

<style lang="scss" scoped>
.font {
	font-size: 12px;
	.warning {
		color: red;
		font-weight: 500;
	}
}
</style>
