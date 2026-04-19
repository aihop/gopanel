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
          :header="$t('container.imageTag')"
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
            <n-form-item :label="$t('container.from')">
              <n-checkbox v-model="form.fromRepo">{{ $t("container.imageRepo") }}</n-checkbox>
            </n-form-item>
            <n-form-item
              v-if="form.fromRepo"
              :label="$t('container.repoName')"
              :rules="Rules.requiredSelect"
              prop="repo"
            >
              <n-select
                style="width: 100%"
                clearable
                filterable
                v-model="form.repo"
                @change="changeRepo"
              >
                <n-option
                  v-for="item in repos"
                  :key="item.id"
                  :value="item.name"
                  :label="item.name"
                />
              </n-select>
            </n-form-item>
            <n-form-item
              :label="$t('container.imageTag')"
              :rules="Rules.imageName"
              prop="targetName"
            >
              <n-input v-model:value="form.targetName" />
            </n-form-item>

            <n-form-item>
              <n-checkbox
                style="width: 100%"
                v-model="form.deleteTag"
              >
                {{ $t("container.imageTagDeleteHelper") }}
              </n-checkbox>
              <n-checkbox-group
                class="ml-5"
                v-if="form.deleteTag"
                v-model="form.deleteTags"
              >
                <n-checkbox
                  style="width: 100%"
                  v-for="item in tags"
                  :key="item"
                  :value="item"
                  :label="item"
                />
              </n-checkbox-group>
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
            {{ $t("commons.button.save") }}
          </n-button>
        </span>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue"
import { Rules } from "@/global/form-rules"
import { t } from "@/i18n"
import { imageRemove, imageTag } from "@/api/modules/container"
import { Container } from "@/api/interface/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { MsgSuccess } from "@/utils/message"

const loading = ref(false)

const drawerVisible = ref(false)
const repos = ref()
const tags = ref()
const form = reactive({
	imageID: "",
	fromRepo: false,
	repo: "",
	originName: "",
	targetName: "",

	deleteTag: false,
	deleteTags: []
})

interface DialogProps {
	repos: Array<Container.RepoOptions>
	imageID: string
	tags: Array<string>
}

const acceptParams = async (params: DialogProps): Promise<void> => {
	drawerVisible.value = true
	form.imageID = params.imageID
	form.originName = params.tags?.length !== 0 ? params.tags[0] : ""
	form.targetName = params.tags?.length !== 0 ? params.tags[0] : ""
	form.fromRepo = false
	form.repo = ""
	form.deleteTag = false
	form.deleteTags = []
	repos.value = params.repos
	tags.value = params.tags
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
		let params = {
			sourceID: form.imageID,
			targetName: form.targetName
		}
		loading.value = true
		await imageTag(params)
			.then(async () => {
				loading.value = false
				if (form.deleteTag && form.deleteTags.length !== 0) {
					await imageRemove({ names: form.deleteTags })
				}
				drawerVisible.value = false
				emit("search")
				MsgSuccess(t("commons.msg.operationSuccess"))
			})
			.catch(() => {
				loading.value = false
			})
	})
}

const changeRepo = val => {
	if (val === "Docker Hub") {
		form.targetName = form.originName
		return
	}
	for (const item of repos.value) {
		if (item.name == val) {
			form.targetName = item.downloadUrl + "/" + form.originName
			return
		}
	}
}

defineExpose({
	acceptParams
})
</script>
