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
      <div
        v-if="dialogData.title === 'edit' && runtimeSummary"
        class="mb-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm text-slate-600"
      >
        {{ runtimeSummary }}
      </div>
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
            <ContainerOperateBasicSection
              :title="dialogData.title"
              :row-data="dialogData.rowData"
              :is-from-app-value="isFromAppValue"
              :images="images"
              @go-router="goRouter"
            />

            <ContainerOperatePortSection
              :row-data="dialogData.rowData"
              :networks="networks"
              :exposed-ports-columns="exposedPortsColumns"
              @add-port="handlePortsAdd"
            />

            <ContainerOperateVolumeSection
              :row-data="dialogData.rowData"
              :volumes="volumes"
              @add-volume="handleVolumesAdd"
              @delete-volume="handleVolumesDelete"
            />

            <ContainerOperateAdvancedSection
              :row-data="dialogData.rowData"
              :limits="limits"
            />
          </n-gi>
        </n-grid>
      </n-form>
      <template #footer>
        <span class="flex justify-end gap-2">
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
  <n-modal
    v-model:show="dialogVisible"
    preset="dialog"
    :title="$t('commons.button.edit')"
    class="w-[30%]"
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
        class="mr-2"
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
import { computed, reactive, ref, watch } from "vue"
import {
	NForm,
	NGrid,
	NGi,
	NModal,
	type FormInst,
	type SelectOption as NSelectOption
} from "naive-ui"
import { Rules, checkNumberRange } from "@/global/form-rules"
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
import { useRouter } from "vue-router"
import { buildRuntimeDetailText } from "@/utils/runtime"
import ContainerOperateAdvancedSection from "./ContainerOperateAdvancedSection.vue"
import ContainerOperateBasicSection from "./ContainerOperateBasicSection.vue"
import { createExposedPortColumns } from "./containerOperateColumns"
import {
	buildSubmitPayload,
	cloneContainerHelper,
	createDefaultContainerHelper,
	createEmptyExposedPort,
	createEmptyVolume,
	hydrateContainerFormForEdit,
	isFromApp,
	validateExposedPorts
} from "./containerOperateForm"
import ContainerOperatePortSection from "./ContainerOperatePortSection.vue"
import ContainerOperateVolumeSection from "./ContainerOperateVolumeSection.vue"

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

const runtimeSummary = computed(() => {
	if (dialogData.value.title !== "edit") return ""
	const row = dialogData.value.rowData
	const detail = buildRuntimeDetailText(row, {
		kindFallback: t("container.runtimeType"),
		userFallback: t("container.userDefault"),
		runtimePrefix: "",
		runUserPrefix: `${t("container.runUser")}: `
	})
	const host = String(row.runtimeHost || "").trim()
	return host ? `${detail} · Host: ${host}` : detail
})

const dialogData = ref<DialogProps>({
	title: "",
	rowData: createDefaultContainerHelper()
})
const acceptParams = (params: DialogProps): void => {
	dialogData.value = {
		...params,
		rowData: cloneContainerHelper(params.rowData)
	}
	title.value = t("container." + dialogData.value.title)
	if (params.title === "edit" && dialogData.value.rowData) {
		hydrateContainerFormForEdit(dialogData.value.rowData)
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

const handlePortsAdd = () => {
	dialogData.value.rowData.exposedPorts.push(createEmptyExposedPort())
}
const handlePortsDelete = (index: number) => {
	dialogData.value.rowData.exposedPorts.splice(index, 1)
}

const exposedPortsColumns = ref(createExposedPortColumns(t, () => dialogData.value.rowData.exposedPorts, handlePortsDelete))

const goRouter = async () => {
	router.push({ name: "AppInstalled" })
}

const handleVolumesAdd = () => {
	dialogData.value.rowData.volumes.push(createEmptyVolume())
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

watch(
	() => dialogData.value.rowData.network,
	(newValue, oldValue) => {
		if (!oldValue || newValue === oldValue) {
			return
		}
		dialogData.value.rowData.ipv4 = ""
		dialogData.value.rowData.ipv6 = ""
	}
)

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
	if (!dialogData.value.rowData.publishAllPorts && !validateExposedPorts(dialogData.value.rowData.exposedPorts, MsgError, t)) {
		return
	}
	const payload = buildSubmitPayload(dialogData.value.rowData)

	loading.value = true
	try {
		if (dialogData.value.title === "create") {
			await createContainer(payload)
		} else {
			await updateContainer(payload)
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

const isFromAppValue = computed(() => isFromApp(dialogData.value.rowData))
defineExpose({
	acceptParams
})
</script>
