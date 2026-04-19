<template>
  <div>
    <div
      v-if="defaultButton"
      class="mb-4 flex items-center gap-2"
    >
      <n-checkbox
        v-model:checked="tailLog"
        @update:checked="changeTail(false)"
      >
        {{ $t("commons.button.watch") }}
      </n-checkbox>
      <n-button
        :disabled="logs.length === 0"
        @click="onDownload"
      >
        <template #icon>
          <n-icon>
            <download />
          </n-icon>
        </template>
        {{ $t("file.download") }}
      </n-button>
      <slot name="button"></slot>
    </div>
    <div
      ref="logContainer"
      class="log-container"
      @scroll="onScroll"
    >
      <div
        class="log-spacer"
        :style="{ height: `${totalHeight}px` }"
      ></div>
      <div
        v-for="(log, index) in visibleLogs"
        :key="startIndex + index"
        class="log-item"
        :style="{ top: `${(startIndex + index) * logHeight}px` }"
      >
        <span>{{ log }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ReadByLine } from "@/api/modules/file"
import { downloadFile } from "@/utils/util"
import { NButton, NCheckbox, NIcon } from "naive-ui"
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from "vue"

interface LogProps {
	id?: number
	type: string
	name?: string
	tail?: boolean
}

interface Props {
	config: LogProps | null
	style?: string
	defaultButton?: boolean
	loading?: boolean
	hasContent?: boolean
}

const props = withDefaults(defineProps<Props>(), {
	config: () => ({
		id: 0,
		type: "",
		name: "",
		tail: false
	}),
	style: "height: calc(100vh - 200px); width: 100%; min-height: 400px; overflow: auto;",
	defaultButton: true,
	loading: true,
	hasContent: false
})

const emit = defineEmits(["update:loading", "update:hasContent", "update:isReading"])

const stopSignals = [
	"docker-compose up failed!",
	"docker-compose up successful!",
	"image build failed!",
	"image build successful!",
	"image pull failed!",
	"image pull successful!",
	"image push failed!",
	"image push successful!",
	"ollama pull completed!"
]

const tailLog = ref(false)
const loading = ref(props.loading)
const readReq = reactive({
	id: 0,
	type: "",
	name: "",
	page: 1,
	limit: 500,
	latest: false
})

const isLoading = ref(false)
const end = ref(false)
const lastLogs = ref<string[]>([])
const maxPage = ref(0)
const minPage = ref(0)
let timer: NodeJS.Timer | null = null
const logPath = ref("")

const logs = ref<string[]>([])
const logContainer = ref<HTMLElement | null>(null)
const logHeight = 20
const logCount = ref(0)
const totalHeight = computed(() => logHeight * logCount.value)
const containerHeight = ref(500)
const visibleCount = computed(() => Math.ceil(containerHeight.value / logHeight))
const startIndex = ref(0)

const visibleLogs = computed(() => {
	return logs.value.slice(startIndex.value, startIndex.value + visibleCount.value)
})

function onScroll() {
	if (!logContainer.value) return

	const scrollTop = logContainer.value.scrollTop
	if (scrollTop === 0) {
		readReq.page = minPage.value - 1
		if (readReq.page < 1) return
		minPage.value = readReq.page
		getContent(true)
	}
	startIndex.value = Math.floor(scrollTop / logHeight)
}

function changeLoading() {
	loading.value = !loading.value
	emit("update:loading", loading.value)
}

async function onDownload() {
	changeLoading()
	downloadFile(logPath.value)
	changeLoading()
}

function changeTail(fromOutSide: boolean) {
	if (fromOutSide) {
		tailLog.value = !tailLog.value
	}
	if (tailLog.value) {
		timer = setInterval(() => {
			getContent(false)
		}, 1000 * 4)
	} else {
		onCloseLog()
	}
}

function clearLog(): void {
	logs.value = []
	readReq.page = 1
	lastLogs.value = []
}

