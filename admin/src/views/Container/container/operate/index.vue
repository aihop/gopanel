<template>
  <n-drawer
    v-model:show="drawerVisible"
    @update:show="show => !show && handleClose()"
    :mask-closable="false"
    :close-on-esc="false"
    width="50%"
  >
    <n-drawer-content
      :title="title"
      closable
    >
      <template #header>
        <DrawerHeader
          :header="title"
          :hideResource="dialogData.title === 'create'"
          :resource="dialogData.rowData?.name"
          :back="handleClose"
        />
      </template>
      <n-form
        ref="formRef"
        label-placement="top"
        v-loading="loading"
        :model="dialogData.rowData"
        :rules="rules"
        label-width="80"
      >
        <n-grid
          :cols="1"
          justify-content="center"
        >
          <n-gi :span="22">
            <n-form-item
              class="mt-5"
              :label="$t('commons.table.name')"
              path="name"
            >
              <n-input
                :disabled="isFromApp(dialogData.rowData)"
                clearable
                v-model:value="dialogData.rowData.name"
              />
              <div v-if="dialogData.title === 'edit' && isFromApp(dialogData.rowData)">
                <span class="input-help">
                  {{ $t("container.containerFromAppHelper1") }}
                  <n-button
                    style="margin-left: -5px"
                    size="small"
                    text
                    type="primary"
                    @click="goRouter()"
                  >
                    <template #icon>
                      <!-- <n-icon><LocationOutline /></n-icon> -->
                    </template>
                    {{ $t("firewall.quickJump") }}
                  </n-button>
                </span>
              </div>
            </n-form-item>
            <n-form-item
              :label="$t('container.image')"
              path="image"
            >
              <div style="display: flex; flex-direction: column">
                <n-checkbox
                  class="mb-2"
                  v-model:checked="dialogData.rowData.imageInput"
                  :label="$t('container.input')"
                />
                <n-select
                  v-if="!dialogData.rowData.imageInput"
                  filterable
                  v-model:value="dialogData.rowData.image"
                  :options="images"
                  label-field="option"
                  value-field="option"
                />
                <n-input
                  v-else
                  v-model:value="dialogData.rowData.image"
                />
              </div>
            </n-form-item>
            <n-form-item
              path="forcePull"
              :show-label="false"
            >
              <div style="display: flex; flex-direction: column">
                <n-checkbox
                  v-model:checked="dialogData.rowData.forcePull"
                  :label="$t('container.forcePull')"
                  class="mb-2"
                />
                <span
                  style="color: #adb0bc"
                  class="input-help"
                >
                  {{ $t("container.forcePullHelper") }}
                </span>
              </div>
            </n-form-item>
            <n-form-item :label="$t('commons.table.port')">
              <n-radio-group
                v-model:value="dialogData.rowData.publishAllPorts"
                name="publishAllPortsGroup"
                class="ml-4"
              >
                <n-radio
                  :value="false"
                  :label="$t('container.exposePort')"
                />
                <n-radio
                  :value="true"
                  :label="$t('container.exposeAll')"
                />
              </n-radio-group>
            </n-form-item>
            <n-form-item v-if="!dialogData.rowData.publishAllPorts">
              <n-card
                class="widthClass"
                :content-style="{ padding: 25 }"
                :header-style="{ padding: '10px' }"
              >
                <n-data-table
                  v-if="dialogData.rowData.exposedPorts.length !== 0"
                  :columns="exposedPortsColumns"
                  :data="dialogData.rowData.exposedPorts"
                  :bordered="false"
                  :single-line="false"
                />
                <n-button
                  class="mb-2 ml-3 mt-2"
                  @click="handlePortsAdd()"
                >
                  {{ $t("commons.button.add") }}
                </n-button>
              </n-card>
            </n-form-item>
            <n-form-item
              :label="$t('container.network')"
              path="network"
            >
              <n-select
                v-model:value="dialogData.rowData.network"
                :options="networks"
                label-field="option"
                value-field="option"
              />
            </n-form-item>

            <n-form-item
              label="IPv4"
              path="ipv4"
            >
              <n-input
                v-model:value="dialogData.rowData.ipv4"
                :placeholder="$t('container.inputIpv4')"
              />
            </n-form-item>
            <n-form-item
              label="IPv6"
              path="ipv6"
            >
              <n-input
                v-model:value="dialogData.rowData.ipv6"
                :placeholder="$t('container.inputIpv6')"
              />
            </n-form-item>

            <n-form-item :label="$t('container.mount')">
              <div class="mount-list">
                <div
                  v-for="(row, index) in dialogData.rowData.volumes"
                  :key="index"
                  class="mount-list-item"
                >
                  <n-card class="mount-card">
                    <div class="mount-card-header">
                      <n-radio-group
                        v-model:value="row.type"
                        name="volumeTypeGroup"
                      >
                        <n-radio-button
                          value="volume"
                          :label="$t('container.volumeOption')"
                        />
                        <n-radio-button
                          value="bind"
                          :label="$t('container.hostOption')"
                        />
                      </n-radio-group>
                      <n-button
                        text
                        type="primary"
                        @click="handleVolumesDelete(index)"
                      >
                        {{ $t("commons.button.delete") }}
                      </n-button>
                    </div>
                    <n-grid
                      class="mount-card-grid"
                      :x-gap="8"
                      :y-gap="8"
                      :cols="24"
                    >
                      <n-gi :span="row.type === 'volume' ? 10 : 10">
                        <n-form-item
                          v-if="row.type === 'volume'"
                          :label="$t('container.volumeOption')"
                          :path="`volumes[${index}].sourceDir`"
                        >
                          <n-select
                            filterable
                            v-model:value="row.sourceDir"
                            :options="volumes"
                            value-field="option"
                          >
                            <template #arrow>
                              <div></div>
                            </template>
                            <template #empty>
                              <div v-if="!volumes || volumes.length === 0">
                                {{ $t("commons.noData") }}
                              </div>
                            </template>
                          </n-select>
                        </n-form-item>
                        <n-form-item
                          v-else
                          :label="$t('container.hostOption')"
                          :path="`volumes[${index}].sourceDir`"
                        >
                          <n-input v-model:value="row.sourceDir" />
                        </n-form-item>
                      </n-gi>
                      <n-gi :span="5">
                        <n-form-item
                          :label="$t('container.mode')"
                          :path="`volumes[${index}].mode`"
                        >
                          <n-select
                            class="widthClass"
                            filterable
                            v-model:value="row.mode"
                            :options="[
														{ value: 'rw', label: $t('container.modeRW') },
														{ value: 'ro', label: $t('container.modeR') }
													]"
                          />
                        </n-form-item>
                      </n-gi>
                      <n-gi :span="9">
                        <n-form-item
                          :label="$t('container.containerDir')"
                          :path="`volumes[${index}].containerDir`"
                        >
                          <n-input v-model:value="row.containerDir" />
                        </n-form-item>
                      </n-gi>
                    </n-grid>
                  </n-card>
                </div>
                <n-button
                  class="mount-add-btn"
                  @click="handleVolumesAdd()"
                >
                  {{ $t("commons.button.add") }}
                </n-button>
              </div>
            </n-form-item>
            <n-form-item
              label="Command"
              path="cmdStr"
            >
              <n-input
                type="textarea"
                v-model:value="dialogData.rowData.cmdStr"
                :placeholder="$t('container.cmdHelper')"
              />
            </n-form-item>
            <n-form-item
              label="Entrypoint"
              path="entrypointStr"
            >
              <n-input
                v-model:value="dialogData.rowData.entrypointStr"
                :placeholder="$t('container.entrypointHelper')"
              />
            </n-form-item>
            <n-form-item path="autoRemove">
              <n-checkbox v-model:checked="dialogData.rowData.autoRemove">
                {{ $t("container.autoRemove") }}
              </n-checkbox>
            </n-form-item>
            <n-form-item>
              <n-checkbox v-model:checked="dialogData.rowData.privileged">
                {{ $t("container.privileged") }}
              </n-checkbox>
              <span class="input-help">{{ $t("container.privilegedHelper") }}</span>
            </n-form-item>
            <n-form-item :label="$t('container.console')">
              <n-checkbox
                v-model:checked="dialogData.rowData.tty"
                :label="$t('container.tty')"
              />
              <n-checkbox
                v-model:checked="dialogData.rowData.openStdin"
                :label="$t('container.openStdin')"
              ></n-checkbox>
            </n-form-item>
            <n-form-item
              :label="$t('container.restartPolicy')"
              path="restartPolicy"
            >
              <n-radio-group
                v-model:value="dialogData.rowData.restartPolicy"
                name="restartPolicyGroup"
              >
                <n-radio
                  value="no"
                  :label="$t('container.no')"
                />
                <n-radio
                  value="always"
                  :label="$t('container.always')"
                />
                <n-radio
                  value="on-failure"
                  :label="$t('container.onFailure')"
                />
                <n-radio
                  value="unless-stopped"
                  :label="$t('container.unlessStopped')"
                />
              </n-radio-group>
            </n-form-item>
            <n-form-item
              :label="$t('container.cpuShare')"
              path="cpuShares"
            >
              <n-input-number
                class="mini-form-item"
                v-model:value="dialogData.rowData.cpuShares"
                :min="0"
              />
              <span class="input-help">{{ $t("container.cpuShareHelper") }}</span>
            </n-form-item>

            <n-form-item
              :label="$t('container.cpuQuota')"
              path="nanoCPUs"
              :rule="checkFloatNumberRange(0, Number(limits.cpu))"
            >
              <n-input-group class="mini-form-item">
                <n-input-number
                  v-model:value="dialogData.rowData.nanoCPUs"
                  :step="0.1"
                  :min="0"
                  :max="Number(limits.cpu)"
                />
                <n-input-group-label :style="{ width: 'auto', minWidth: '50px', textAlign: 'center' }">
                  {{ $t("commons.units.core") }}
                </n-input-group-label>
              </n-input-group>

              <span class="input-help">
                {{ $t("container.limitHelper", [limits.cpu]) }}{{ $t("commons.units.core") }}
              </span>
            </n-form-item>
            <n-form-item
              :label="$t('container.memoryLimit')"
              path="memory"
              :rule="checkFloatNumberRange(0, Number(limits.memory))"
            >
              <n-input-group class="mini-form-item">
                <n-input-number
                  v-model:value="dialogData.rowData.memory"
                  :step="1"
                  :min="0"
                  :max="Number(limits.memory)"
                />
                <n-input-group-label :style="{ width: 'auto', minWidth: '35px', textAlign: 'center' }">
                  MB
                </n-input-group-label>
              </n-input-group>
              <span class="input-help">{{ $t("container.limitHelper", [limits.memory]) }}MB</span>
            </n-form-item>
            <n-form-item
              :label="$t('container.tag')"
              path="labelsStr"
            >
              <n-input
                type="textarea"
                :placeholder="$t('container.tagHelper')"
                :rows="3"
                v-model:value="dialogData.rowData.labelsStr"
              />
            </n-form-item>
            <n-form-item
              :label="$t('container.env')"
              path="envStr"
            >
              <n-input
                type="textarea"
                :placeholder="$t('container.tagHelper')"
                :rows="3"
                v-model:value="dialogData.rowData.envStr"
              />
            </n-form-item>
          </n-gi>
        </n-grid>
      </n-form>
      <template #footer>
        <span
          class="dialog-footer"
          style="display: flex; justify-content: flex-end"
        >
          <n-button
            :disabled="loading"
            @click="drawerVisible = false"
            style="margin-right: 8px"
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
  <n-modal
    v-model:show="dialogVisible"
    preset="dialog"
    :title="$t('commons.button.edit')"
    style="width: 30%"
  >
    <div
      v-if="dialogData.title === 'edit' && isFromApp(dialogData.rowData)"
      class="leading-6"
    >
      <div>
        <span>{{ $t("container.updateHelper1") }}</span>
      </div>
      <br />
      <div>
        <span>{{ $t("container.updateHelper2") }}</span>
      </div>
      <div>
        <span>{{ $t("container.updateHelper3") }}</span>
      </div>
      <br />
    </div>
    <div>
      <span>{{ $t("container.updateHelper4") }}</span>
    </div>
    <template #action>
      <n-button
        :disabled="loading"
        @click="dialogVisible = false"
        style="margin-right: 8px"
      >
        {{ $t("commons.button.cancel") }}
      </n-button>
      <n-button
        :disabled="loading"
        type="primary"
        @click="submit()"
      >
        {{ $t("commons.button.confirm") }}
      </n-button>
    </template>
  </n-modal>
