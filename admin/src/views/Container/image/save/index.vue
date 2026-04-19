<template>
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
          :header="$t('container.exportImage')"
          :back="handleClose"
        />
      </template>
      <n-form
        v-loading="loading"
        label-position="top"
        ref="formRef"
        :model="form"
        label-width="80px"
      >
        <n-row
          type="flex"
          justify="center"
        >
          <n-col :span="22">
            <n-form-item
              :label="$t('container.tag')"
              :rules="Rules.requiredSelect"
              prop="tagName"
            >
              <n-select
                filterable
                v-model="form.tagName"
              >
                <n-option
                  :disabled="item.indexOf(':<none>') !== -1"
                  v-for="item in form.tags"
                  :key="item"
                  :value="item"
                  :label="item"
                />
              </n-select>
            </n-form-item>
            <n-form-item
              :label="$t('container.path')"
              :rules="Rules.requiredInput"
              prop="path"
            >
              <n-input v-model:value="form.path" />
            </n-form-item>
            <n-form-item
              :label="$t('container.fileName')"
              :rules="Rules.requiredInput"
              prop="name"
            >
              <div class="flex items-center gap-2">
                <n-input
                  v-model:value="form.name"
                  class="flex-1"
                />
                <span class="text-sm text-slate-500">.tar</span>
              </div>
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
            {{ $t("container.export") }}
          </n-button>
        </span>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue"
import { Rules } from "@/global/form-rules"
import { imageSave } from "@/api/modules/container"
import { Container } from "@/api/interface/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { MsgSuccess } from "@/utils/message"
import { t } from "@/i18n"

const loading = ref(false)

const drawerVisible = ref(false)
const form = reactive({
	tags: [] as Array<string>,
	tagName: "",
	path: "",
	name: ""
})

interface DialogProps {
	repos: Array<Container.RepoOptions>
	tags: Array<string>
}
const dialogData = ref<DialogProps>({
	repos: [] as Array<Container.RepoOptions>,
	tags: [] as Array<string>
})

const acceptParams = async (params: DialogProps): Promise<void> => {
	drawerVisible.value = true
	form.tags = params.tags
	form.tagName = form.tags.length !== 0 ? form.tags[0] : ""
	form.path = ""
	form.name = ""
	dialogData.value.repos = params.repos
}
const emit = defineEmits<{ (e: "search"): void }>()

const handleClose = () => {
	drawerVisible.value = false
}

const formRef = ref<any>()

const onSubmit = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		loading.value = true
		await imageSave(form)
			.then(() => {
				loading.value = false
				drawerVisible.value = false
				emit("search")
				MsgSuccess(t("commons.msg.operationSuccess"))
			})
			.catch(() => {
				loading.value = false
			})
	})
}

const loadSaveDir = async (path: string) => {
	form.path = path
}

defineExpose({
	acceptParams
})
</script>
