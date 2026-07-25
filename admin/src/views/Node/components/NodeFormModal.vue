<script setup lang="ts">
import type { NodeItem, NodeSaveParams } from "@/api/modules/node"
import { nodeCreateAPI, nodeProbeDraftAPI, nodeUpdateAPI } from "@/api/modules/node"
import { t } from "@/i18n"
import { useMessage } from "naive-ui"
import { computed, ref, watch } from "vue"

const props = defineProps<{
	show: boolean
	node: NodeItem | null
}>()

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "saved"): void
}>()

const message = useMessage()

const saving = ref(false)
const testing = ref(false)
const form = ref<NodeSaveParams>(emptyForm())

const isEdit = computed(() => !!props.node?.id)

const visible = computed({
	get: () => props.show,
	set: (value: boolean) => emit("update:show", value)
})

function emptyForm(): NodeSaveParams {
	return {
		name: "",
		addr: "",
		accessToken: "",
		entrance: "",
		skipVerify: false,
		isProd: false,
		sort: 0
	}
}

watch(
	() => props.show,
	value => {
		if (!value) return
		if (props.node) {
			form.value = {
				id: props.node.id,
				name: props.node.name,
				addr: props.node.addr,
				// 令牌不回显：后端只存密文且不出接口。留空提交表示保留原值
				accessToken: "",
				entrance: props.node.entrance,
				skipVerify: props.node.skipVerify,
				isProd: props.node.isProd,
				sort: props.node.sort
			}
		} else {
			form.value = emptyForm()
		}
	}
)

function validate(): string {
	if (!form.value.name.trim()) return t("node.form.nameRequired")
	if (!form.value.addr.trim()) return t("node.form.addrRequired")
	if (!/^https?:\/\//.test(form.value.addr.trim())) return t("node.form.addrScheme")
	// 新增时必须给令牌；编辑时留空表示不修改
	if (!isEdit.value && !form.value.accessToken.trim()) return t("node.form.tokenRequired")
	return ""
}

async function testConnection() {
	const invalid = validate()
	if (invalid) {
		message.error(invalid)
		return
	}
	if (isEdit.value && !form.value.accessToken.trim()) {
		message.warning(t("node.form.testNeedToken"))
		return
	}
	testing.value = true
	try {
		const res = await nodeProbeDraftAPI(form.value)
		if (res.code === 0) {
			message.success(t("node.form.testOk", { hostname: res.data?.hostname || "-", version: res.data?.version || "-" }))
		} else {
			message.error(res.msg || t("node.form.testFailed"))
		}
	} catch (e: any) {
		message.error(e?.message || t("node.form.testFailed"))
	} finally {
		testing.value = false
	}
}

async function submit() {
	const invalid = validate()
	if (invalid) {
		message.error(invalid)
		return
	}
	saving.value = true
	try {
		const res = isEdit.value ? await nodeUpdateAPI(form.value) : await nodeCreateAPI(form.value)
		if (res.code === 0) {
			message.success(t("commons.msg.operationSuccess"))
			emit("saved")
		} else {
			message.error(res.msg || t("node.saveFailed"))
		}
	} catch (e: any) {
		message.error(e?.message || t("node.saveFailed"))
	} finally {
		saving.value = false
	}
}
</script>

<template>
	<!-- 宽度必须用 style 内联，Naive UI 的 NModal 挂 Tailwind 宽度类无效 -->
	<n-modal
		v-model:show="visible"
		preset="card"
		style="width: 540px"
		:title="isEdit ? t('node.form.editTitle') : t('node.form.createTitle')"
	>
		<n-form label-placement="top">
			<n-form-item :label="t('node.form.name')" required>
				<n-input v-model:value="form.name" :placeholder="t('node.form.namePlaceholder')" />
			</n-form-item>

			<n-form-item :label="t('node.form.addr')" required>
				<n-input v-model:value="form.addr" placeholder="https://1.2.3.4:5470" />
			</n-form-item>

			<n-form-item :label="t('node.form.token')" :required="!isEdit">
				<n-input
					v-model:value="form.accessToken"
					type="password"
					show-password-on="click"
					:placeholder="isEdit ? t('node.form.tokenKeepHint') : t('node.form.tokenPlaceholder')"
				/>
			</n-form-item>

			<n-form-item :label="t('node.form.entrance')">
				<n-input v-model:value="form.entrance" :placeholder="t('node.form.entrancePlaceholder')" />
			</n-form-item>

			<div class="flex items-center gap-6">
				<n-checkbox v-model:checked="form.skipVerify">{{ t("node.form.skipVerify") }}</n-checkbox>
				<n-checkbox v-model:checked="form.isProd">{{ t("node.form.isProd") }}</n-checkbox>
			</div>

			<n-alert type="info" :show-icon="false" class="mt-3 text-xs">
				{{ t("node.form.tokenTip") }}
			</n-alert>
		</n-form>

		<template #footer>
			<div class="flex justify-between">
				<n-button :loading="testing" @click="testConnection">{{ t("node.form.test") }}</n-button>
				<div class="flex gap-2">
					<n-button @click="visible = false">{{ t("commons.button.cancel") }}</n-button>
					<n-button type="primary" :loading="saving" @click="submit">{{ t("commons.button.save") }}</n-button>
				</div>
			</div>
		</template>
	</n-modal>
</template>
