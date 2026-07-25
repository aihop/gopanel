<script setup lang="ts">
import type { NodeItem, NodeSaveParams } from "@/api/modules/node"
import { nodeCreateAPI, nodeProbeAPI, nodeProbeDraftAPI, nodeUpdateAPI } from "@/api/modules/node"
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

/** 已保存的节点是否已经配好令牌。用于在编辑态说明"输入框空着不代表没有令牌" */
const hasStoredToken = computed(() => !!props.node?.hasToken)

const expectedLen = computed(() => props.node?.tokenLenExpected || 40)

/**
 * 已存令牌长度是否可疑。长度不等于签发长度，几乎总是粘贴漏了或填的不是节点签发的串——
 * 这种情况下测试连接可能因为用的是当场输入的值而通过，保存后采集却一直失败，很难自查。
 */
const storedTokenSuspicious = computed(() => {
	const node = props.node
	if (!node?.hasToken || !node.tokenLen) return false
	return node.tokenLen !== node.tokenLenExpected
})

/** 当前输入框里的值长度是否可疑（留空不算） */
const inputTokenSuspicious = computed(() => {
	const value = form.value.accessToken.trim()
	if (!value) return false
	return value.length !== expectedLen.value
})

/** 连接相关字段是否被改动过（令牌留空时决定能不能拿已存配置去测连接） */
const connectionFieldsChanged = computed(() => {
	if (!props.node) return false
	return (
		form.value.addr.trim() !== props.node.addr ||
		(form.value.entrance || "").trim() !== (props.node.entrance || "") ||
		!!form.value.skipVerify !== !!props.node.skipVerify
	)
})

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
	const blankToken = isEdit.value && !form.value.accessToken.trim()
	// 令牌留空时只能用已保存的配置去探测。此时如果连接相关字段被改过，
	// 测出来的是旧地址的结果，会误导——让用户要么重输令牌，要么先保存
	if (blankToken && connectionFieldsChanged.value) {
		message.warning(t("node.form.testAfterSave"))
		return
	}
	testing.value = true
	try {
		const useSaved = blankToken && !!props.node?.id
		if (useSaved) {
			const res = await nodeProbeAPI(props.node!.id)
			if (res.code === 0) {
				message.success(
					t("node.form.testOk", {
						hostname: res.data?.summary?.hostname || "-",
						version: res.data?.version || "-"
					})
				)
			} else {
				message.error(res.msg || t("node.form.testFailed"))
			}
			return
		}
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

			<n-form-item :required="!isEdit">
				<template #label>
					<div class="flex items-center gap-2">
						<span>{{ t("node.form.token") }}</span>
						<!-- 密文不回显，输入框在编辑态永远是空的。不给个明确标记，用户会以为令牌没保存上 -->
						<n-tag
							v-if="isEdit && hasStoredToken"
							size="tiny"
							:type="storedTokenSuspicious ? 'warning' : 'success'"
							:bordered="false"
						>
							{{ t("node.form.tokenStored") }} · {{ props.node?.tokenLen }}
						</n-tag>
					</div>
				</template>
				<n-input
					v-model:value="form.accessToken"
					type="password"
					show-password-on="click"
					:placeholder="isEdit ? t('node.form.tokenKeepHint') : t('node.form.tokenPlaceholder')"
				/>
			</n-form-item>

			<!-- 长度对不上时明确点出来。这是"测试通过但采集一直失败"最常见的原因 -->
			<n-alert v-if="inputTokenSuspicious" type="warning" :show-icon="false" class="-mt-2 mb-3 text-xs">
				{{ t("node.form.tokenLenWarn", { actual: form.accessToken.trim().length, expected: expectedLen }) }}
			</n-alert>
			<n-alert
				v-else-if="storedTokenSuspicious && !form.accessToken.trim()"
				type="warning"
				:show-icon="false"
				class="-mt-2 mb-3 text-xs"
			>
				{{ t("node.form.tokenStoredLenWarn", { actual: props.node?.tokenLen, expected: expectedLen }) }}
			</n-alert>

			<n-form-item :label="t('node.form.entrance')">
				<n-input v-model:value="form.entrance" :placeholder="t('node.form.entrancePlaceholder')" />
			</n-form-item>

			<n-form-item :label="t('node.form.sort')">
				<n-input-number v-model:value="form.sort" :min="0" class="w-32" />
				<span class="ml-2 text-xs opacity-60">{{ t("node.form.sortHint") }}</span>
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
