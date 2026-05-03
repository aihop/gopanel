<script setup lang="ts">
import type { ProcessStatusTagType } from "./processColumns"

defineProps<{
  show: boolean
  detailDrawerTitle: string
  selectedProcess: any
  getStatusType: (status: string | undefined) => ProcessStatusTagType
  openFilesColumns: any[]
  drawerNetworkConnectionsColumns: any[]
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()
</script>

<template>
  <n-drawer
    :show="show"
    :width="500"
    placement="right"
    @update:show="emit('update:show', $event)"
  >
    <n-drawer-content
      :title="detailDrawerTitle"
      closable
    >
      <n-tabs
        type="line"
        animated
      >
        <n-tab-pane
          name="basicInfo"
          tab="基本信息"
        >
          <n-descriptions
            label-placement="left"
            bordered
            :column="1"
            size="small"
          >
            <n-descriptions-item label="名称">{{ selectedProcess?.name }}</n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag
                :type="getStatusType(selectedProcess?.status)"
                size="small"
              >
                {{ selectedProcess?.status }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="进程ID">{{ selectedProcess?.PID }}</n-descriptions-item>
            <n-descriptions-item label="父进程ID">{{ selectedProcess?.PPID }}</n-descriptions-item>
            <n-descriptions-item label="线程">{{ selectedProcess?.numThreads }}</n-descriptions-item>
            <n-descriptions-item label="连接">{{ selectedProcess?.numConnections ?? "N/A" }}</n-descriptions-item>
            <n-descriptions-item label="磁盘读">{{ selectedProcess?.diskRead ?? "N/A" }}</n-descriptions-item>
            <n-descriptions-item label="磁盘写">{{ selectedProcess?.diskWrite ?? "N/A" }}</n-descriptions-item>
            <n-descriptions-item label="用户">{{ selectedProcess?.username }}</n-descriptions-item>
            <n-descriptions-item label="启动时间">{{ selectedProcess?.startTime }}</n-descriptions-item>
            <n-descriptions-item label="启动命令">{{ selectedProcess?.cmdLine ?? "N/A" }}</n-descriptions-item>
          </n-descriptions>
        </n-tab-pane>
        <n-tab-pane
          name="memoryInfo"
          tab="内存信息"
        >
          <n-descriptions
            v-if="selectedProcess?.memoryInfo"
            label-placement="left"
            bordered
            :columns="2"
            size="small"
          >
            <n-descriptions-item label="rss">{{ selectedProcess.memoryInfo.rss }}</n-descriptions-item>
            <n-descriptions-item label="swap">{{ selectedProcess.memoryInfo.swap }}</n-descriptions-item>
            <n-descriptions-item label="vms">{{ selectedProcess.memoryInfo.vms }}</n-descriptions-item>
            <n-descriptions-item label="hwm">{{ selectedProcess.memoryInfo.hwm }}</n-descriptions-item>
            <n-descriptions-item label="data">{{ selectedProcess.memoryInfo.data }}</n-descriptions-item>
            <n-descriptions-item label="stack">{{ selectedProcess.memoryInfo.stack }}</n-descriptions-item>
            <n-descriptions-item label="locked">{{ selectedProcess.memoryInfo.locked }}</n-descriptions-item>
          </n-descriptions>
          <p v-else>暂无内存信息</p>
        </n-tab-pane>
        <n-tab-pane
          name="fileOpen"
          tab="文件打开"
        >
          <n-data-table
            v-if="selectedProcess?.openFiles?.length"
            :columns="openFilesColumns"
            :data="selectedProcess.openFiles"
            :pagination="false"
            :bordered="true"
            size="small"
          />
          <p v-else>暂无打开文件信息</p>
        </n-tab-pane>
        <n-tab-pane
          name="envVar"
          tab="环境变量"
        >
          <n-code
            v-if="selectedProcess?.environmentVariables"
            language="js"
            :code="selectedProcess.environmentVariables"
            show-line-numbers
          />
          <p v-else>暂无环境变量信息</p>
        </n-tab-pane>
        <n-tab-pane
          name="networkLink"
          tab="网络连接"
        >
          <n-data-table
            v-if="selectedProcess?.connects?.length"
            :columns="drawerNetworkConnectionsColumns"
            :data="selectedProcess.connects"
            :pagination="false"
            :bordered="true"
            size="small"
          />
          <p v-else>暂无网络连接信息</p>
        </n-tab-pane>
      </n-tabs>
    </n-drawer-content>
  </n-drawer>
</template>
