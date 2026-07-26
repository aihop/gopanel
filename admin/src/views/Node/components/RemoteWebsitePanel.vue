<script setup lang="ts">
import { remoteWebsiteListAPI } from "@/api/modules/nodeProxy"
import { t } from "@/i18n"
import { onMounted, ref } from "vue"

const props = defineProps<{
	nodeId: number
}>()

interface RemoteWebsite {
	id: number
	primaryDomain: string
	protocol: string
	type: string
	status: string
	expireDate?: string
	remark?: string
}

const loading = ref(false)
const rows = ref<RemoteWebsite[]>([])

async function fetchData() {
	loading.value = true
	try {
		const res = await remoteWebsiteListAPI(props.nodeId)
		// 站点列表接口走的是 Contextx，返回结构可能是数组或 { items }
		const data: any = res.data
		rows.value = Array.isArray(data) ? data : data?.items || []
	} catch {
		// 失败提示由拦截器统一弹出
		rows.value = []
	} finally {
		loading.value = false
	}
}

function siteUrl(row: RemoteWebsite): string {
	const scheme = (row.protocol || "http").toLowerCase().includes("https") ? "https" : "http"
	return `${scheme}://${row.primaryDomain}`
}

onMounted(fetchData)
</script>

<template>
	<div class="flex flex-col gap-3">
		<div class="flex items-center justify-between gap-2">
			<span class="text-xs opacity-60">{{ t("node.workspace.websiteHint") }}</span>
			<n-button size="small" :loading="loading" @click="fetchData">{{ t("commons.button.refresh") }}</n-button>
		</div>

		<n-spin :show="loading">
			<n-empty v-if="!rows.length && !loading" :description="t('node.workspace.noWebsite')" />
			<div v-else class="flex flex-col gap-2">
				<div
					v-for="row of rows"
					:key="row.id"
					class="rounded border p-2"
					style="border-color: var(--border-color)"
				>
					<div class="flex items-center justify-between gap-2">
						<div class="min-w-0">
							<div class="truncate text-sm font-medium">{{ row.primaryDomain }}</div>
							<div class="truncate text-xs opacity-60">{{ row.type }} · {{ row.protocol }}</div>
						</div>
						<div class="flex shrink-0 items-center gap-2">
							<n-tag size="tiny" :type="row.status === 'Running' ? 'success' : 'default'" :bordered="false">
								{{ row.status }}
							</n-tag>
							<!-- 站点跑在节点上，只能在新标签打开；主控代理的是管理接口，不代理站点流量 -->
							<a
								class="cursor-pointer text-xs text-primary"
								:href="siteUrl(row)"
								target="_blank"
								rel="noopener noreferrer"
							>
								{{ t("commons.button.open") }}
							</a>
						</div>
					</div>
				</div>
			</div>
		</n-spin>
	</div>
</template>
