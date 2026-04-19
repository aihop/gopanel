<template>
  <n-drawer
    v-model:show="drawerVisible"
    :destroy-on-close="true"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    size="50%"
    @close="handleClose"
  >
    <n-drawer-content>
      <template #header>
        <DrawerHeader
          :header="$t('container.imageBuild')"
          :back="handleClose"
        />
      </template>
      <n-row
        type="flex"
        justify="center"
      >
        <n-col :span="22">
          <n-form
            ref="formRef"
            label-position="top"
            :model="form"
            label-width="80px"
            :rules="rules"
          >
            <n-form-item
              :label="$t('commons.table.name')"
              prop="name"
            >
              <n-input
                v-model.trim="form.name"
                :placeholder="$t('container.imageNameHelper')"
                clearable
              />
            </n-form-item>
            <n-form-item
              label="Dockerfile"
              prop="from"
            >
              <n-radio-group
                v-model="form.from"
                @change="onEdit()"
              >
                <n-radio value="edit">{{ $t("commons.button.edit") }}</n-radio>
                <n-radio value="path">{{ $t("container.pathSelect") }}</n-radio>
              </n-radio-group>
            </n-form-item>
            <n-form-item
              v-if="form.from === 'edit'"
              :rules="Rules.requiredInput"
            >
              <FtEditor
                v-model="form.dockerfile"
                language="dockerfile"
                height="calc(100vh - 520px)"
                :readonly="true"
                @change="onEdit()"
              />
            </n-form-item>
            <n-form-item
              v-else
              :rules="Rules.requiredSelect"
              prop="dockerfile"
            >
              <n-input
                v-model:value="form.dockerfile"
                clearable
                @change="onEdit()"
              />
            </n-form-item>
            <n-form-item :label="$t('container.tag')">
              <n-input
                v-model:value="form.tagStr"
                :placeholder="$t('container.tagHelper')"
                type="textarea"
                :rows="3"
                @change="onEdit()"
              />
            </n-form-item>
          </n-form>

          <LogFile
            v-if="logVisible"
            ref="logRef"
            v-model:is-reading="isReading"
            :config="logConfig"
            :default-button="false"
            style="height: calc(100vh - 370px); min-height: 200px"
          />
        </n-col>
      </n-row>

      <template #footer>
        <span class="dialog-footer">
          <n-button @click="drawerVisible = false">{{ $t("commons.button.cancel") }}</n-button>
          <n-button
            :disabled="isStartReading || isReading"
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
import { imageBuild } from "@/api/modules/container"
import FtEditor from "@/components/FtEditor/index.vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { Rules } from "@/global/form-rules"
import { MsgSuccess } from "@/utils/message"
import { nextTick, reactive, ref } from "vue"
const emit = defineEmits<{ (e: "search"): void }>()
import { t } from "@/i18n"
const logVisible = ref<boolean>(false)
const drawerVisible = ref(false)
const logRef = ref()
const isStartReading = ref(false)
const isReading = ref(false)

const logConfig = reactive({
	type: "image-build",
	name: ""
})
const form = reactive({
	from: "path",
	dockerfile: "",
	name: "",
	tagStr: "",
	tags: [] as Array<string>
})

const rules = reactive({
	name: [Rules.requiredInput, Rules.imageName],
	from: [Rules.requiredSelect],
	dockerfile: [Rules.requiredInput]
})
async function acceptParams() {
	logVisible.value = false
	drawerVisible.value = true
	form.from = "path"
	form.dockerfile = ""
	form.tagStr = ""
	form.name = ""
	isStartReading.value = false
}
function handleClose() {
	drawerVisible.value = false
	emit("search")
}

const formRef = ref<any>()

function onEdit() {
	if (!isReading.value && isStartReading.value) {
		isStartReading.value = false
	}
}
async function onSubmit(formEl: any | undefined) {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		if (form.tagStr !== "") {
			form.tags = form.tagStr.split("\n")
		}
		const res = await imageBuild(form)
		isStartReading.value = true
		logConfig.name = res.data
		loadLogs()
		MsgSuccess(t("commons.msg.operationSuccess"))
	})
}

function loadLogs() {
	logVisible.value = false
	nextTick(() => {
		logVisible.value = true
		nextTick(() => {
			logRef.value.changeTail(true)
		})
	})
}

async function loadBuildDir(path: string) {
	form.dockerfile = path
}

defineExpose({
	acceptParams
})
</script>