</template>

<script lang="ts" setup>
import { reactive, ref, h } from "vue"
import {
	NForm,
	NFormItem,
	NGrid,
	NGi,
	NInput,
	NButton,
	NIcon,
	NCheckbox,
	NSelect,
	NRadioGroup,
	NRadio,
	NRadioButton,
	NTooltip,
	NModal,
	NInputGroup,
	NInputGroupLabel,
	NCard,
	NDataTable,
	NInputNumber,
	type FormInst,
	type DrawerProps as NDrawerProps,
	type SelectOption as NSelectOption
} from "naive-ui"
import { Rules, checkFloatNumberRange, checkNumberRange } from "@/global/form-rules"
import { i18n } from "@/i18n"
import DrawerHeader from "@/components/DrawerHeader.vue"
import {
	listImage,
	listVolume,
	createContainer,
	updateContainer,
	loadResourceLimit,
	listNetwork,
	containerListAPI
} from "@/api/modules/container"
import { type Container } from "@/api/interface/container"
import { MsgError, MsgSuccess } from "@/utils/message"
import { checkIpV4V6, checkPort } from "@/utils/util"
import { useRouter } from "vue-router"

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
interface DialogProps {
	title: string
	rowData: Container.ContainerHelper
	getTableList?: () => Promise<any>
}

