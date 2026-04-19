<template>
	<div class="inline-block">
		<n-popover
			:show="showPopover"
			trigger="click"
			placement="bottom-end"
			:width="popoverWidth"
			:show-arrow="false"
			@update:show="handlePopoverUpdate"
		>
			<template #trigger>
				<n-button
					:size="size"
					:type="buttonType"
					:text="buttonText"
					:secondary="buttonSecondary"
					:class="{ '!px-0': buttonText }"
				>
					<template #icon>
						<Icon icon="lucide:settings-2" :width="16" :height="16" />
					</template>
					{{ buttonLabel || "列设置" }}
				</n-button>
			</template>

			<div class="flex max-h-[400px] min-w-[220px] flex-col">
				<div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
					<span class="text-sm font-medium">自定义列</span>
					<n-button text size="small" @click="handleReset">重置</n-button>
				</div>

				<div class="flex-1 overflow-y-auto py-1">
					<div class="flex flex-col">
						<div
							v-for="col in filteredColumns"
							:key="col.key"
							class="flex items-center justify-between px-4 py-2 transition-colors hover:bg-gray-50 dark:hover:bg-gray-800/50"
							:class="{ 'cursor-not-allowed opacity-60': isFixedColumn(col.key) }"
						>
							<n-checkbox
								:checked="col.visible"
								:disabled="isFixedColumn(col.key)"
								@update:checked="() => handleVisibilityChange(col)"
							>
								<span class="text-sm">{{ col.title }}</span>
							</n-checkbox>
							<div v-if="sortable && !isFixedColumn(col.key)" class="cursor-move text-gray-400">
								<Icon icon="lucide:grip-vertical" :width="14" :height="14" />
							</div>
						</div>
					</div>
				</div>

				<div v-if="showSearch" class="border-t border-gray-200 px-3 py-2 dark:border-gray-700">
					<n-input
						:value="searchKeyword"
						size="small"
						placeholder="搜索列名"
						clearable
						@update:value="value => (searchKeyword = value)"
					>
						<template #prefix>
							<Icon icon="lucide:search" :width="14" :height="14" class="text-gray-400" />
						</template>
					</n-input>
				</div>
			</div>
		</n-popover>
	</div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue"
import { NButton, NPopover, NCheckbox, NInput, useMessage } from "naive-ui"
import { Icon } from "@iconify/vue"

/**
 * 列配置项
 */
export interface ColumnConfig {
	/** 列唯一标识 */
	key: string
	/** 列显示标题 */
	title: string
	/** 是否可见 */
	visible: boolean
	/** 是否固定列（不可隐藏/排序） */
	fixed?: boolean
	/** 列宽度 */
	width?: number
	/** 原始列配置（用于渲染） */
	original?: any
}

/**
 * 组件 Props
 */
interface Props {
	/** 所有可用列的配置 */
	columns: ColumnConfig[]
	/** 存储 key（用于 localStorage 持久化） */
	storageKey?: string
	/** 按钮大小 */
	size?: "small" | "medium" | "large"
	/** 按钮类型 */
	buttonType?: "default" | "primary" | "success" | "error" | "warning" | "info"
	/** 是否为文本按钮 */
	buttonText?: boolean
	/** 按钮次要样式 */
	buttonSecondary?: boolean
	/** 按钮自定义文本 */
	buttonLabel?: string
	/** 下拉面板宽度 */
	popoverWidth?: number
	/** 是否支持拖拽排序（需要额外实现拖拽逻辑） */
	sortable?: boolean
	/** 是否显示搜索框 */
	showSearch?: boolean
	/** 是否启用本地持久化 */
	persistent?: boolean
	/** 列变化回调 */
	onChange?: (columns: ColumnConfig[]) => void
}

/**
 * 组件 Emits
 */
interface Emits {
	/** 列配置变化时触发 */
	(e: "update:columns", columns: ColumnConfig[]): void
	/** 列可见性变化时触发 */
	(e: "visibilityChange", key: string, visible: boolean): void
	/** 列顺序变化时触发 */
	(e: "orderChange", columns: ColumnConfig[]): void
}

const props = withDefaults(defineProps<Props>(), {
	size: "small",
	buttonType: "default",
	buttonText: false,
	buttonSecondary: false,
	popoverWidth: 280,
	sortable: false,
	showSearch: true,
	persistent: true
})

const emit = defineEmits<Emits>()
const message = useMessage()

// 响应式数据
const showPopover = ref(false)
const searchKeyword = ref("")
const localColumns = ref<ColumnConfig[]>([...props.columns])

// 固定列的 key 列表（不可隐藏/排序）
const fixedColumnKeys = computed(() => localColumns.value.filter(col => col.fixed).map(col => col.key))

// 过滤后的列（支持搜索）
const filteredColumns = computed(() => {
	if (!props.showSearch || !searchKeyword.value) {
		return localColumns.value
	}
	const keyword = searchKeyword.value.toLowerCase()
	return localColumns.value.filter(col => col.title.toLowerCase().includes(keyword))
})

/**
 * 判断是否为固定列
 */
const isFixedColumn = (key: string): boolean => {
	return fixedColumnKeys.value.includes(key)
}

/**
 * 处理列可见性变化
 */
const handleVisibilityChange = (column: ColumnConfig) => {
	column.visible = !column.visible
	emit("visibilityChange", column.key, column.visible)
	emit("update:columns", localColumns.value)
	if (props.onChange) {
		props.onChange(localColumns.value)
	}
	saveToStorage()
}

/**
 * 重置为默认配置
 */
const handleReset = () => {
	const defaultColumns = props.columns.map(col => ({
		...col,
		visible: col.visible !== undefined ? col.visible : true
	}))
	localColumns.value = defaultColumns
	emit("update:columns", localColumns.value)
	if (props.onChange) {
		props.onChange(localColumns.value)
	}
	message?.success("重置成功")
	saveToStorage()
}

/**
 * 打开面板时的处理
 */
const handleOpen = () => {
	localColumns.value = [...props.columns]
}

const handlePopoverUpdate = (value: boolean) => {
	if (value) {
		handleOpen()
	}
	showPopover.value = value
}

/**
 * 保存配置到 localStorage
 */
const saveToStorage = () => {
	if (!props.persistent || !props.storageKey) return

	const configToSave = localColumns.value.map(col => ({
		key: col.key,
		visible: col.visible
	}))
	localStorage.setItem(`table_column_config_${props.storageKey}`, JSON.stringify(configToSave))
}

/**
 * 从 localStorage 加载配置
 */
const loadFromStorage = () => {
	if (!props.persistent || !props.storageKey) return

	const saved = localStorage.getItem(`table_column_config_${props.storageKey}`)
	if (!saved) return

	try {
		const savedConfig = JSON.parse(saved) as Array<{
			key: string
			visible: boolean
		}>

		// 应用保存的可见性
		const newColumns = localColumns.value.map(col => {
			const savedCol = savedConfig.find(sc => sc.key === col.key)
			if (savedCol && !col.fixed) {
				return { ...col, visible: savedCol.visible }
			}
			return col
		})

		localColumns.value = newColumns
	} catch (e) {
		console.error("Failed to load column config from storage", e)
	}
}

// 监听外部 columns 变化（当持久化禁用时自动同步）
watch(
	() => props.columns,
	newCols => {
		if (!props.persistent) {
			localColumns.value = [...newCols]
		}
	},
	{ deep: true }
)

// 组件挂载时加载保存的配置
onMounted(() => {
	loadFromStorage()
})
</script>
