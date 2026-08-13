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
import { ref, computed } from "vue"
import { useI18n } from "vue-i18n"
import RouterButton from "@/components/RouterButton.vue"
import LoginLog from "./components/LoginLog.vue"
import OperationLog from "./components/OperationLog.vue"
import SSHLoginLog from "./components/SSHLoginLog.vue"
import SystemLog from "./components/SystemLog.vue"
import SecurityRisk from "./components/SecurityRisk.vue"

const { t } = useI18n()
const securityTab = computed(() => t("securityMonitoring.title"))
const activeTab = ref("操作日志")

const tabList = computed(() => [
	{ label: "操作日志" },
	{ label: "登录日志" },
	{ label: "SSH 登录" },
	{ label: "系统日志" },
	{ label: securityTab.value }
])

const currentComponent = computed(() => {
	switch (activeTab.value) {
		case "操作日志":
			return OperationLog
		case "登录日志":
			return LoginLog
		case "SSH 登录":
			return SSHLoginLog
		case "系统日志":
			return SystemLog
		case securityTab.value:
			return SecurityRisk
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
