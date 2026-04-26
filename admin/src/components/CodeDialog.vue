<template>
	<n-drawer
		v-model:show="codeVisible"
		:destroy-on-close="true"
		:close-on-click-modal="false"
		:close-on-press-escape="false"
		:width="'50%'"
		class="fullcalendar-drawer"
	>
		<n-drawer-content :title="$t('commons.button.log')" :native-scrollbar="false" closable>
			<template #header>
				<DrawerHeader :header="header" :back="handleClose" />
			</template>
			<div
				v-if="summary"
				class="mb-4 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
			>
				{{ summary }}
			</div>
			<FtEditor v-model="detailInfo" height="calc(100vh - 160px)" :readonly="true" />
			<template #footer>
				<n-space justify="end">
					<n-button @click="codeVisible = false">{{ $t("commons.button.cancel") }}</n-button>
				</n-space>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script lang="ts" setup>
import { ref } from "vue"
import FtEditor from "@/components/FtEditor/index.vue"
import DrawerHeader from "@/components/DrawerHeader.vue"
const header = ref()
const detailInfo = ref()
const summary = ref("")
const codeVisible = ref(false)

interface DialogProps {
	header: string
	detailInfo: string
	summary?: string
}

const acceptParams = (props: DialogProps): void => {
	header.value = props.header
	detailInfo.value = props.detailInfo
	summary.value = props.summary || ""
	codeVisible.value = true
}

const handleClose = () => {
	codeVisible.value = false
	summary.value = ""
}

defineExpose({
	acceptParams
})
</script>
