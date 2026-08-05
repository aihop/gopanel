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
          :header="$t('container.compose')"
          :back="handleClose"
        />
      </template>
      <div v-loading="loading">
        <n-row
          type="flex"
          justify="center"
        >
          <n-col :span="22">
            <n-form
              ref="formRef"
              label-position="top"
              :model="form"
              :rules="rules"
              @submit.prevent
            >
              <n-form-item :label="$t('container.from')">
                <n-radio-group
                  v-model="form.from"
                  @change="onEdit('form')"
                >
                  <n-radio value="edit">{{ $t("commons.button.edit") }}</n-radio>
                  <n-radio value="path">{{ $t("container.pathSelect") }}</n-radio>
                  <n-radio value="template">{{ $t("container.composeTemplate") }}</n-radio>
                </n-radio-group>
              </n-form-item>
              <n-form-item
                v-if="form.from === 'path'"
                prop="path"
              >
                <n-input
                  v-model:value="form.path"
                  :placeholder="`${$t('commons.example')}/tmp/docker-compose.yml`"
                  @change="onEdit('')"
                ></n-input>
              </n-form-item>
              <n-form-item
                v-if="form.from === 'template'"
                prop="template"
              >
                <n-select
                  v-model="form.template"
                  @change="onEdit('template')"
                  :options="templateOptions.map(item => ({ label: item.name, value: item.id }))"
                />
              </n-form-item>
              <n-form-item
                v-if="form.from === 'edit' || form.from === 'template'"
                prop="name"
              >
                <n-input
                  v-model.trim="form.name"
                  @input="changePath"
                  @change="onEdit('')"
                >
                  <template #prefix>
                    <span style="margin-right: 8px">{{ $t("file.dir") }}</span>
                  </template>
                </n-input>
                <span class="input-help">
                  {{ $t("container.composePathHelper", [composeFile]) }}
                </span>
              </n-form-item>
              <n-form-item>
                <div
                  v-if="form.from === 'edit' || form.from === 'template'"
                  style="width: 100%"
                >
                  <n-radio-group
                    v-model="mode"
                    size="small"
                  >
                    <n-radio label="edit">{{ $t("commons.button.edit") }}</n-radio>
                    <n-radio label="log">{{ $t("commons.button.log") }}</n-radio>
                  </n-radio-group>
                  <FtEditor
                    v-if="mode === 'edit'"
                    v-model="form.file"
                    language="yaml"
                    height="calc(100vh - 400px)"
                    @change="onEdit('')"
                  />
                </div>
                <div style="width: 100%">
                  <LogFile
                    v-if="mode === 'log' && showLog"
                    ref="logRef"
                    v-model:is-reading="isReading"
                    :config="logConfig"
                    :default-button="false"
                    style="height: calc(100vh - 370px); min-height: 200px"
                  />
                </div>
              </n-form-item>
              <n-form-item
                :label="$t('container.env')"
                prop="envStr"
              >
                <n-input
                  v-model:value="form.envStr"
                  type="textarea"
                  :placeholder="$t('container.tagHelper')"
                  :rows="3"
                />
              </n-form-item>
              <span class="input-help whitespace-break-spaces">
                {{ $t("container.editComposeHelper") }}
              </span>
              <FtEditor
                v-model="form.envFileContent"
                language="yaml"
                height="220px"
                :readonly="true"
              />
            </n-form>
          </n-col>
        </n-row>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <n-button @click="drawerVisible = false">
            {{ $t("commons.button.cancel") }}
          </n-button>
          <n-button
            type="primary"
            :disabled="isStartReading || isReading"
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
import { listComposeTemplate, testCompose, upCompose } from "@/api/modules/container"
import { settingSystemBaseDirAPI } from "@/api/modules/setting"
import FtEditor from "@/components/FtEditor/index.vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { Rules } from "@/global/form-rules"
import { t } from "@/i18n"
import { MsgError } from "@/utils/message"
import { useDialog } from "naive-ui"
import { nextTick, onBeforeUnmount, reactive, ref } from "vue"
const emit = defineEmits<{ (e: "search"): void }>()

