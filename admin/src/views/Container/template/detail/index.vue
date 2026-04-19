<template>
	<n-drawer
		v-model:show="detailVisible"
		:destroy-on-close="true"
		:close-on-click-modal="false"
		:close-on-press-escape="false"
		size="50%"
	>
		<n-drawer-content>
			<template #header>
				<DrawerHeader :header="$t('commons.button.view')" :back="handleClose" />
			</template>
			<FtEditor v-model="detailInfo" height="calc(100vh - 160px)" :readonly="true" />
			<template #footer>
				<span class="dialog-footer">
					<n-button @click="detailVisible = false">{{ $t("commons.button.cancel") }}</n-button>
				</span>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>

<script lang="ts" setup>
import { ref } from "vue"
import FtEditor from "@/components/FtEditor/index.vue"
import DrawerHeader from "@/components/DrawerHeader.vue"

const detailVisible = ref(false)
const detailInfo = ref()

interface DialogProps {
	content: string
}
const acceptParams = (params: DialogProps): void => {
	detailInfo.value = params.content
	detailVisible.value = true
}

const handleClose = () => {
	detailVisible.value = false
}

defineExpose({
	acceptParams
})
</script>
