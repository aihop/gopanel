<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from "vue"
import {
	NAlert,
	NButton,
	NCard,
	NCheckbox,
	NCollapse,
	NCollapseItem,
	NDataTable,
	NInputNumber,
	NModal,
	NProgress,
	NSpace,
	NSpin,
	NTag,
	useMessage,
	type DataTableColumns
} from "naive-ui"
import { cancelDiskScan, cleanDiskPaths, getDiskOverview, getDiskScanResult, startDiskScan } from "@/api/modules/disk"
import type { Disk } from "@/api/interface/disk"
import { computeSizeFromByte } from "@/utils/util"
import { useAuthStore } from "@/store/auth"

const message = useMessage()
const authStore = useAuthStore()

const disks = ref<Disk.DiskUsage[]>([])
const task = ref<Disk.ScanTask | null>(null)
const scanning = ref(false)
const cleaning = ref(false)
const checkedPaths = ref<string[]>([])
const confirmVisible = ref(false)
const confirmTruncate = ref(false)
const confirmChecked = ref(false)

// 扫描参数：默认只看 ≥100MB，够大才值得清
const scanRoots = ref("/")
const minSizeMB = ref(100)
const topN = ref(200)
const crossDevice = ref(false)

let eventSource: EventSource | null = null

const categoryMeta: Record<string, { label: string; type: "default" | "info" | "success" | "warning" | "error"; advice: string }> = {
	log: { label: "日志", type: "warning", advice: "建议用「清空」而不是删除：日志正被进程写入时，删除不会释放空间" },
	cache: { label: "缓存", type: "info", advice: "包管理/应用缓存，通常可安全删除" },
	container: { label: "容器", type: "error", advice: "容器镜像层，请到容器页面执行 prune，不要直接删" },
	archive: { label: "归档包", type: "default", advice: "压缩包，确认无用后可删除" },
	coredump: { label: "崩溃转储", type: "default", advice: "core dump，通常可安全删除" },
	temp: { label: "临时文件", type: "default", advice: "临时文件，通常可安全删除" },
	backup: { label: "备份", type: "warning", advice: "备份文件，删除前请确认已有其他副本" },
	other: { label: "其他", type: "default", advice: "无法自动归类，请自行确认用途" }
}

const fetchOverview = async () => {
	try {
		const res = await getDiskOverview()
		disks.value = res.data || []
	} catch (error: any) {
		message.error(error.message || "获取磁盘信息失败")
	}
}

const stopStream = () => {
	if (eventSource) {
		eventSource.close()
		eventSource = null
	}
}

const openStream = (taskId: string) => {
	stopStream()
	const apiUrl = (window as any).__VITE_API_URL__ || "/api"
	const safeToken = encodeURIComponent(authStore.auth || "")
	eventSource = new EventSource(
		`${apiUrl}/host/disk/scan/stream?taskId=${encodeURIComponent(taskId)}&token=${safeToken}`
	)
	eventSource.addEventListener("progress", (e: MessageEvent) => {
		if (task.value) {
			task.value.progress = JSON.parse(e.data)
		}
	})
	eventSource.addEventListener("done", (e: MessageEvent) => {
		task.value = JSON.parse(e.data)
		scanning.value = false
		checkedPaths.value = []
		if (task.value?.status === "failed") {
			message.error(task.value.error || "扫描失败")
		} else if (task.value?.status === "canceled") {
			message.warning("扫描已取消")
		}
	})
	eventSource.addEventListener("eof", () => {
		stopStream()
		scanning.value = false
		// SSE 兜底：连接异常结束时补一次拉取，避免界面停在"扫描中"
		if (task.value?.id) {
			void refreshTask(task.value.id)
		}
	})
	eventSource.onerror = () => {
		stopStream()
		if (task.value?.id) void refreshTask(task.value.id)
	}
}

const refreshTask = async (taskId: string) => {
	try {
		const res = await getDiskScanResult(taskId)
		task.value = res.data
		scanning.value = res.data.status === "running"
	} catch (_error) {
		scanning.value = false
	}
}

const handleScan = async () => {
	const roots = scanRoots.value
		.split(/[,\n]/)
		.map(v => v.trim())
		.filter(Boolean)
	if (!roots.length) {
		message.warning("请填写扫描根目录")
		return
	}
	scanning.value = true
	checkedPaths.value = []
	try {
		const res = await startDiskScan({
			roots,
			minSize: Math.max(1, minSizeMB.value) * 1024 * 1024,
			topN: topN.value,
			crossDevice: crossDevice.value
		})
		task.value = res.data
		openStream(res.data.id)
	} catch (error: any) {
		scanning.value = false
		message.error(error.message || "启动扫描失败")
	}
}

