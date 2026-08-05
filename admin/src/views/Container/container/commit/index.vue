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
          :header="$t('container.makeImage')"
          :resource="form.containerName"
          :back="handleClose"
        />
      </template>
      <n-row v-loading="loading">
        <n-col
          :span="22"
          :offset="1"
        >
          <n-form
            @submit.prevent
            ref="formRef"
            :model="form"
            label-position="top"
          >
            <n-form-item
              prop="newImageName"
              :rules="Rules.imageName"
            >
              <template #label>
                {{ $t("container.newImageName") }}
              </template>
              <n-input v-model:value="form.newImageName" />
            </n-form-item>
            <n-form-item prop="comment">
              <template #label>
                {{ $t("container.commitMessage") }}
              </template>
              <n-input v-model:value="form.comment" />
            </n-form-item>
            <n-form-item prop="author">
              <template #label>
                {{ $t("container.author") }}
              </template>
              <n-input v-model:value="form.author" />
            </n-form-item>
            <n-form-item prop="pause">
              <n-checkbox v-model="form.pause">
                {{ $t("container.ifPause") }}
              </n-checkbox>
            </n-form-item>
          </n-form>
        </n-col>
      </n-row>
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

<script setup lang="ts">
import { reactive, ref } from "vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { Rules } from "@/global/form-rules"
import { commitContainer } from "@/api/modules/container"
import { MsgSuccess } from "@/utils/message"
import { t } from "@/i18n"
import { useDialog, useMessage } from "naive-ui"

const drawerVisible = ref<boolean>(false)
const dialog = useDialog()
const message = useMessage()
const emit = defineEmits<{ (e: "search"): void }>()
const loading = ref(false)
const form = reactive({
	containerID: "",
	containerName: "",
	newImageName: "",
	comment: "",
	author: "",
	pause: false
})

interface DialogProps {
	containerID: string
	containerName: string
}
const acceptParams = (props: DialogProps): void => {
	form.containerID = props.containerID
	form.containerName = props.containerName
	drawerVisible.value = true
}

const formRef = ref<any>()

const onSubmit = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		  await dialog.warning({
				title: "确认提交",
				content: `确定要将容器 ${form.containerName} 提交为新镜像吗？`,
				positiveText: "确定",
				negativeText: "取消",
				onPositiveClick: async () => {
				try {
						loading.value = true
						await commitContainer(form)
							.then(() => {
								loading.value = false
								emit("search")
								drawerVisible.value = false
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

const handleClose = async () => {
	drawerVisible.value = false
	emit("search")
}

defineExpose({
	acceptParams
})
</script>
