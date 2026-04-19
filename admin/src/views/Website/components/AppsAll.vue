<template>
  <div class="flex flex-wrap gap-4">
    <n-card
      v-for="item in apps"
      :key="item.id"
      class="app-card"
    >
      <template #header>
        <img
          v-if="item.icon"
          :src="item.icon"
          alt="icon"
          class="mr-2 h-8 w-8 align-middle"
        />
        <span>{{ item.name }}</span>
        <n-tag
          v-if="item.installed"
          type="success"
          size="small"
          class="ml-2"
        >已安装</n-tag>
      </template>

      <!-- <template #header-extra>
				<n-button v-if="item.installed" secondary type="info" size="small" disabled>安装</n-button>
				<n-button v-else secondary type="info" size="small" @click="() => handleInstallApp(item)">
					安装
				</n-button>
			</template> -->

      <div>类型：{{ item.type }}</div>
      <div v-if="item.resource">来源：{{ item.resource }}</div>
      <div v-if="item.description">描述：{{ item.description }}</div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from "vue"
import { appsSearchAPI } from "@/api/modules/apps"
import type { AppsSearchParams } from "@/api/modules/apps"
import { useMessage } from "naive-ui"
import { useRouter } from "vue-router"

const props = defineProps<{
	searchName: string
	page: number
	pageSize: number
}>()
const emits = defineEmits(["update:total"])

const message = useMessage()
const router = useRouter()
const apps = ref<any[]>([])
const loading = ref(false)

const fetchData = async () => {
	loading.value = true
	try {
		const params: AppsSearchParams = {
			page: props.page,
			pageSize: props.pageSize,
			name: props.searchName.trim() || undefined
		}
		const res = await appsSearchAPI(params)
		const data = res.data as any
		if (res.code === 0 && data && Array.isArray(data.items)) {
			apps.value = data.items
			emits("update:total", data.total)
		} else {
			message.error(res.msg || "获取应用列表失败")
		}
	} catch (e) {
	} finally {
		loading.value = false
	}
}

watch([() => props.searchName, () => props.page, () => props.pageSize], fetchData, { immediate: true })

 
</script>

<style scoped>
.app-card {
	width: 350px;
	margin-bottom: 16px;
}
</style>