const handleCancel = async () => {
	if (!task.value?.id) return
	try {
		await cancelDiskScan(task.value.id)
		message.info("已请求取消")
	} catch (error: any) {
		message.error(error.message || "取消失败")
	}
}

const files = computed(() => task.value?.result?.files || [])
const dirs = computed(() => task.value?.result?.dirs || [])

const selectedFiles = computed(() => files.value.filter(f => checkedPaths.value.includes(f.path)))
const selectedSize = computed(() => selectedFiles.value.reduce((sum, f) => sum + f.size, 0))

const openConfirm = (truncate: boolean) => {
	if (!checkedPaths.value.length) {
		message.warning("请先选择要处理的文件")
		return
	}
	confirmTruncate.value = truncate
	confirmChecked.value = false
	confirmVisible.value = true
}

const doClean = async () => {
	if (!task.value?.id) return
	cleaning.value = true
	try {
		const res = await cleanDiskPaths(task.value.id, [...checkedPaths.value], confirmTruncate.value)
		const items = res.data || []
		const okItems = items.filter(i => i.ok)
		const failed = items.filter(i => !i.ok)
		if (okItems.length) {
			const freed = okItems.reduce((sum, i) => sum + i.freed, 0)
			message.success(`成功处理 ${okItems.length} 个文件，释放约 ${computeSizeFromByte(freed)}`)
		}
		if (failed.length) {
			message.warning(`${failed.length} 个文件未处理：${failed[0].message || "未知原因"}`)
		}
		confirmVisible.value = false
		checkedPaths.value = []
		// 结果里已处理的条目要移除，避免用户对着不存在的文件反复操作
		if (task.value.result) {
			const done = new Set(okItems.filter(i => !confirmTruncate.value).map(i => i.path))
			task.value.result.files = task.value.result.files.filter(f => !done.has(f.path))
		}
		void fetchOverview()
	} catch (error: any) {
		message.error(error.message || "清理失败")
	} finally {
		cleaning.value = false
	}
}

const fileColumns: DataTableColumns<Disk.FileItem> = [
	{ type: "selection", disabled: (row: Disk.FileItem) => row.isContainer },
	{
		title: "路径",
		key: "path",
		minWidth: 320,
		render: (row: Disk.FileItem) => h("div", { class: "break-all text-xs" }, row.path)
	},
	{
		title: "大小",
		key: "size",
		width: 110,
		sorter: (a: Disk.FileItem, b: Disk.FileItem) => a.size - b.size,
		defaultSortOrder: "descend",
		render: (row: Disk.FileItem) => computeSizeFromByte(row.size)
	},
	{
		title: "分类",
		key: "category",
		width: 110,
		render: (row: Disk.FileItem) => {
			const meta = categoryMeta[row.category] || categoryMeta.other
			return h(NTag, { size: "small", type: meta.type }, { default: () => meta.label })
		}
	},
	{
		title: "建议",
		key: "advice",
		minWidth: 240,
		render: (row: Disk.FileItem) =>
			h("div", { class: "text-xs text-slate-500" }, (categoryMeta[row.category] || categoryMeta.other).advice)
	},
	{
		title: "修改时间",
		key: "modTime",
		width: 160,
		render: (row: Disk.FileItem) => (row.modTime ? new Date(row.modTime).toLocaleString() : "-")
	}
]

const dirColumns: DataTableColumns<Disk.DirItem> = [
	{ title: "目录", key: "path", minWidth: 320, render: (row: Disk.DirItem) => h("div", { class: "break-all text-xs" }, row.path) },
	{
		title: "直属文件占用",
		key: "size",
		width: 140,
		render: (row: Disk.DirItem) => computeSizeFromByte(row.size)
	},
	{ title: "文件数", key: "count", width: 100 }
]

// 选中某块盘 = 把它的挂载点设成扫描根目录。
// 这跟"默认不跨文件系统"正好配套：扫 /data 就只会扫到 /data 这块盘，
// 不会顺着目录树串到别的挂载点上去。
const selectedDisk = computed(() => scanRoots.value.trim())

const selectDisk = (path: string) => {
	scanRoots.value = path
	crossDevice.value = false // 选定单盘时跨文件系统没有意义，还会扫串
}

