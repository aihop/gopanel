<template>
	<n-tag :type="getType(status)" round>
		<span class="flx-align-center">
			{{ $t("commons.status." + status) }}
			<Icon v-if="loadingIcon(status)" class="is-loading ml-1" name="line-md:loading-twotone-loop" :size="14" />
		</span>
	</n-tag>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui"
import { onMounted, ref } from "vue"

const props = defineProps({
	status: {
		type: String,
		default: "running"
	}
})
let status = ref("running")

const getType = (status: string) => {
	if (status.includes("error") || status.includes("err")) {
		return "error"
	}
	switch (status) {
		case "running":
			return "success"
		case "stopped":
			return "error"
		case "unhealthy":
		case "paused":
		case "exited":
		case "dead":
		case "removing":
			return "warning"
		default:
			return "info"
	}
}

const loadingStatus = [
	"installing",
	"building",
	"restarting",
	"upgrading",
	"rebuilding",
	"recreating",
	"creating",
	"starting",
	"removing",
	"applying"
]

const loadingIcon = (status: string): boolean => {
	return loadingStatus.indexOf(status) > -1
}

onMounted(() => {
	status.value = props.status.toLocaleLowerCase()
})
</script>