const title = ref<string>("")
const drawerVisible = ref(false)
const dialogVisible = ref(false)

const dialogData = ref<DialogProps>({
	title: "",
	rowData: {
		containerID: "",
		memoryItem: 0,
		name: "",
		image: "",
		imageInput: false,
		forcePull: false,
		publishAllPorts: false,
		exposedPorts: [],
		network: "",
		ipv4: "",
		ipv6: "",
		volumes: [],
		cmdStr: "",
		entrypointStr: "",
		autoRemove: false,
		privileged: false,
		tty: false,
		openStdin: false,
		restartPolicy: "no",
		cpuShares: 0,
		nanoCPUs: 0,
		memory: 0,
		labelsStr: "",
		envStr: "",
		cmd: [],
		entrypoint: [],
		labels: [],
		env: []
	} as Container.ContainerHelper
})
const acceptParams = (params: DialogProps): void => {
	dialogData.value = {
		...params,
		rowData: params.rowData ? JSON.parse(JSON.stringify(params.rowData)) : dialogData.value.rowData
	}
	title.value = t("container." + dialogData.value.title)
	if (params.title === "edit" && dialogData.value.rowData) {
		const currentMemory = dialogData.value.rowData.memory
		dialogData.value.rowData.memory = Number(typeof currentMemory === "number" ? currentMemory.toFixed(2) : 0)

		let itemCmd = ""
		dialogData.value.rowData.cmd = dialogData.value.rowData?.cmd || []
		for (const item of dialogData.value.rowData.cmd) {
			if (item.indexOf(" ") !== -1) {
				itemCmd += `"${escapeQuotes(item)}" `
			} else {
				itemCmd += item + " "
			}
		}
		dialogData.value.rowData.cmdStr = itemCmd.trimEnd()
		let itemEntrypoint = ""
		dialogData.value.rowData.entrypoint = dialogData.value.rowData?.entrypoint || []
		for (const item of dialogData.value.rowData.entrypoint) {
			if (item.indexOf(" ") !== -1) {
				itemEntrypoint += `"${escapeQuotes(item)}" `
			} else {
				itemEntrypoint += item + " "
			}
		}
		dialogData.value.rowData.entrypointStr = itemEntrypoint.trimEnd()

		dialogData.value.rowData.labels = dialogData.value.rowData.labels || []
		dialogData.value.rowData.env = dialogData.value.rowData.env || []
		dialogData.value.rowData.labelsStr = dialogData.value.rowData.labels.join("\n")
		dialogData.value.rowData.envStr = dialogData.value.rowData.env.join("\n")
		dialogData.value.rowData.exposedPorts = dialogData.value.rowData.exposedPorts || []
		for (const item of dialogData.value.rowData.exposedPorts) {
			if (item.hostIP) {
				item.host = item.hostIP + ":" + item.hostPort
			} else {
				item.host = item.hostPort
			}
		}
		dialogData.value.rowData.volumes = dialogData.value.rowData.volumes || []
	}
	loadLimit()
	loadImageOptions()
	loadVolumeOptions()
	loadNetworkOptions()
	drawerVisible.value = true
}
const emit = defineEmits<{ (e: "search"): void }>()