async function getContent(pre: boolean) {
	if (isLoading.value || !props.config) return

	readReq.id = props.config.id ?? 0
	readReq.type = props.config.type
	readReq.name = props.config.name ?? ""

	if (readReq.page < 1) {
		readReq.page = 1
	}

	isLoading.value = true
	emit("update:isReading", true)

	try {
		const res = await ReadByLine(readReq)
		logPath.value = res.data.path

		if (!end.value && res.data.end) {
			lastLogs.value = [...logs.value]
		}

		if (res.data.lines && res.data.lines.length > 0) {
			const newLogs = res.data.lines.map((line: string) =>
				line
					.replace(/\\u(\w{4})/g, (_match: string, grp: string) => {
						return String.fromCharCode(Number.parseInt(grp, 16))
					})
					.replace(/\x1B\[[0-?]*[mKhlGA]/g, "")
			)

			if (newLogs.length === readReq.limit && readReq.page < res.data.total) {
				readReq.page++
			}

			if (
				stopSignals.some(signal => newLogs[newLogs.length - 1].endsWith(signal)) ||
				/successful|failed/i.test(newLogs[newLogs.length - 1])
			) {
				onCloseLog()
			}

			if (end.value) {
				logs.value =
					logs.value.length === 0
						? newLogs
						: pre
							? [...newLogs, ...lastLogs.value]
							: [...lastLogs.value, ...newLogs]
			} else {
				logs.value =
					logs.value.length === 0 ? newLogs : pre ? [...newLogs, ...logs.value] : [...logs.value, ...newLogs]
			}

			nextTick(() => {
				if (!logContainer.value) return

				if (pre) {
					logContainer.value.scrollTop = 2000
				} else {
					logContainer.value.scrollTop = totalHeight.value
					containerHeight.value = logContainer.value.getBoundingClientRect().height
				}
			})
		}

		logCount.value = logs.value.length
		end.value = res.data.end
		emit("update:hasContent", logs.value.length > 0)

		if (readReq.latest) {
			readReq.page = res.data.total
			readReq.latest = false
			maxPage.value = res.data.total
			minPage.value = res.data.total
		}

		if (logs.value && logs.value.length > 3000) {
			if (pre) {
				logs.value.splice(logs.value.length - readReq.limit, readReq.limit)
				if (maxPage.value > 1) {
					maxPage.value--
				}
			} else {
				logs.value.splice(0, readReq.limit)
				if (minPage.value > 1) {
					minPage.value++
				}
			}
		}
	} catch (error) {
		console.error("Failed to fetch logs:", error)
	} finally {
		isLoading.value = false
	}
}

async function onCloseLog() {
	tailLog.value = false
	if (timer) {
		clearInterval(Number(timer))
		timer = null
	}
	isLoading.value = false
	emit("update:isReading", false)
}

watch(
	() => props.loading,
	newLoading => {
		loading.value = newLoading
	}
)

async function init() {
	if (props.config?.tail) {
		tailLog.value = props.config.tail
	} else {
		tailLog.value = false
	}

	if (tailLog.value) {
		changeTail(false)
	}

	readReq.latest = true
	await getContent(false)
}

onMounted(async () => {
	await init()
	nextTick(() => {
		if (logContainer.value) {
			logContainer.value.scrollTop = totalHeight.value
			containerHeight.value = logContainer.value.getBoundingClientRect().height
		}
	})
})

onUnmounted(() => {
	onCloseLog()
})

defineExpose({ changeTail, onDownload, clearLog })
</script>

<style scoped>
.log-container {
	height: calc(100vh - 405px);
	overflow-y: auto;
	overflow-x: auto;
	position: relative;
	background-color: #1a1a1a;
	margin-top: 10px;
	border-radius: 4px;
	padding: 8px;
}

.log-spacer {
	position: relative;
	width: 100%;
}

.log-item {
	position: absolute;
	width: 100%;
	padding: 2px 8px;
	color: #ffffff;
	box-sizing: border-box;
	white-space: nowrap;
	line-height: 1.5;
}

.log-item span {
	font-size: 14px;
	font-weight: 400;
	color: #ffffff;
	font-family: "Monaco", "Menlo", "Ubuntu Mono", "Consolas", "source-code-pro", monospace;
}

/* 自定义滚动条样式 */
.log-container::-webkit-scrollbar {
	width: 8px;
	height: 8px;
}

.log-container::-webkit-scrollbar-track {
	background: #2a2a2a;
	border-radius: 4px;
}

.log-container::-webkit-scrollbar-thumb {
	background: #4a4a4a;
	border-radius: 4px;
}

.log-container::-webkit-scrollbar-thumb:hover {
	background: #5a5a5a;
}
</style>
