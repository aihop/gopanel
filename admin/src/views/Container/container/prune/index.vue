<template>
	<n-modal
		v-model:show="dialogVisible"
		preset="dialog"
		:title="$t('container.containerPrune')"
		:mask-closable="false"
		:close-on-esc="false"
		style="width: 400px"
	>
		<div>
			<ul class="help-ul">
				<li class="lineClass" style="color: red">{{ $t("container.containerPruneHelper1") }}</li>
				<!-- <li class="lineClass">{{ $t('container.containerPruneHelper2') }}</li> -->
				<li class="lineClass">{{ $t("container.containerPruneHelper3") }}</li>
			</ul>
		</div>
		<template #action>
			<n-button :disabled="loading" @click="dialogVisible = false">
				{{ $t("commons.button.cancel") }}
			</n-button>
			<n-button :disabled="loading" type="primary" @click="onClean">
				{{ $t("commons.button.confirm") }}
			</n-button>
		</template>
	</n-modal>
</template>

<script lang="ts" setup>
import { containerPrune } from "@/api/modules/container"
import { t } from "@/i18n"
import { MsgSuccess } from "@/utils/message"
import { ref } from "vue"
import { computeSize } from "@/utils/util"
import { NModal, NButton } from "naive-ui"

const loading = ref(false)
const dialogVisible = ref<boolean>(false)

const emit = defineEmits<{ (e: "search"): void }>()

const onClean = async () => {
	loading.value = true
	let params = {
		pruneType: "container",
		withTagAll: false
	}
	await containerPrune(params)
		.then(res => {
			loading.value = false
			MsgSuccess(
				t("container.cleanSuccessWithSpace", [res.data.deletedNumber, computeSize(res.data.spaceReclaimed)])
			)
			dialogVisible.value = false
			emit("search")
		})
		.catch(() => {
			loading.value = false
		})
}

const acceptParams = (): void => {
	dialogVisible.value = true
}

defineExpose({
	acceptParams
})
</script>

<style lang="scss" scoped>
.lineClass {
	line-height: 30px;
}
</style>