const scanDisk = (path: string) => {
	selectDisk(path)
	void handleScan()
}

// 折叠时头部要能一眼看出"有没有盘快满了"，否则折叠等于把主信息藏起来
const diskSummary = computed(() => {
	if (!disks.value.length) return "暂无数据"
	const worst = disks.value.reduce((a, b) => (a.usedPercent >= b.usedPercent ? a : b))
	return `${disks.value.length} 个挂载点 · 最高 ${worst.usedPercent.toFixed(1)}%（${worst.path}）`
})

const worstPercent = computed(() => {
	if (!disks.value.length) return 0
	return Math.max(...disks.value.map(d => d.usedPercent))
})

// 扫描没有"总量"可言（不预先数一遍文件就不知道分母），所以不显示百分比。
// 之前用 scannedFiles % 10000 造了个假进度，结果是进度条每 1 万个文件归零一次，
// 全盘扫描时来回跑好几轮，看着像卡死。真实的计数本身就是最好的进度反馈。
const hasLiveProgress = computed(() => task.value?.progressLive !== false)

onMounted(fetchOverview)
onBeforeUnmount(stopStream)
</script>

<template>
	<div class="page-container">
		<div class="page-header mb-6 flex items-center justify-between">
			<div class="page-title text-2xl font-bold text-gray-800 dark:text-gray-200">磁盘管理</div>
			<n-space>
				<n-button v-if="scanning" @click="handleCancel">取消扫描</n-button>
				<n-button type="primary" :loading="scanning" @click="handleScan">开始扫描</n-button>
			</n-space>
		</div>

		<n-collapse class="mb-4 rounded-lg border border-slate-200 px-4 py-1 dark:border-slate-700">
			<n-collapse-item name="disks">
				<template #header>
					<div class="flex items-center gap-2 py-2">
						<span class="text-sm font-medium">磁盘占用</span>
						<n-tag size="tiny" :type="worstPercent >= 90 ? 'error' : worstPercent >= 75 ? 'warning' : 'default'">
							{{ diskSummary }}
						</n-tag>
						<span class="text-xs text-slate-400">展开可选择只扫描某块盘</span>
					</div>
				</template>
				<div v-if="!disks.length" class="pb-3 text-sm text-slate-400">暂无数据</div>
				<div v-else class="grid grid-cols-1 gap-4 pb-3 md:grid-cols-2 xl:grid-cols-3">
					<div
						v-for="d in disks"
						:key="d.path + d.device"
						class="cursor-pointer rounded-lg border p-3 transition-colors"
						:class="
							selectedDisk === d.path
								? 'border-primary bg-primary/5'
								: 'border-slate-200 hover:border-slate-300 dark:border-slate-700'
						"
						@click="selectDisk(d.path)"
					>
						<div class="mb-1 flex items-center justify-between">
							<span class="truncate text-sm font-medium" :title="d.path">{{ d.path }}</span>
							<n-tag size="tiny" :type="d.usedPercent >= 90 ? 'error' : d.usedPercent >= 75 ? 'warning' : 'default'">
								{{ d.usedPercent.toFixed(1) }}%
							</n-tag>
						</div>
						<n-progress
							type="line"
							:percentage="Math.min(100, d.usedPercent)"
							:status="d.usedPercent >= 90 ? 'error' : d.usedPercent >= 75 ? 'warning' : 'success'"
							:show-indicator="false"
						/>
						<div class="mt-1 flex items-center justify-between">
							<div class="text-xs text-slate-500">
								已用 {{ computeSizeFromByte(d.used) }} / 共 {{ computeSizeFromByte(d.total) }}
								<span v-if="d.inodesUsedPercent > 0"> · inode {{ d.inodesUsedPercent.toFixed(1) }}%</span>
							</div>
							<n-button
								size="tiny"
								quaternary
								type="primary"
								:disabled="scanning"
								@click.stop="scanDisk(d.path)"
							>
								扫描此盘
							</n-button>
						</div>
					</div>
				</div>
			</n-collapse-item>
		</n-collapse>

		<n-card title="扫描设置" class="mb-4">
			<n-space align="center" :wrap="true">
				<span class="text-sm">根目录</span>
				<n-input v-model:value="scanRoots" class="w-[260px]" placeholder="/，多个用逗号分隔" />
				<span class="text-sm">最小体积</span>
				<n-input-number v-model:value="minSizeMB" class="w-[130px]" :min="1" :step="50">
					<template #suffix>MB</template>
				</n-input-number>
				<span class="text-sm">保留条数</span>
				<n-input-number v-model:value="topN" class="w-[110px]" :min="10" :max="1000" :step="50" />
				<n-checkbox v-model:checked="crossDevice">跨文件系统</n-checkbox>
			</n-space>
			<div class="mt-2 text-xs text-slate-400">
				默认不跨文件系统，避免扫进网络盘和容器 overlay；/proc、/sys、/dev、/run 始终跳过。
			</div>
		</n-card>

		<n-alert v-if="task?.degraded" type="warning" class="mb-4" :show-icon="true">
			{{ task.degradedReason }}
		</n-alert>

		<n-card v-if="scanning" class="mb-4">
			<n-space align="center" :size="12">
				<n-spin size="small" />
				<div v-if="hasLiveProgress" class="text-sm">
					正在扫描：已检查 {{ task?.progress?.scannedFiles || 0 }} 个文件 ·
					{{ computeSizeFromByte(task?.progress?.scannedBytes || 0) }}
					<span v-if="task?.progress?.errors" class="text-amber-500">
						· 跳过 {{ task.progress.errors }} 个无权限项
					</span>
				</div>
				<div v-else class="text-sm">
					正在通过 gpc 扫描，该模式下没有实时进度，请耐心等待…
				</div>
			</n-space>
			<div v-if="hasLiveProgress" class="mt-1 truncate text-xs text-slate-400">
				{{ task?.progress?.currentDir }}
			</div>
		</n-card>

		<n-card v-if="task?.status === 'success'" title="大文件" class="mb-4">
			<template #header-extra>
				<n-space align="center">
					<span class="text-xs text-slate-500">
						已选 {{ checkedPaths.length }} 项 · {{ computeSizeFromByte(selectedSize) }}
					</span>
					<n-button size="small" :disabled="!checkedPaths.length" @click="openConfirm(true)">清空（保留文件）</n-button>
					<n-button size="small" type="error" :disabled="!checkedPaths.length" @click="openConfirm(false)">
						删除
					</n-button>
				</n-space>
			</template>
			<n-data-table
				v-model:checked-row-keys="checkedPaths"
				:columns="fileColumns"
				:data="files"
				:row-key="(row: Disk.FileItem) => row.path"
				:max-height="420"
				:bordered="false"
			/>
			<div v-if="!files.length" class="py-6 text-center text-sm text-slate-400">
				没有超过 {{ minSizeMB }}MB 的文件
			</div>
		</n-card>

		<n-card v-if="task?.status === 'success' && dirs.length" title="目录占用（仅统计直属文件）">
			<n-data-table :columns="dirColumns" :data="dirs" :max-height="320" :bordered="false" />
		</n-card>

		<n-modal
			v-model:show="confirmVisible"
			preset="card"
			:title="confirmTruncate ? '确认清空这些文件？' : '确认删除这些文件？'"
			style="width: 640px"
		>
			<n-alert :type="confirmTruncate ? 'warning' : 'error'" :show-icon="true" class="mb-3">
				{{
					confirmTruncate
						? "清空会把文件内容截断为 0，但保留文件本身。正在写日志的进程不受影响。"
						: "删除不可恢复。如果文件正被进程占用，删除后空间不会立即释放，需要重启相关进程。"
				}}
			</n-alert>
			<div class="mb-3 text-sm">
				共 {{ selectedFiles.length }} 个文件，合计 {{ computeSizeFromByte(selectedSize) }}：
			</div>
			<div class="mb-3 max-h-[220px] overflow-auto rounded bg-slate-50 p-2 dark:bg-slate-800">
				<div v-for="f in selectedFiles" :key="f.path" class="break-all py-0.5 text-xs">
					{{ f.path }} · {{ computeSizeFromByte(f.size) }}
				</div>
			</div>
			<n-checkbox v-model:checked="confirmChecked">
				我已确认上述文件可以{{ confirmTruncate ? "清空" : "删除" }}
			</n-checkbox>
			<template #footer>
				<n-space justify="end">
					<n-button @click="confirmVisible = false">取消</n-button>
					<n-button
						:type="confirmTruncate ? 'warning' : 'error'"
						:disabled="!confirmChecked"
						:loading="cleaning"
						@click="doClean"
					>
						确认执行
					</n-button>
				</n-space>
			</template>
		</n-modal>
	</div>
</template>
