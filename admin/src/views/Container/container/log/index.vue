<template>
  <n-drawer
    v-model:show="logVisible"
    :width="globalStore.isFullScreen ? '100%' : '50%'"
    :trap-focus="false"
    :block-scroll="false"
    @update:show="handleClose"
  >
    <n-drawer-content
      :title="$t('commons.button.log')"
      :native-scrollbar="false"
      closable
    >
      <template #header>
        <div class="flex justify-between">
          <div class="flex items-center">
            <div
              class="flex cursor-pointer items-center gap-2 text-gray-500"
              @click="handleClose"
            >
              <n-icon>
                <Icon name="mdi:arrow-left" />
              </n-icon>
              返回
            </div>
            <n-divider vertical />
            {{ $t("commons.button.log") }}
          </div>

          <n-tooltip
            v-if="!mobile"
            trigger="hover"
          >
            <template #trigger>
              <n-button
                class="fullScreen"
                quaternary
                circle
                @click="toggleFullscreen"
              >
                <template #icon>
                  <n-icon>
                    <Icon :name="globalStore.isFullScreen ? 'mdi:fullscreen-exit' : 'mdi:fullscreen'" />
                  </n-icon>
                </template>
              </n-button>
            </template>
            {{ $t(`commons.button.${globalStore.isFullScreen ? "quitFullscreen" : "fullscreen"}`) }}
          </n-tooltip>
        </div>
      </template>

      <div class="flex w-full flex-col gap-3 md:flex-row">
        <n-select
          v-model:value="logSearch.mode"
          :options="timeOptions"
          @update:value="searchLogs"
        ></n-select>

        <n-select
          v-model:value="logSearch.tail"
          :options="[
						{ label: $t('commons.table.all'), value: 0 },
						{ label: '100', value: 100 },
						{ label: '200', value: 200 },
						{ label: '500', value: 500 },
						{ label: '1000', value: 1000 }
					]"
          @update:value="searchLogs"
        >
          <template #header>{{ $t("container.lines") }}</template>
        </n-select>
        <n-space class="flex items-center">
          <n-checkbox
            v-model:checked="logSearch.isWatch"
            class="min-w-[100px]"
            @update:checked="searchLogs"
          >
            {{ $t("commons.button.watch") }}
          </n-checkbox>
        </n-space>
        <n-button @click="onDownload">
          <template #icon>
            <n-icon>
              <Icon name="mdi:download" />
            </n-icon>
          </template>
          {{ $t("file.download") }}
        </n-button>

        <n-button @click="onClean">
          <template #icon>
            <n-icon>
              <Icon name="mdi:delete" />
            </n-icon>
          </template>
          {{ $t("commons.button.clean") }}
        </n-button>
      </div>

      <div
        v-if="runtimeSummary"
        class="mt-4 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
      >
        {{ runtimeSummary }}
      </div>

      <div class="mt-4 h-[70vh] overflow-auto">
        <FtEditor
          ref="editorRef"
          v-model="logInfo"
          language="shell"
          height="100%"
          :readonly="true"
        />
      </div>

      <template #footer>
        <n-space justify="end">
          <n-button @click="handleClose">{{ $t("commons.button.cancel") }}</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import { containerCleanLogsAPI, DownloadFile } from "@/api/modules/container"
import FtEditor from "@/components/FtEditor/index.vue"
import { useAuthStore } from "@/store/auth"
import GlobalStore from "@/store/modules/global"
import { dateFormatForName } from "@/utils/util"
import { useDialog, useMessage } from "naive-ui"
import screenfull from "screenfull"
import { computed, nextTick, onBeforeUnmount, reactive, ref, shallowRef, watch } from "vue"
import { useI18n } from "vue-i18n"

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const logVisible = ref(false)
const mobile = computed(() => {
	return globalStore.isMobile()
})

const logInfo = ref<string>("")
const editorRef = ref<InstanceType<typeof FtEditor> | null>(null)
const globalStore = GlobalStore()
const terminalSocket = ref<WebSocket>()
const runtimeSummary = ref("")

const logSearch = reactive({
	isWatch: true,
	container: "",
	containerID: "",
	runtimeHost: "",
	mode: "all",
	tail: 100
})

