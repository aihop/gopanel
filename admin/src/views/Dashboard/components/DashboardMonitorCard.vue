<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { getIOOptions, getNetworkOptions } from "@/api/modules/host"
import { computeSize, computeSizeFromKBs } from "@/utils/util"
import {
	currentChartInfo,
	currentInfo,
	ioReadBytes,
	ioWriteBytes,
	netBytesRecvs,
	netBytesSents,
	searchInfo,
	timeIODatas,
	timeNetDatas
} from "../Index"
import TrendSvg from "./TrendSvg.vue"
import { dashboardMessages } from "../dashboardMessages"

const emit = defineEmits<{
	reload: [range: "network" | "io"]
}>()

const { t } = useI18n({ messages: dashboardMessages })
const message = useMessage()
const chartOption = ref<"network" | "io">("network")
const ioOptions = ref<string[]>([])
const netOptions = ref<string[]>([])
const optionsLoading = ref(true)

const netSelectOptions = computed(() =>
	netOptions.value.map(value => ({
		label: value === "all" ? t("commons.table.all") : value,
		value
	}))
)

const ioSelectOptions = computed(() =>
	ioOptions.value.map(value => ({
		label: value === "all" ? t("commons.table.all") : value,
		value
	}))
)

async function loadOptions() {
	try {
		const [networkResponse, ioResponse] = await Promise.all([getNetworkOptions(), getIOOptions()])
		netOptions.value = networkResponse.data || []
		ioOptions.value = ioResponse.data || []
		searchInfo.netOption = netOptions.value[0] || "all"
		searchInfo.ioOption = ioOptions.value[0] || "all"
	} catch {
		message.error(t("dashboardControl.monitorOptionsLoadFailed"))
	} finally {
		optionsLoading.value = false
	}
}

onMounted(loadOptions)
</script>

<template>
	<n-card :title="$t('menu.monitor')" :bordered="false" class="shadow 2xl:col-start-1">
		<template #header-extra>
			<n-space>
				<n-radio-group v-model:value="chartOption">
					<n-radio-button value="network">{{ $t("home.network") }}</n-radio-button>
					<n-radio-button value="io">{{ $t("home.io") }}</n-radio-button>
				</n-radio-group>

				<n-select
					v-if="chartOption === 'network'"
					v-model:value="searchInfo.netOption"
					:options="netSelectOptions"
					:loading="optionsLoading"
					placeholder=""
					style="width: 200px"
					@update:value="emit('reload', 'network')"
				/>

				<n-select
					v-if="chartOption === 'io'"
					v-model:value="searchInfo.ioOption"
					:options="ioSelectOptions"
					:loading="optionsLoading"
					placeholder=""
					style="width: 200px"
					@update:value="emit('reload', 'io')"
				/>
			</n-space>
		</template>

		<n-space v-if="chartOption === 'network'" class="monitor-tags">
			<n-tag type="info">{{ $t("monitor.up") }}: {{ computeSizeFromKBs(currentChartInfo.netBytesSent) }}</n-tag>
			<n-tag type="info">{{ $t("monitor.down") }}: {{ computeSizeFromKBs(currentChartInfo.netBytesRecv) }}</n-tag>
			<n-tag type="info">{{ $t("home.totalSend") }}: {{ computeSize(currentInfo.netBytesSent) }}</n-tag>
			<n-tag type="info">{{ $t("home.totalRecv") }}: {{ computeSize(currentInfo.netBytesRecv) }}</n-tag>
		</n-space>

		<n-space v-if="chartOption === 'io'" class="monitor-tags">
			<n-tag type="info">{{ $t("monitor.read") }}: {{ currentChartInfo.ioReadBytes }} MB</n-tag>
			<n-tag type="info">{{ $t("monitor.write") }}: {{ currentChartInfo.ioWriteBytes }} MB</n-tag>
			<n-tag type="info">
				{{ $t("home.rwPerSecond") }}: {{ currentChartInfo.ioCount }} {{ $t("commons.units.time") }}/s
			</n-tag>
			<n-tag type="info">{{ $t("home.ioDelay") }}: {{ currentChartInfo.ioTime }} ms</n-tag>
		</n-space>

		<div v-if="chartOption === 'io'" class="mobile-monitor-chart mt-10">
			<TrendSvg
				:title="$t('home.io')"
				:metric="`${currentChartInfo.ioReadBytes} MB / ${currentChartInfo.ioWriteBytes} MB`"
				:points="ioReadBytes"
				:secondary-points="ioWriteBytes"
				:labels="timeIODatas"
				:primary-label="$t('monitor.read')"
				:secondary-label="$t('monitor.write')"
				:badge="`${$t('home.ioDelay')}: ${currentChartInfo.ioTime} ms`"
			/>
		</div>

		<div v-if="chartOption === 'network'" class="mobile-monitor-chart mt-10">
			<TrendSvg
				:title="$t('home.network')"
				:metric="`${computeSizeFromKBs(currentChartInfo.netBytesSent)} / ${computeSizeFromKBs(currentChartInfo.netBytesRecv)}`"
				:points="netBytesSents"
				:secondary-points="netBytesRecvs"
				:labels="timeNetDatas"
				:primary-label="$t('monitor.up')"
				:secondary-label="$t('monitor.down')"
				:badge="searchInfo.netOption"
			/>
		</div>
	</n-card>
</template>
