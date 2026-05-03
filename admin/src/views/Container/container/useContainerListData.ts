import { reactive, ref } from "vue"
import { containerListAPI, containerStatsAPI, loadInstanceStatus } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"

interface UseContainerListDataOptions {
  filters?: string
}

export const useContainerListData = (options: UseContainerListDataOptions) => {
  const loading = ref(false)
  const data = ref<any[]>([])
  const selects = ref<any[]>([])
  const searchName = ref()
  const searchState = ref("all")
  const includeAppStore = ref<boolean>()
  const dockerStatus = ref("Running")

  const paginationConfig = reactive({
    cacheSizeKey: "container-page-size",
    currentPage: 1,
    limit: 10,
    total: 0,
    orderBy: "created_at",
    order: "null"
  })

  const loadStats = async () => {
    const res = await containerStatsAPI()
    const stats = res.data || []
    if (!stats.length) {
      return
    }
    for (const container of data.value) {
      for (const item of stats) {
        if (container.containerID === item.containerID) {
          container.hasLoad = true
          container.cpuTotalUsage = item.cpuTotalUsage
          container.systemUsage = item.systemUsage
          container.cpuPercent = item.cpuPercent
          container.percpuUsage = item.percpuUsage
          container.memoryCache = item.memoryCache
          container.memoryUsage = item.memoryUsage
          container.memoryLimit = item.memoryLimit
          container.memoryPercent = item.memoryPercent
          break
        }
      }
    }
  }

  const search = async (column?: any) => {
    localStorage.setItem("includeAppStore", includeAppStore.value ? "true" : "false")
    const filterItem = options.filters || ""
    paginationConfig.orderBy = column?.order ? column.prop : paginationConfig.orderBy
    paginationConfig.order = column?.order ? column.order : paginationConfig.order
    const params = {
      name: searchName.value,
      state: searchState.value || "all",
      page: paginationConfig.currentPage,
      limit: paginationConfig.limit,
      filters: filterItem,
      orderBy: "created_at",
      order: paginationConfig.order,
      excludeAppStore: !includeAppStore.value
    }
    loading.value = true
    loadStats()
    await containerListAPI(params)
      .then(res => {
        loading.value = false
        data.value = Array.isArray(res.data.items) ? res.data.items : []
        paginationConfig.total = res.data.total
        selects.value = []
      })
      .catch(() => {
        loading.value = false
      })
  }

  const refresh = async () => {
    const filterItem = options.filters || ""
    const params = {
      name: searchName.value,
      state: searchState.value || "all",
      page: paginationConfig.currentPage,
      limit: paginationConfig.limit,
      filters: filterItem,
      orderBy: paginationConfig.orderBy,
      order: paginationConfig.order
    }
    loadStats()
    const res = await containerListAPI(params)
    const containers = res.data.items || []
    for (const container of containers) {
      for (const current of data.value) {
        current.hasLoad = true
        if (container.containerID === current.containerID) {
          const containerData = container as Record<string, any>
          for (const key in containerData) {
            if (key !== "cpuPercent" && key !== "memoryPercent") {
              ;(current as Record<string, any>)[key] = containerData[key]
            }
          }
        }
      }
    }
  }

  const handlePageSizeChange = (size: number) => {
    paginationConfig.limit = size
    paginationConfig.currentPage = 1
    if (paginationConfig.cacheSizeKey) {
      localStorage.setItem(paginationConfig.cacheSizeKey, String(size))
    }
    search()
  }

  const loadStatus = async () => {
    loading.value = true
    await loadInstanceStatus()
      .then(res => {
        loading.value = false
        dockerStatus.value = res.data
        if (dockerStatus.value === "Running") {
          search()
        }
      })
      .catch(() => {
        dockerStatus.value = "Failed"
        loading.value = false
      })
  }

  const checkStatus = (operation: string, row: Container.ContainerInfo | null) => {
    let opList: Container.ContainerInfo[] = []
    if (row) {
      opList = [row]
    } else if (selects.value && selects.value.length > 0) {
      const selectedIds = new Set(selects.value.map((item: any) => (typeof item === "object" ? item.containerID : item)))
      opList = data.value.filter((item: Container.ContainerInfo) => selectedIds.has(item.containerID))
    }

    if (opList.length < 1) {
      return true
    }
    switch (operation) {
      case "start":
        return opList.some(item => item && item.state === "running")
      case "stop":
        return opList.some(item => item.state === "stopped" || item.state === "exited")
      case "pause":
        return opList.some(item => item.state === "paused" || item.state === "exited")
      case "unpause":
        return opList.some(item => item.state !== "paused")
      case "restart":
      case "kill":
      case "remove":
      case "commit":
        return false
      default:
        return true
    }
  }

  const initIncludeAppStore = () => {
    const includeItem = localStorage.getItem("includeAppStore")
    includeAppStore.value = !includeItem || includeItem === "true"
  }

  return {
    loading,
    data,
    selects,
    searchName,
    searchState,
    includeAppStore,
    dockerStatus,
    paginationConfig,
    search,
    refresh,
    handlePageSizeChange,
    loadStatus,
    checkStatus,
    initIncludeAppStore
  }
}
