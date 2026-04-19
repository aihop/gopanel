<template>
	<div>
		<n-drawer
			v-model:show="deleteVisible"
			:destroy-on-close="true"
			:close-on-click-modal="false"
			:close-on-press-escape="false"
			size="30%"
		>
			<n-drawer-content>
				<template #header>
					<DrawerHeader :header="$t('container.imageDelete')" :back="handleClose" />
				</template>
				<n-form @submit.prevent :model="form" label-position="top">
					<n-row type="flex" justify="center">
						<n-col :span="22">
							<n-form-item :label="$t('container.tag')" prop="tagName">
								<div style="width: 100%">
									<n-checkbox
										v-model="deleteAll"
										:indeterminate="isIndeterminate"
										@change="handleCheckAllChange"
									>
										{{ $t("container.removeAll") }}
									</n-checkbox>
								</div>
								<n-checkbox-group v-model="form.deleteTags" @change="handleCheckedChange">
									<div>
										<n-checkbox
											style="width: 100%"
											v-for="item in form.tags"
											:key="item"
											:value="item"
											:label="item"
										/>
									</div>
								</n-checkbox-group>
							</n-form-item>
						</n-col>
					</n-row>
				</n-form>
				<template #footer>
					<span class="dialog-footer">
						<n-button @click="deleteVisible = false">{{ $t("commons.button.cancel") }}</n-button>
						<n-button type="primary" :disabled="form.deleteTags.length === 0" @click="batchDelete()">
							{{ $t("commons.button.delete") }}
						</n-button>
					</span>
				</template>
			</n-drawer-content>
		</n-drawer>

		<OpDialog ref="opRef" @search="onSearch" @cancel="handleClose" />
	</div>
</template>
<script lang="ts" setup>
import { reactive, ref } from "vue"
import { imageRemove } from "@/api/modules/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import OpDialog from "@/components/OpDialog.vue"
import { t } from "@/i18n"

const deleteVisible = ref(false)
const form = reactive({
	id: "",
	force: false,
	tags: [] as Array<string>,
	deleteTags: [] as Array<string>
})

const deleteAll = ref()
const isIndeterminate = ref(true)
const opRef = ref()

interface DialogProps {
	id: string
	isUsed: boolean
	tags: Array<string>
}
const acceptParams = (params: DialogProps) => {
	deleteAll.value = false
	deleteVisible.value = true
	form.deleteTags = []
	form.id = params.id
	form.tags = params.tags
	form.force = !params.isUsed
}
const handleClose = () => {
	deleteVisible.value = false
}
const emit = defineEmits<{ (e: "search"): void }>()

const onSearch = () => {
	emit("search")
}

const handleCheckAllChange = (val: boolean) => {
	form.deleteTags = val ? form.tags : []
	isIndeterminate.value = false
}
const handleCheckedChange = (value: string[]) => {
	const checkedCount = value.length
	deleteAll.value = checkedCount === form.tags.length
	isIndeterminate.value = checkedCount > 0 && checkedCount < form.tags.length
}

const batchDelete = async () => {
	let names = []
	let showNames = []
	if (deleteAll.value) {
		names.push(form.id)
		for (const item of form.deleteTags) {
			showNames.push(item)
		}
	} else {
		for (const item of form.deleteTags) {
			names.push(item)
			showNames.push(item)
		}
	}
	opRef.value.acceptParams({
		title: t("commons.button.delete"),
		names: showNames,
		msg: t("commons.msg.operatorHelper", [t("container.image"), t("commons.button.delete")]),
		api: imageRemove,
		params: { names: names, force: form.force }
	})
}

defineExpose({
	acceptParams
})
</script>
