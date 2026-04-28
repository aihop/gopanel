<template>
  <n-drawer
    v-model:show="monitorVisible"
    :width="'50%'"
    :close-on-esc="false"
    :mask-closable="false"
    :trap-focus="false"
    :block-scroll="false"
    @update:show="
			val => {
				if (!val) handleClose()
			}
		"
  >
    <n-drawer-content
      :title="$t('container.monitor')"
      closable
    >
      <template #header>
        <DrawerHeader
          :header="$t('container.monitor')"
          :resource="title"
          :back="handleClose"
        />
      </template>
      <n-form label-placement="top">
        <div
          v-if="runtimeSummary"
          class="mb-4 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
        >
          {{ runtimeSummary }}
        </div>
        <n-form-item :label="$t('container.refreshTime')">
          <n-select
            v-model:value="timeInterval"
            :options="[
							{ label: '3s', value: 3 },
							{ label: '5s', value: 5 },
							{ label: '10s', value: 10 },
							{ label: '30s', value: 30 },
							{ label: '60s', value: 60 }
						]"
            @update:value="changeTimer"
            class="w-40"
          />
        </n-form-item>
      </n-form>
      <div class="space-y-4">
        <div
          v-for="chart in chartCards"
          :key="chart.key"
          class="rounded-[24px] border border-slate-200/80 bg-base-100 p-5 shadow-[0_18px_48px_rgba(15,23,42,0.06)]"
        >
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <div class="text-lg font-semibold fg-base-100">{{ chart.title }}</div>
              <div class="mt-1 text-xs font-medium uppercase tracking-[0.16em] text-slate-400">
                Unit · {{ chart.unit }}
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-3 text-xs text-slate-500">
              <div
                v-for="item in chart.series"
                :key="item.name"
                class="inline-flex items-center gap-2 rounded-full bg-slate-100 px-3 py-1.5"
              >
                <span
                  class="h-2.5 w-2.5 rounded-full"
                  :style="{ backgroundColor: item.color }"
                />
                <span>{{ item.name }}</span>
              </div>
            </div>
          </div>
          <SvgTrendChart
            :x-labels="chart.labels"
            :tooltip-labels="chart.labels"
            :series="chart.series"
            :y-formatter="value => `${value}${chart.unit}`"
            :height="320"
          />
        </div>
      </div>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, ref } from "vue"
import { useI18n } from "vue-i18n"
import { containerStatsGetAPI } from "@/api/modules/container"
import SvgTrendChart from "@/components/monitor/SvgTrendChart.vue"
import { dateFormatForSecond } from "@/utils/util"
import DrawerHeader from "@/components/DrawerHeader.vue"

const { t } = useI18n()

const title = ref()
const monitorVisible = ref(false)
const timeInterval = ref(5)
const runtimeSummary = ref("")
let timer: ReturnType<typeof setInterval> | null = null
let isInit = ref<boolean>(true)
interface DialogProps {
	containerID: string
	container: string
	runtimeSummary?: string
}
const dialogData = ref<DialogProps>({
	containerID: "",
	container: ""
})

const acceptParams = async (params: DialogProps): Promise<void> => {
	monitorVisible.value = true
	dialogData.value.containerID = params.containerID
	title.value = params.container
	runtimeSummary.value = params.runtimeSummary || ""
	cpuDatas.value = []
	memDatas.value = []
	cacheDatas.value = []
	ioReadDatas.value = []
	ioWriteDatas.value = []
	netTxDatas.value = []
	netRxDatas.value = []
	timeDatas.value = []
	timeInterval.value = 5
	isInit.value = true
	loadData()
	timer = setInterval(async () => {
		if (monitorVisible.value) {
			isInit.value = false
			loadData()
		}
	}, 1000 * timeInterval.value)
}

const cpuDatas = ref<Array<number>>([])
const memDatas = ref<Array<number>>([])
const cacheDatas = ref<Array<number>>([])
const ioReadDatas = ref<Array<number>>([])
const ioWriteDatas = ref<Array<number>>([])
const netTxDatas = ref<Array<number>>([])
const netRxDatas = ref<Array<number>>([])
const timeDatas = ref<Array<string>>([])

const createSeries = (label: string, data: number[], color: string) => ({
	name: label,
	values: [...data],
	color,
	fillFrom: `${color}33`,
	fillTo: `${color}08`,
	valueFormatter: (value: number) => value.toFixed(2)
})

const chartCards = computed(() => [
	{
		key: "cpu",
		title: "CPU",
		unit: "%",
		labels: [...timeDatas.value],
		series: [createSeries("CPU", cpuDatas.value, "#14b8a6")]
	},
	{
		key: "memory",
		title: t("monitor.memory"),
		unit: "MB",
		labels: [...timeDatas.value],
		series: [
			createSeries(t("monitor.memory"), memDatas.value, "#3b82f6"),
			createSeries(t("container.cache"), cacheDatas.value, "#84cc16")
		]
	},
	{
		key: "io",
		title: `${t("monitor.disk")} IO`,
		unit: "MB",
		labels: [...timeDatas.value],
		series: [
			createSeries(t("monitor.read"), ioReadDatas.value, "#f59e0b"),
			createSeries(t("monitor.write"), ioWriteDatas.value, "#a855f7")
		]
	},
	{
		key: "network",
		title: t("monitor.network"),
		unit: "KB",
		labels: [...timeDatas.value],
		series: [
			createSeries(t("monitor.up"), netTxDatas.value, "#0d9488"),
			createSeries(t("monitor.down"), netRxDatas.value, "#ef4444")
		]
	}
])

const changeTimer = () => {
	clearInterval(Number(timer))
	timer = setInterval(async () => {
		if (monitorVisible.value) {
			loadData()
		}
	}, 1000 * timeInterval.value)
}

const loadData = async () => {
	const res = await containerStatsGetAPI(dialogData.value.containerID)
	if (!res.data) return
	cpuDatas.value.push(Number(res.data.cpuPercent.toFixed(2)))
	if (cpuDatas.value.length > 20) {
		cpuDatas.value.splice(0, 1)
	}
	memDatas.value.push(Number(res.data.memory.toFixed(2)))
	if (memDatas.value.length > 20) {
		memDatas.value.splice(0, 1)
	}
	cacheDatas.value.push(Number(res.data.cache.toFixed(2)))
	if (cacheDatas.value.length > 20) {
		cacheDatas.value.splice(0, 1)
	}
	ioReadDatas.value.push(Number(res.data.ioRead.toFixed(2)))
	if (ioReadDatas.value.length > 20) {
		ioReadDatas.value.splice(0, 1)
	}
	ioWriteDatas.value.push(Number(res.data.ioWrite.toFixed(2)))
	if (ioWriteDatas.value.length > 20) {
		ioWriteDatas.value.splice(0, 1)
	}
	netTxDatas.value.push(Number(res.data.networkTX.toFixed(2)))
	if (netTxDatas.value.length > 20) {
		netTxDatas.value.splice(0, 1)
	}
	netRxDatas.value.push(Number(res.data.networkRX.toFixed(2)))
	if (netRxDatas.value.length > 20) {
		netRxDatas.value.splice(0, 1)
	}
	timeDatas.value.push(dateFormatForSecond(res.data.shotTime))
	if (timeDatas.value.length > 20) {
		timeDatas.value.splice(0, 1)
	}
}
const handleClose = async () => {
	monitorVisible.value = false
	runtimeSummary.value = ""
	clearInterval(Number(timer))
	timer = null
}

onBeforeUnmount(() => {
	handleClose()
})

defineExpose({
	acceptParams
})
</script>
