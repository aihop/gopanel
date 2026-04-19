import type { Ref } from "vue"
import { ref, computed } from "vue"
import { isArray, isSucc } from "@/utils/is"
import { MsgSuccess } from "@/utils/message"

export function useTable(params: any) {
	const list: Ref<any[]> = ref([])
	const total = ref(0)
	const curPage = ref(1)
	const selected: any = ref([])
	const pageSize = ref(params.limit || 20)
	const pageSizeOptions = ref([10, 20, 50, 100])
	const loading = ref(false)
	const filters: Ref<any[]> = ref([])
	const joinsData: Ref<any[]> = ref([])
	const pages = computed(() => {
		if (total.value > pageSize.value) {
			return Math.ceil(total.value / pageSize.value)
		}
		return 1
	})
	function getParams() {
		const newParams: any = {}
		for (const key in params.params) {
			if (key === "conditions") {
				const conditions = params.params[key]
				newParams[key] = conditions
			} else if (key === "wheres") {
				const wheres = params.params[key]
				if (!isArray(wheres)) {
					return
				}
				const newWheres: any = []
				for (let i = 0; i < wheres.length; i++) {
					const where = { ...wheres[i] }
					if (where.val !== "" && where.val !== undefined) {
						if (isArray(where.val)) {
							where.val = where.val.join(",")
						} else {
							where.val = where.val.toString()
						}
						newWheres.push(where)
					}
				}
				if (newWheres.length > 0) {
					newParams[key] = newWheres
				}
			} else if (key === "joins") {
				const joins = params.params[key]
				const newJoins: any[] = []
				joins.forEach((item: any) => {
					const obj: any = {
						table: item.table,
						wheres: []
					}
					item.wheres.forEach((where: any) => {
						if (isArray(where.val)) {
							if (where.val.length > 0) {
								where.val = where.val.join(",")
								obj.wheres.push(where)
							}
						} else if (where.val) {
							where.val = where.val.toString()
							obj.wheres.push(where)
						}
					})
					if (obj.wheres.length) {
						newJoins.push(obj)
					}
				})
				newParams[key] = newJoins
			} else if (params.params[key] || params.params[key] === 0) {
				newParams[key] = params.params[key]
			}
		}
		if (filters.value.length) {
			filters.value = filters.value.filter((where: any) => {
				if (where.val) {
					if (isArray(where.val)) {
						where.val = where.val.join(",")
					} else {
						where.val = where.val.toString()
					}
					return true
				}
			})
			if (!newParams.wheres) {
				newParams.wheres = filters.value
			} else {
				newParams.wheres = [...newParams.wheres, ...filters.value]
			}
		}
		if (joinsData.value.length) {
			if (newParams.joins) {
				newParams.joins = [...newParams.joins, ...joinsData.value]
			} else {
				newParams.joins = joinsData.value
			}
		}
		return newParams
	}
	const getList = async () => {
		loading.value = true
		const query = getParams()
		await params
			.listAPI({
				page: curPage.value,
				limit: pageSize.value,
				...query
			})
			.then((res: any) => {
				if (isSucc(res.code)) {
					list.value = res.data || []
				}
			})
			.catch(() => {
				loading.value = false
			})
		loading.value = false
	}
	const getCount = async () => {
		const query = getParams()
		await params.countAPI(query).then((res: any) => {
			if (isSucc(res.code)) {
				total.value = res.data || 0
			}
		})
	}
	const onPageChange = () => {
		getList()
	}
	const onPageSizeChange = () => {
		curPage.value = 1
		getList()
	}
	const getData = async () => {
		await getList()
		if (curPage.value === 1 && list.value.length < pageSize.value) {
			total.value = list.value.length
		} else if (!total.value) {
			getCount()
		}
	}
	const onSearch = () => {
		total.value = 0
		curPage.value = 1
		getData()
	}
	const handleDel = (id: number) => {
		if (!params.delAPI) return
		if (!id) return
		params.delAPI({ id }).then((res: any) => {
			if (isSucc(res.code)) {
				MsgSuccess(res.msg)
				onSearch()
			}
		})
	}

	// 筛选
	const filterSubmit = (data: any[], joins: any[]) => {
		filters.value = data || []
		joinsData.value = joins || []
		onSearch()
	}
	const filterReset = () => {
		filters.value = []
		joinsData.value = []
	}
	const setJoins = (table: string, field: string, val: string | number, rule: string = "eq") => {
		if (val) {
			if (!params.params.joins) {
				params.params.joins = [
					{
						table,
						wheres: [{ field, rule, val }]
					}
				]
			} else {
				const join = params.params.joins.find((join: any) => join.table == table)
				if (join) {
					if (!join.wheres) join.wheres = []
					const where = join.wheres.find((where: any) => where.field == field)
					if (where) {
						where.rule = rule
						where.val = val
					} else {
						join.wheres.push({ field, rule, val })
					}
				} else {
					params.params.joins.push({ table, wheres: [{ field, rule, val }] })
				}
			}
		} else if (params.params.joins && params.params.joins.length) {
			const join = params.params.joins.find((join: any) => join.table == table)
			if (join && join.wheres && join.wheres.length) {
				const index = join.wheres.findIndex((where: any) => where.field == field)
				if (index > -1) {
					join.wheres.splice(index, 1)
				}
				if (!join.wheres.length) {
					const index = params.params.joins.findIndex((join: any) => join.table == table)
					if (index > -1) {
						params.params.joins.splice(index, 1)
					}
					if (!params.params.joins.length) {
						delete params.params.joins
					}
				}
			}
		}
	}
	const setWheres = (field: string, val: string | number, rule: string = "eq") => {
		const index = params.params.wheres.findIndex((item: any) => item.field === field)
		if (val) {
			if (index > -1) {
				params.params.wheres[index].val = val
				params.params.wheres[index].rule = rule
			} else {
				params.params.wheres.push({
					field,
					rule,
					val
				})
			}
		} else {
			if (index > -1) {
				params.params.wheres.splice(index, 1)
			}
		}
	}
	const filterChange = (field: string, [value, like]: any) => {
		const arr = field.split(".")
		if (arr.length > 1) {
			setJoins(arr[0], arr[1], value, like ? "like" : "eq")
		} else {
			setWheres(field, value, like ? "like" : "eq")
		}
		onSearch()
	}
	return {
		list,
		total,
		curPage,
		pageSize,
		pageSizeOptions,
		getList,
		loading,
		onPageChange,
		onPageSizeChange,
		onSearch,
		handleDel,
		getData,
		selected,
		filterSubmit,
		filterReset,
		filters,
		filterChange,
		setWheres,
		pages
	}
}