const images = ref<NSelectOption[]>([])
const volumes = ref<NSelectOption[]>([])
const networks = ref<NSelectOption[]>([])
const limits = ref<Container.ResourceLimit>({
	cpu: 0,
	memory: 0
})

const handleClose = () => {
	emit("search")
	drawerVisible.value = false
	dialogVisible.value = false
}

const rules = reactive({
	name: [Rules.requiredInput, Rules.containerName],
	image: [Rules.imageName],
	cpuShares: [Rules.integerNumberWith0, checkNumberRange(0, 262144)]
})

const formRef = ref<FormInst>()

const exposedPortsColumns = ref([
	{
		title: () => t("container.server"),
		key: "host",
		minWidth: 150,
		render(row: any, index: number) {
			return h(NInput, {
				placeholder: t("container.serverExample"),
				value: row.host,
				"onUpdate:value": v => {
					dialogData.value.rowData.exposedPorts[index].host = v
				}
			})
		}
	},
	{
		title: () => t("container.container"),
		key: "containerPort",
		minWidth: 80,
		render(row: any, index: number) {
			return h(NInput, {
				placeholder: t("container.containerExample"),
				value: row.containerPort,
				"onUpdate:value": v => {
					dialogData.value.rowData.exposedPorts[index].containerPort = v
				}
			})
		}
	},
	{
		title: () => t("commons.table.protocol"),
		key: "protocol",
		minWidth: 50,
		render(row: any, index: number) {
			return h(NSelect, {
				value: row.protocol,
				options: [
					{ label: "tcp", value: "tcp" },
					{ label: "udp", value: "udp" }
				],
				style: { width: "100%" },
				placeholder: t("container.serverExample"),
				"onUpdate:value": v => {
					dialogData.value.rowData.exposedPorts[index].protocol = v
				}
			})
		}
	},
	{
		title: "",
		key: "actions",
		minWidth: 35,
		render(row: any, index: number) {
			return h(
				NButton,
				{
					text: true,
					type: "primary",
					onClick: () => handlePortsDelete(index)
				},
				{ default: () => t("commons.button.delete") }
			)
		}
	}
])