const timeOptions = ref([
	{ label: t("container.all"), value: "all" },
	{ label: t("container.lastDay"), value: "24h" },
	{ label: t("container.last4Hour"), value: "4h" },
	{ label: t("container.lastHour"), value: "1h" },
	{ label: t("container.last10Min"), value: "10m" }
])

function toggleFullscreen() {
	globalStore.isFullScreen = !globalStore.isFullScreen
}

async function handleClose() {
	if (
		terminalSocket.value?.readyState === terminalSocket.value?.OPEN ||
		terminalSocket.value?.readyState === terminalSocket.value?.CONNECTING
	) {
		// terminalSocket.value?.send("close conn")
		terminalSocket.value?.close()
	}

	logVisible.value = false
	runtimeSummary.value = ""
	globalStore.isFullScreen = false
}

watch(logVisible, val => {
	if (screenfull.isEnabled && !val && !mobile.value) screenfull.exit()
})

async function searchLogs() {
	if (Number(logSearch.tail) < 0) {
		message.error(t("container.linesHelper"))
		return
	}
	terminalSocket.value?.send("close conn")
	terminalSocket.value?.close()
	logInfo.value = ""
	const href = window.location.href
	const protocol = href.split("//")[0] === "http:" ? "ws" : "wss"
	const host = href.split("//")[1].split("/")[0]

	const authStore = useAuthStore()
	const auth = authStore.getAuth() || ""

	const url = `${protocol}://${host}/api/container/logs?container=${logSearch.containerID}&since=${logSearch.mode}&tail=${logSearch.tail}&follow=${logSearch.isWatch}&runtimeHost=${encodeURIComponent(logSearch.runtimeHost || "")}&token=${encodeURIComponent(auth)}`
	terminalSocket.value = new WebSocket(url)

	terminalSocket.value.onmessage = event => {
		logInfo.value += event.data.replace(/\x1B\[[0-?]*[ -/]*[@-~]/g, "")
		nextTick(() => {
			editorRef.value?.scrollToBottom()
		})
	}
}

async function onDownload() {
	logSearch.tail = 0
	dialog.warning({
		title: t("file.download"),
		content: t("container.downLogHelper1", [logSearch.container]),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			const params = {
				container: logSearch.containerID,
				since: logSearch.mode,
				tail: logSearch.tail,
				containerType: "container",
				runtimeHost: logSearch.runtimeHost || ""
			}
			const fileName = `${logSearch.container}-${dateFormatForName(new Date())}.log`
			try {
				const res = await DownloadFile(params)
				const downloadUrl = window.URL.createObjectURL(new Blob([res]))
				const a = document.createElement("a")
				a.style.display = "none"
				a.href = downloadUrl
				a.download = fileName
				const event = new MouseEvent("click")
				a.dispatchEvent(event)
			} catch (error) {
				// 错误提示由请求拦截器统一处理
			}
		}
	})
}

async function onClean() {
	dialog.warning({
		title: t("container.cleanLog"),
		content: t("container.cleanLogHelper"),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: async () => {
			try {
				await containerCleanLogsAPI(logSearch.container)
				searchLogs()
				message.success(t("commons.msg.operationSuccess"))
			} catch (error) {
			}
		}
	})
}

interface DialogProps {
	container: string
	containerID: string
	runtimeHost?: string
	runtimeSummary?: string
}

function acceptParams(props: DialogProps): void {
	logVisible.value = true
	logSearch.containerID = props.containerID
	logSearch.runtimeHost = props.runtimeHost || ""
	runtimeSummary.value = props.runtimeSummary || ""
	logSearch.tail = 100
	logSearch.mode = "all"
	logSearch.isWatch = true
	logSearch.container = props.container
	searchLogs()

	if (!mobile.value) {
		screenfull.on("change", () => {
			globalStore.isFullScreen = screenfull.isFullscreen
		})
	}
}

onBeforeUnmount(() => {
	handleClose()
})

defineExpose({
	acceptParams
})
</script>

<style scoped lang="scss">
.fullScreen {
	border: none;
}
</style>