const showLog = ref(false)
const loading = ref()
const mode = ref("edit")
const oldFrom = ref("edit")
const drawerVisible = ref(false)
const templateOptions = ref()
const baseDir = ref()
const composeFile = ref()
const dialog = useDialog()
let timer: NodeJS.Timer | null = null
const logRef = ref()
const isStartReading = ref(false)
const isReading = ref()

const logConfig = reactive({
	type: "compose-create",
	name: ""
})

const form = reactive({
	name: "",
	from: "edit",
	path: "",
	file: "",
	template: null as number,
	env: [],
	envStr: "",
	envFileContent: `env_file:\n  - gopanel.env`
})
const rules = reactive({
	name: [Rules.requiredInput, Rules.composeName],
	path: [Rules.requiredInput],
	template: [Rules.requiredSelect]
})

async function loadTemplates() {
	const res = await listComposeTemplate()
	templateOptions.value = res.data
}

function acceptParams(): void {
	mode.value = "edit"
	drawerVisible.value = true
	form.name = ""
	form.from = "edit"
	form.path = ""
	form.file = ""
	form.template = null
	form.envStr = ""
	form.env = []
	loadTemplates()
	loadPath()
	isStartReading.value = false
}
function changeTemplate() {
	for (const item of templateOptions.value) {
		if (form.template === item.id) {
			form.file = item.content
			break
		}
	}
}

function changeFrom() {
	if ((oldFrom.value === "edit" || oldFrom.value === "template") && form.file) {
		  dialog.warning({
        title: "确认删除",
        content: `确定要删除 ${form.name} 吗？`,
        positiveText: "确定",
        negativeText: "取消",
        onPositiveClick: async () => {
        try {
            if (oldFrom.value === "template") {
            form.template = null
            form.file = ""
          }
          if (oldFrom.value === "edit") {
            form.file = ""
          }
          oldFrom.value = form.from
        } catch (error: any) {
          form.from = oldFrom.value
        }
			}
		})
	} else {
		oldFrom.value = form.from
	}
}

function handleClose() {
	emit("search")
	clearInterval(Number(timer))
	timer = null
	drawerVisible.value = false
}

async function loadPath() {
	const pathRes = await settingSystemBaseDirAPI()
	baseDir.value = pathRes.data
	changePath()
}

async function changePath() {
	composeFile.value = `${baseDir.value}/docker/compose/${form.name}`
}

const formRef = ref<any>()

function onEdit(item: string) {
	if (item === "template") {
		changeTemplate()
	}
	if (item === "form") {
		changeFrom()
	}
	if (!isReading.value && isStartReading.value) {
		isStartReading.value = false
	}
}
async function onSubmit(formEl: any | undefined) {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		if ((form.from === "edit" || form.from === "template") && form.file.length === 0) {
			MsgError(t("container.contentEmpty"))
			return
		}
		if (form.envStr) {
			form.env = form.envStr.split("\n")
		}
		loading.value = true
		await testCompose(form)
			.then(async res => {
				loading.value = false
				if (res.data) {
					mode.value = "log"
					await upCompose(form)
						.then(res => {
							logConfig.name = res.data
							loadLogs()
							isStartReading.value = true
						})
						.catch(() => {
							loading.value = false
						})
				}
			})
			.catch(() => {
				loading.value = false
			})
	})
}

function loadLogs() {
	showLog.value = false
	nextTick(() => {
		showLog.value = true
		nextTick(() => {
			logRef.value.changeTail(true)
		})
	})
}

async function loadDir(path: string) {
	form.path = path
}

onBeforeUnmount(() => {
	clearInterval(Number(timer))
	timer = null
})

defineExpose({
	acceptParams
})
</script>

<style scoped lang="scss"></style>