const handlePortsAdd = () => {
	let item = {
		host: "",
		hostIP: "",
		containerPort: "",
		hostPort: "",
		protocol: "tcp"
	}
	dialogData.value.rowData.exposedPorts.push(item)
}
const handlePortsDelete = (index: number) => {
	dialogData.value.rowData.exposedPorts.splice(index, 1)
}

const goRouter = async () => {
	router.push({ name: "AppInstalled" })
}

const handleVolumesAdd = () => {
	let item = {
		type: "bind",
		sourceDir: "",
		containerDir: "",
		mode: "rw"
	}
	dialogData.value.rowData.volumes.push(item)
}
const handleVolumesDelete = (index: number) => {
	dialogData.value.rowData.volumes.splice(index, 1)
}

const loadLimit = async () => {
	try {
		const res = await loadResourceLimit()
		if (res && res.data) {
			limits.value = res.data
			if (typeof limits.value.memory === "number") {
				limits.value.memory = Number((limits.value.memory / 1024 / 1024).toFixed(2))
			} else {
				limits.value.memory = 0
			}
			if (typeof limits.value.cpu !== "number") {
				limits.value.cpu = 0
			}
		} else {
			limits.value.cpu = 0
			limits.value.memory = 0
		}
	} catch (error) {
		console.error("Failed to load resource limits:", error)
		limits.value.cpu = 0
		limits.value.memory = 0
	}
}

