<template>
  <div class="mb-6 flex items-center bg-base-100 pr-6 shadow">
    <div class="flex flex-1">
      <div
        class="cursor-pointer rounded text-center"
        @click="changeKey(item.name)"
        style="border-width: 2px; height: 42px; line-height: 38px; width: 100px"
        v-for="(item, index) in menuOptions"
        :class="{ 'border-primary': activeKey == item.name, 'border-transparent': activeKey != item.name }"
        :key="index"
      >
        {{ item.meta?.title || item.title }}
      </div>
    </div>
    <div class="flex items-center">
      <!-- <n-button text type="primary" @click="openAlert('panel')">重启面板</n-button> -->
      <n-divider vertical />
      <!-- <n-button text type="primary" @click="openAlert('system')">重启服务器</n-button> -->
    </div>
  </div>
  <RouterView></RouterView>
  <RebootAlert
    ref="RebootAlertModel"
    @confirm="alertConfirm"
  />
</template>
<script setup lang="ts">
import { computed, ref, watch } from "vue"
import type { RouteRecordRaw } from "vue-router"
import { useRoute, useRouter } from "vue-router"
import { systemRestartPanelAPI, systemRestartSystemAPI } from "@/api/system/restart"
import RebootAlert from "@/components/RebootAlert.vue"
type MenuOption = {
	name: string
	title: string
	meta?: {
		title?: string
		input?: string
	}
}
const alertOptions = {
	panel: {
		title: "重启面板",
		input: "立即重启"
	},
	system: {
		title: "重启服务器",
		input: "立即重启"
	}
}
const RebootAlertModel = ref<InstanceType<typeof RebootAlert> | null>(null)
let alertApi: Function = systemRestartPanelAPI
const openAlert = (type: "panel" | "system") => {
	if (type == "panel") {
		alertApi = systemRestartPanelAPI
	} else {
		alertApi = systemRestartSystemAPI
	}
	RebootAlertModel.value?.open(alertOptions[type])
}
const alertConfirm = (loading: any) => {
	if (alertApi) {
		loading.value = true
		alertApi()
			.then((res: any) => {
				if (res.code == 0) {
					RebootAlertModel.value?.close()
				}
			})
			.finally(() => {
				loading.value = false
			})
	}
}
const route = useRoute()
const router = useRouter()
const changeKey = (key?: string) => {
	if (!key) return
	if (activeKey.value == key) return
	activeKey.value = key
	router.replace({ name: key })
}
const activeKey = ref(typeof route.name === "string" ? route.name : "")
watch(
	() => route.name,
	name => {
		activeKey.value = typeof name === "string" ? name : ""
	}
)
const menuOptions = computed<MenuOption[]>(() => {
	const toolboxRoute = router.options.routes.find(item => item.path == "/toolbox")
	return ((toolboxRoute?.children || []) as RouteRecordRaw[]).map(item => ({
		name: typeof item.name === "string" ? item.name : "",
		title: typeof item.meta?.title === "string" ? item.meta.title : String(item.name || ""),
		meta: {
			title: typeof item.meta?.title === "string" ? item.meta.title : String(item.name || "")
		}
	}))
})
</script>
