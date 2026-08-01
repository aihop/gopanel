<script setup lang="ts">
import type { RemoteContainer } from "@/api/modules/nodeProxy"
import { remoteContainerListAPI, remoteContainerOperateAPI } from "@/api/modules/nodeProxy"
import { t } from "@/i18n"
import { useDialog, useMessage } from "naive-ui"
import { onMounted, ref, watch } from "vue"
import ContainerWebsiteDialog from "@/views/Container/container/ContainerWebsiteDialog.vue"

const props = defineProps<{
	nodeId: number
	/** 节点名，用于高危操作确认里点明操作对象 */
	nodeName: string
	/** 未配置控制令牌时只能看不能动 */
	canControl: boolean
}>()

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const rows = ref<RemoteContainer[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref("")
const operatingName = ref("")
const bindDialogRef = ref()

async function fetchData() {
	loading.value = true
	try {
		const res = await remoteContainerListAPI(props.nodeId, {
			page: page.value,
			limit: pageSize.value,
			name: keyword.value
		})
		rows.value = res.data?.items || []
		total.value = res.data?.total || 0
	} catch {
		// 失败提示由 axios 拦截器统一弹出
		rows.value = []
		total.value = 0
	} finally {
		loading.value = false
	}
}

/** 高危操作必须带节点名确认——这是远程操作最容易搞错对象的地方 */
function operate(row: RemoteContainer, operation: string) {
	const run = async () => {
		operatingName.value = row.name
		try {
			await remoteContainerOperateAPI(props.nodeId, [row.name], operation)
			message.success(t("commons.msg.operationSuccess"))
			await fetchData()
		} catch {
			// 拦截器已提示
		} finally {
			operatingName.value = ""
		}
	}
	if (operation === "start") {
		run()
		return
	}
	dialog.warning({
		title: t("commons.button.confirm"),
		content: t("node.workspace.confirmOperate", {
			operation: t(`node.workspace.op.${operation}`),
			name: row.name,
			node: props.nodeName
		}),
		positiveText: t("commons.button.confirm"),
		negativeText: t("commons.button.cancel"),
		onPositiveClick: run
	})
}

/** 搜索：重置到第一页再拉。模板里不要写多语句表达式，Vue 编译器不接受 */
function search() {
	page.value = 1
	fetchData()
}

watch(
	() => props.nodeId,
	() => {
		page.value = 1
		fetchData()
	}
)

onMounted(fetchData)
</script>

<template>
	<div class="flex flex-col gap-3">
		<n-alert v-if="!canControl" type="warning" :show-icon="false" class="text-xs">
			{{ t("node.workspace.noControl") }}
		</n-alert>

		<div class="flex items-center gap-2">
			<n-input
				v-model:value="keyword"
				clearable
				size="small"
				class="max-w-60"
				:placeholder="t('node.workspace.searchContainer')"
				@keyup.enter="search"
			/>
			<n-button size="small" @click="search">
				{{ t("commons.button.search") }}
			</n-button>
			<n-button size="small" :loading="loading" @click="fetchData">{{ t("commons.button.refresh") }}</n-button>
		</div>

		<n-spin :show="loading">
			<n-empty v-if="!rows.length && !loading" :description="t('node.workspace.noContainer')" />
			<div v-else class="flex flex-col gap-2">
				<div v-for="row of rows" :key="row.containerID" class="rounded border p-2" style="border-color: var(--border-color)">
					<div class="flex items-center justify-between gap-2">
						<div class="min-w-0">
							<div class="truncate text-sm font-medium">{{ row.name }}</div>
							<div class="truncate text-xs opacity-60">{{ row.imageName }}</div>
						</div>
						<n-tag size="tiny" :type="row.state === 'running' ? 'success' : 'default'" :bordered="false">
							{{ row.state }}
						</n-tag>
					</div>
					<div v-if="canControl" class="mt-2 flex gap-3 text-xs">
						<a
							v-if="row.state === 'running'"
							class="cursor-pointer text-primary"
							@click="bindDialogRef?.acceptParams({ containerId: row.containerID, containerName: row.name, nodeId })"
						>
							{{ t("container.bindWebsite") }}
						</a>
						<a
							v-if="row.state !== 'running'"
							class="cursor-pointer text-primary"
							@click="operate(row, 'start')"
						>
							{{ t("node.workspace.op.start") }}
						</a>
						<a v-if="row.state === 'running'" class="cursor-pointer text-primary" @click="operate(row, 'restart')">
							{{ t("node.workspace.op.restart") }}
						</a>
						<a v-if="row.state === 'running'" class="cursor-pointer text-red-500" @click="operate(row, 'stop')">
							{{ t("node.workspace.op.stop") }}
						</a>
						<span v-if="operatingName === row.name" class="opacity-60">
							{{ t("node.workspace.operating") }}
						</span>
					</div>
				</div>

				<n-pagination
					v-if="total > pageSize"
					v-model:page="page"
					:page-size="pageSize"
					:item-count="total"
					size="small"
					class="justify-end"
					@update:page="fetchData"
				/>
			</div>
		</n-spin>
		<ContainerWebsiteDialog ref="bindDialogRef" @success="fetchData" />
	</div>
</template>