const loadImageOptions = async () => {
	const res = await listImage()
	images.value = res.data.map(item => ({ ...item, key: item.option }))
}
const loadVolumeOptions = async () => {
	const res = await listVolume()
	volumes.value = res.data.map(item => ({ ...item, key: item.option }))
}
const loadNetworkOptions = async () => {
	const res = await listNetwork()
	networks.value = res.data.map(item => ({ ...item, key: item.option }))
}

const onSubmit = async (formEl: FormInst | undefined) => {
	if (dialogData.value.rowData.volumes.length !== 0) {
		for (const item of dialogData.value.rowData.volumes) {
			if (!item.containerDir || !item.sourceDir) {
				MsgError(t("container.volumeHelper"))
				return
			}
		}
	}
	if (!formEl) return
	formEl.validate(async errors => {
		if (!errors) {
			if (dialogData.value.title === "create") {
				submit()
			} else {
				dialogVisible.value = true
			}
		}
	})
}

const submit = async () => {
	dialogVisible.value = false
	if (dialogData.value.rowData?.envStr) {
		dialogData.value.rowData.env = dialogData.value.rowData.envStr.split("\n")
	}
	if (dialogData.value.rowData?.labelsStr) {
		dialogData.value.rowData.labels = dialogData.value.rowData.labelsStr.split("\n")
	}

	if (dialogData.value.rowData?.cmdStr) {
		dialogData.value.rowData.cmd = splitStringIgnoringQuotes(dialogData.value.rowData.cmdStr)
	} else {
		dialogData.value.rowData.cmd = []
	}

	if (dialogData.value.rowData?.entrypointStr) {
		dialogData.value.rowData.entrypoint = splitStringIgnoringQuotes(dialogData.value.rowData.entrypointStr)
	} else {
		dialogData.value.rowData.entrypoint = []
	}

	if (dialogData.value.rowData.publishAllPorts) {
		dialogData.value.rowData.exposedPorts = []
	} else {
		if (!checkPortValid()) {
			return
		}
	}
	dialogData.value.rowData.memory = Number(dialogData.value.rowData.memory)
	dialogData.value.rowData.nanoCPUs = Number(dialogData.value.rowData.nanoCPUs)

	loading.value = true
	try {
		if (dialogData.value.title === "create") {
			await createContainer(dialogData.value.rowData)
		} else {
			await updateContainer(dialogData.value.rowData)
		}
		MsgSuccess(t("commons.msg.operationSuccess"))
		emit("search")
		drawerVisible.value = false
		dialogVisible.value = false
	} catch (error) {
		if (dialogData.value.title !== "create") {
			updateContainerID()
		}
	} finally {
		loading.value = false
	}
}

