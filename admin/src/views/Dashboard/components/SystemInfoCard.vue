<template>
	<n-card :title="$t('home.systemInfo')" :bordered="false" class="shadow">
		<n-descriptions :column="1" bordered label-placement="left">
			<n-descriptions-item :label="$t('home.hostname')">
				<n-tooltip
					v-if="props.baseInfo.hostname.length > 30"
					:content="props.baseInfo.hostname"
					placement="bottom"
				>
					{{ props.baseInfo.hostname.substring(0, 27) + "..." }}
				</n-tooltip>
				<span v-else>{{ props.baseInfo.hostname }}</span>
			</n-descriptions-item>

			<n-descriptions-item :label="$t('home.platformVersion')">
				{{
					props.baseInfo.platformVersion
						? props.baseInfo.platform
						: props.baseInfo.platform + "-" + props.baseInfo.platformVersion
				}}
			</n-descriptions-item>

			<n-descriptions-item :label="$t('home.kernelVersion')">
				<n-tooltip
					v-if="props.baseInfo.kernelVersion.length > 30"
					:content="props.baseInfo.kernelVersion"
					placement="bottom"
				>
					{{ props.baseInfo.kernelVersion.substring(0, 27) + "..." }}
				</n-tooltip>
				<span v-else>{{ props.baseInfo.kernelVersion }}</span>
			</n-descriptions-item>

			<n-descriptions-item :label="$t('home.kernelArch')">
				{{ props.baseInfo.kernelArch }}
			</n-descriptions-item>

			<n-descriptions-item
				v-if="props.baseInfo.ipv4Addr && props.baseInfo.ipv4Addr !== 'IPNotFound'"
				:label="$t('home.ip')"
			>
				{{ props.baseInfo.ipv4Addr }}
			</n-descriptions-item>

			<n-descriptions-item
				v-if="props.baseInfo.systemProxy && props.baseInfo.systemProxy !== 'noProxy'"
				:label="$t('home.proxy')"
			>
				{{ props.baseInfo.systemProxy }}
			</n-descriptions-item>

			<n-descriptions-item :label="$t('home.uptime')">
				{{ currentInfo.timeSinceUptime }}
			</n-descriptions-item>

			<n-descriptions-item :label="$t('home.runningTime')">
				{{ loadUpTime(currentInfo.uptime) }}
			</n-descriptions-item>
		</n-descriptions>
	</n-card>
</template>

<script setup lang="ts">
import type { Dashboard } from "@/api/interface/dashboard"
import { ref } from "vue"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

// 定义 props
interface Props {
	baseInfo: Dashboard.BaseInfo
}

const props = defineProps<Props>()

const currentInfo = ref({
	timeSinceUptime: "",
	uptime: 0
})

// 计算运行时长
const loadUpTime = (uptime: number) => {
	const days = Math.floor(uptime / 86400)
	const hours = Math.floor((uptime % 86400) / 3600)
	const minutes = Math.floor((uptime % 3600) / 60)
	const seconds = uptime % 60

	let result = ""
	if (days > 0) {
		result += `${days}天`
	}
	if (hours > 0) {
		result += `${hours}小时`
	}
	if (minutes > 0) {
		result += `${minutes}分钟`
	}
	if (seconds > 0 || result === "") {
		result += `${seconds}秒`
	}
	return result
}

const acceptParams = (current: Dashboard.CurrentInfo): void => {
	currentInfo.value.timeSinceUptime = current.timeSinceUptime
	currentInfo.value.uptime = current.uptime
}

defineExpose({
	acceptParams
})
</script>

<style scoped lang="scss">
.shadow {
	box-shadow:
		0 1px 3px 0 rgba(0, 0, 0, 0.1),
		0 1px 2px 0 rgba(0, 0, 0, 0.06);
}
</style>
