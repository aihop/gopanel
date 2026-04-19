<template>
  <div class="complex-table">
    <div
      v-if="$slots.header || header"
      class="complex-table__header"
    >
      <slot name="header">{{ header }}</slot>
    </div>
    <div
      v-if="$slots.toolbar"
      style="margin-bottom: 10px"
    >
      <slot name="toolbar"></slot>
    </div>

    <div class="complex-table__body">
      <n-data-table
        v-bind="attrs"
        ref="tableRef"
        v-slots="slots"
        :columns="tableColumns"
        :row-key="getRowKey"
        :checked-row-keys="checkedRowKeys"
        @update:checked-row-keys="handleSelectionChange"
        @update:sort-state="handleSortChange"
      >
        <template #empty>
          <slot name="empty" />
        </template>
      </n-data-table>
    </div>

    <div
      v-if="props.paginationConfig"
      class="complex-table__pagination"
    >
      <slot name="pagination">
        <n-pagination
          v-model:page="paginationConfig.currentPage"
          v-model:page-size="paginationConfig.pageSize"
          :page-sizes="[5, 10, 20, 50, 100]"
          :show-size-picker="true"
          :show-quick-jumper="!mobile"
          :page-count="Math.ceil(paginationConfig.total / paginationConfig.pageSize)"
          @update:page="currentChange"
          @update:page-size="sizeChange"
        />
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import GlobalStore from "@/store/modules/global"
import { NButton, NSpace } from "naive-ui"
import { computed, h, onMounted, ref, toRefs, useSlots, useAttrs, VNode } from "vue"

type RowKey = string | number

defineOptions({ name: "ComplexTable" })

const props = defineProps({
	header: String,
	rowKey: { type: String, default: "id" },
	paginationConfig: {
		type: Object,
		required: false,
		default: () => {}
	}
})

const emit = defineEmits(["search", "update:selects", "update:paginationConfig"])

const slots = useSlots()
const attrs = useAttrs()

function parseChildren(nodes: VNode[] = [], columns: any[] = []) {
	nodes.forEach(node => {
		if (!node) return
		const typeName = (node.type as any)?.name
		if (typeName === "ElTableColumn") {
			const props = node.props || {}
			if (props.type === "selection") {
				columns.push({
					type: "selection",
					width: props.width || props["min-width"] || 48,
					fixed: props.fixed ? "left" : undefined
				})
				return
			}
			columns.push({
				title: props.label,
				key: props.prop,
				width: props.width,
				minWidth: props.minWidth || props["min-width"],
				fixed: props.fixed ? "left" : undefined,
				align: props.align,
				ellipsis: props.showOverflowTooltip ? { tooltip: true } : undefined,
				sorter: props.sortable ? "default" : undefined,
				render: (row: any, index: number) => {
					if (node.children && typeof node.children === "object" && "default" in node.children) {
						return (node.children as any).default?.({ row, $index: index })
					}
					if (typeof props.formatter === "function") {
						return props.formatter(row, null, row[props.prop], index)
					}
					return row[props.prop]
				}
			})
			return
		}
		if ((node.props as any)?.buttons && (node.props as any)?.label) {
			const props = node.props as any
			columns.push({
				title: props.label,
				key: `operations-${columns.length}`,
				width: props.width,
				fixed: props.fix ? "right" : undefined,
				render: (row: any) =>
					h(
						NSpace,
						{ wrap: false },
						{
							default: () =>
								(props.buttons || []).map((button: any) =>
									h(
										NButton,
										{
											size: "small",
											disabled:
												typeof button.disabled === "function"
													? button.disabled(row)
													: !!button.disabled,
											onClick: () => button.click?.(row)
										},
										{ default: () => button.label }
									)
								)
						}
					)
			})
			return
		}
		if (node.children && typeof node.children === "object" && "default" in node.children) {
			parseChildren((node.children as any).default?.() || [], columns)
		}
	})
	return columns
}

const tableColumns = computed(() => {
	const externalColumns = attrs.columns
	if (Array.isArray(externalColumns)) {
		return externalColumns
	}
	return parseChildren((slots.default?.() as VNode[]) || [])
})

const getRowKey = (row: Record<string, any>) => row[props.rowKey]
const { paginationConfig } = toRefs(props)

const globalStore = GlobalStore()

const mobile = computed(() => {
	return globalStore.isMobile()
})

const checkedRowKeys = ref<RowKey[]>([])
function handleSelectionChange(keys: RowKey[]) {
	checkedRowKeys.value = keys
	emit("update:selects", keys)
}

const tableRef = ref()

function currentChange(page: number) {
	emit("update:paginationConfig", { ...paginationConfig.value, currentPage: page })
}

function sizeChange(size: number) {
	emit("update:paginationConfig", { ...paginationConfig.value, pageSize: size, currentPage: 1 })
}

// 排序
function handleSortChange(sortState: any) {
	// sortState = { columnKey, order }
	// 你可以把它加到 paginationConfig 或者直接 emit 出去
	emit("search")
}

function sort(prop: string, order: string) {
	return
}

function clearSelects() {
	checkedRowKeys.value = []
	emit("update:selects", [])
}

function clearSort() {
	return
}

defineExpose({
	clearSelects,
	sort,
	clearSort
})

onMounted(() => {
	if (props.paginationConfig?.cacheSizeKey) {
		let itemSize = Number(localStorage.getItem(props.paginationConfig.cacheSizeKey))
		if (itemSize) {
			props.paginationConfig.pageSize = itemSize
		}
	}
})
</script>

<style lang="scss">
.complex-table {
	.complex-table__header {
		display: flex;

		line-height: 60px;
		font-size: 18px;
	}

	.complex-table__toolbar {
		.fu-search-bar {
			width: auto;
		}
	}
	.complex-table__pagination {
		margin-top: 20px;
	}
}
</style>
