<template>
	<div>
		<n-popover placement="bottom-start" trigger="click">
			<template #trigger>
				<n-button round class="timer-button">
					{{ $t("commons.table.tableSetting") }}
				</n-button>
			</template>
			<div style="margin-left: 15px; white-space: nowrap">
				<span>{{ $t("commons.table.refreshRate") }}</span>
				<n-select
					v-model:value="refreshRate"
					@update:value="changeRefresh"
					:options="rateOptions"
					style="margin-left: 5px; width: 120px"
				/>
			</div>
		</n-popover>
	</div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue"
import { useI18n } from "vue-i18n"
import { NPopover, NButton, NSelect } from "naive-ui"

defineOptions({ name: "TableSetting" })
const props = defineProps<{ title?: string }>()
const emit = defineEmits<{
	(e: "search"): void
}>()

const { t } = useI18n()
const refreshRate = ref(0)
const rates = [0, 5, 10, 30, 60, 120, 300]
const rateOptions = computed(() =>
	rates.map(rate => ({
		label: t("commons.table.refreshRateUnit", rate),
		value: rate
	}))
)

let timer: ReturnType<typeof setInterval> | null = null
function changeRefresh() {
	if (timer) clearInterval(timer)
	if (refreshRate.value) {
		timer = setInterval(() => emit("search"), refreshRate.value * 1000)
	} else {
		timer = null
	}
	props.title && localStorage.setItem(props.title, `${refreshRate.value}`)
}

onMounted(() => {
	if (props.title) {
		refreshRate.value = Number(localStorage.getItem(props.title) || 0)
	}
	changeRefresh()
})
onUnmounted(() => {
	timer && clearInterval(timer)
	props.title && localStorage.setItem(props.title, `${refreshRate.value}`)
})
</script>

<style scoped lang="scss">
.timer-button {
	float: right;
}
</style>
