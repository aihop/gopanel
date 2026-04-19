<template>
  <n-tabs
    type="line"
    animated
    class="mt-2"
  >
    <template #suffix>
      <n-space
        v-if="!isSubAdmin"
        size="small"
        align="center"
      >
        <n-button
          ghost
          class="!rounded-[16px]"
          :loading="restartingPanel"
          @click="handleRestart('panel')"
        >
          重启面板
        </n-button>
        <n-button
          ghost
          class="!rounded-[16px]"
          :loading="restartingServer"
          @click="handleRestart('server')"
        >
          重启服务器
        </n-button>
      </n-space>
    </template>
    <n-tab-pane
      name="system"
      tab="面板设置"
    >
      <conf />
    </n-tab-pane>
    <n-tab-pane
      v-if="!isSubAdmin"
      name="subadmin"
      tab="管理员设置"
    >
      <sub-admin />
    </n-tab-pane>
    <n-tab-pane
      name="cloud"
      tab="云账号授权"
    >
      <cloud-account />
    </n-tab-pane>
    <n-tab-pane
      name="update"
      tab="版本更新"
    >
      <update />
    </n-tab-pane>
  </n-tabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane, NButton, NSpace, useDialog, useMessage } from "naive-ui"
import Update from "./components/Update.vue"
import Conf from "./components/Conf.vue"
import CloudAccount from "./components/CloudAccount.vue"
import SubAdmin from "./components/SubAdmin.vue"
import { useAuthStore } from "@/store/auth"
import { computed, ref } from "vue"
import { settingSystemRestart } from "@/api/modules/setting"

const authStore = useAuthStore()
const dialog = useDialog()
const message = useMessage()
const isSubAdmin = computed(() => authStore.user?.role === 'SUB_ADMIN')
const restartingPanel = ref(false)
const restartingServer = ref(false)

const highlights = [
	{
		label: "Update",
		value: "版本更新",
		desc: "集中查看版本、升级状态与日志。"
	},
	{
		label: "Access",
		value: "安全入口",
		desc: "直接维护访问端口和后台入口。"
	},
	{
		label: "Storage",
		value: "目录管理",
		desc: "快速识别数据、日志与临时目录。"
	}
]

const handleRestart = (operation: "panel" | "server") => {
	const isPanel = operation === "panel"
	dialog.warning({
		title: isPanel ? "重启面板" : "重启服务器",
		content: isPanel
			? "确定要重启 GoPanel 面板吗？当前页面连接会短暂中断。"
			: "确定要重启当前服务器吗？这会中断当前所有连接与服务。",
		positiveText: isPanel ? "立即重启面板" : "立即重启服务器",
		negativeText: "取消",
		onPositiveClick: async () => {
			if (isPanel) {
				restartingPanel.value = true
			} else {
				restartingServer.value = true
			}
			try {
				const res = await settingSystemRestart(operation)
				if (res.code === 0) {
					message.success(isPanel ? "面板重启指令已发送" : "服务器重启指令已发送")
				} else {
					message.error(res.msg || (isPanel ? "重启面板失败" : "重启服务器失败"))
				}
			} catch (_e) {
				message.error(isPanel ? "重启面板异常" : "重启服务器异常")
			} finally {
				if (isPanel) {
					restartingPanel.value = false
				} else {
					restartingServer.value = false
				}
			}
		}
	})
}
</script>
