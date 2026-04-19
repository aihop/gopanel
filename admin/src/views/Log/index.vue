<template>
  <div class="log-container h-full w-full">
    <n-card
      :bordered="false"
      class="h-full rounded-2xl"
    >
      <RouterButton
        :buttons="tabList"
        @update:active="handleTabChange"
      />

      <div class="mt-4 flex-1 min-h-0">
        <keep-alive>
          <component :is="currentComponent" />
        </keep-alive>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, onMounted } from "vue"
import RouterButton from "@/components/RouterButton.vue"
import LoginLog from "./components/LoginLog.vue"
import OperationLog from "./components/OperationLog.vue"
import SystemLog from "./components/SystemLog.vue"

const activeTab = ref("操作日志")

const tabList = [
	{ label: "操作日志" },
	{ label: "登录日志" },
	{ label: "系统日志" }
]

const currentComponent = computed(() => {
	switch (activeTab.value) {
		case "操作日志":
			return OperationLog
		case "登录日志":
			return LoginLog
		case "系统日志":
			return SystemLog
		default:
			return OperationLog
	}
})

const handleTabChange = (val: string) => {
	activeTab.value = val
}
</script>

<style scoped lang="scss">
.log-container {
	:deep(.n-card > .n-card__content) {
		padding: 20px;
		height: 100%;
		min-height: 0;
		display: flex;
		flex-direction: column;
	}
}
</style>
