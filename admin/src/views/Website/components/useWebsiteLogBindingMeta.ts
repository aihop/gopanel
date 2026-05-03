import type { ComputedRef, Ref } from "vue"
import { computed, ref } from "vue"
import type { Website } from "@/api/interface/website"
import type { App } from "@/api/interface/apps"
import type { Pipeline } from "@/api/interface/pipeline"
import { ListAppInstalled } from "@/api/modules/apps"
import { listAllPipelines } from "@/utils/pipeline"
import { resolveWebsiteBindingMeta } from "@/utils/websiteRuntime"

export const useWebsiteLogBindingMeta = (website: Ref<Website.WebsiteDTO | null>): {
  appInstallMap: Ref<Record<number, App.AppInstalledInfo>>
  pipelineMap: Ref<Record<number, Pipeline.ResPipeline>>
  bindingRuntimeText: ComputedRef<string>
  loadBindingMeta: () => Promise<void>
} => {
  const appInstallMap = ref<Record<number, App.AppInstalledInfo>>({})
  const pipelineMap = ref<Record<number, Pipeline.ResPipeline>>({})

  const bindingRuntimeText = computed(() => {
    if (!website.value) return ""
    return (
      resolveWebsiteBindingMeta(
        website.value,
        {
          appInstallMap: appInstallMap.value,
          pipelineMap: pipelineMap.value
        },
        {
          sourcePrefix: "绑定目标：",
          includeSourceInDetail: true,
          kindFallback: "Runtime",
          userFallback: "镜像默认",
          runUserPrefix: "运行用户："
        }
      )?.detail || ""
    )
  })

  const loadBindingMeta = async () => {
    if (!website.value) return
    try {
      if (website.value.appInstallId) {
        const res = await ListAppInstalled()
        const list: App.AppInstalledInfo[] = Array.isArray(res.data) ? res.data : []
        const nextMap: Record<number, App.AppInstalledInfo> = {}
        for (const item of list) nextMap[item.id] = item
        appInstallMap.value = nextMap
      }
      if (website.value.pipelineId) {
        const list = await listAllPipelines()
        const nextMap: Record<number, Pipeline.ResPipeline> = {}
        for (const item of list) nextMap[item.id] = item
        pipelineMap.value = nextMap
      }
    } catch {
      appInstallMap.value = {}
      pipelineMap.value = {}
    }
  }

  return {
    appInstallMap,
    pipelineMap,
    bindingRuntimeText,
    loadBindingMeta
  }
}