const updateContainerID = async () => {
	let params = {
		page: 1,
		limit: 1,
		state: "all",
		name: dialogData.value.rowData.name,
		filters: "",
		orderBy: "created_at",
		order: "null"
	}
	await containerListAPI(params).then(res => {
		if (res.data.items?.length === 1) {
			dialogData.value.rowData.containerID = res.data.items[0].containerID
			return
		}
	})
}

const checkPortValid = () => {
	if (dialogData.value.rowData.exposedPorts.length === 0) {
		return true
	}
	for (const port of dialogData.value.rowData.exposedPorts) {
		if (port.host.indexOf(":") !== -1) {
			port.hostIP = port.host.substring(0, port.host.lastIndexOf(":"))
			if (checkIpV4V6(port.hostIP)) {
				MsgError(t("firewall.addressFormatError"))
				return false
			}
			port.hostPort = port.host.substring(port.host.lastIndexOf(":") + 1)
		} else {
			port.hostIP = ""
			port.hostPort = port.host
		}
		if (port.hostPort.indexOf("-") !== -1) {
			if (checkPort(port.hostPort.split("-")[0])) {
				MsgError(t("firewall.portFormatError"))
				return false
			}
			if (checkPort(port.hostPort.split("-")[1])) {
				MsgError(t("firewall.portFormatError"))
				return false
			}
		} else {
			if (checkPort(port.hostPort)) {
				MsgError(t("firewall.portFormatError"))
				return false
			}
		}
		if (port.containerPort.indexOf("-") !== -1) {
			if (checkPort(port.containerPort.split("-")[0])) {
				MsgError(t("firewall.portFormatError"))
				return false
			}
			if (checkPort(port.containerPort.split("-")[1])) {
				MsgError(t("firewall.portFormatError"))
				return false
			}
		} else {
			if (checkPort(port.containerPort)) {
				MsgError(t("firewall.portFormatError"))
				return false
			}
		}
	}
	return true
}

const isFromApp = (rowData: Container.ContainerHelper) => {
	if (rowData && rowData.labels) {
		return rowData.labels.indexOf("createdBy=Apps") > -1
	}
	return false
}

const escapeQuotes = (input: string): string => {
	if (!input) return ""
	const placeholder = "___TEMP_ESCAPED_QUOTE___"
	let result = input.replace(/\\"/g, placeholder)
	result = result.replace(/"/g, '\\"')
	result = result.replace(new RegExp(placeholder, "g"), '\\"')
	return result
}

const splitStringIgnoringQuotes = (input: string): string[] => {
	if (!input) return []
	input = input.replace(/\\"/g, "<quota>")
	const regex = /"([^"]*)"|(\S+)/g
	const result: string[] = []
	let match

	while ((match = regex.exec(input)) !== null) {
		if (match[1]) {
			result.push(match[1].replace(/<quota>/g, '\\"'))
		} else if (match[2]) {
			result.push(match[2].replace(/<quota>/g, '\\"'))
		}
	}
	return result
}
defineExpose({
	acceptParams
})
</script>

<style lang="scss" scoped>
.widthClass {
	width: 100%;
}
.mini-form-item {
}

.n-card > .n-card__content {
	padding: 0;
}
.n-card > .n-card__header {
}

.mount-list {
	width: 100%;
}

.mount-list-item + .mount-list-item {
	margin-top: 8px;
}

.mount-card {
	width: 100%;
}

.mount-card-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
	flex-wrap: wrap;
}

.mount-card-grid {
	margin-top: 16px;
}

.mount-add-btn {
	margin-top: 12px;
}

.dialog-footer {
	display: flex;
	justify-content: flex-end;
	.n-button {
		margin-left: 8px;
	}
}
.input-help {
	color: #adb0bc;
}
</style>
