<template>
  <div>
    <ContainerRuntimeBanner
      :validate="validate"
      :runtime-badge-text="runtimeBadgeText"
      :current-runtime-host="currentRuntimeHost"
      :docker-only="dockerOnly"
    />

    <ContainerDaemonStatusBar
      :daemon="daemon"
      :validate="validate"
      :status-loading="statusLoading"
      :reload-loading="reloadLoading"
      :repair-hint-type="repairHintType"
      @update-status="updateDockerStatus"
      @open-repair="openRepairModal"
    />

    <div class="mt-8 rounded-[28px] bg-base-100 p-8 shadow">
      <n-tabs
        type="line"
        animated
        @update:value="handleTabChange"
      >
        <n-tab-pane
          name="basic"
          tab="基础配置"
        >
          <ContainerBasicSettings
            :daemon="daemon"
            :validate="validate"
            :docker-only="dockerOnly"
            :daemon-loading="daemonLoading"
            :log-prune-loading="logPruneLoading"
            :log-switch-value="logSwitchValue"
            :mirror-save-loading="mirrorSaveLoading"
            @update-mirror-urls="updateMirrorUrls"
            @save-mirror-urls="saveMirrorUrls"
            @open-drawer="handleOpenDrawer"
            @log-switch-change="handleLogSwitchChange"
            @iptables-change="handleIptablesToggle"
            @live-restore-change="handleLiveRestoreToggle"
            @cgroup-driver-change="handleCgroupDriverChange"
          />
        </n-tab-pane>
        <n-tab-pane
          name="advanced"
          tab="全部配置"
        >
          <ContainerAdvancedSettings
            :docker-conf="dockerConf"
            :daemon-loading="daemonLoading"
            :docker-only="dockerOnly"
            :show-restart-confirm="showRestartConfirm"
            :save-loading="saveLoading"
            @update:docker-conf="dockerConf = $event"
            @save-file="onSaveFile"
            @confirm-restart="handleConfirmRestart"
            @update:show-restart-confirm="showRestartConfirm = $event"
          />
        </n-tab-pane>
      </n-tabs>
    </div>

    <DockerDrawer
      ref="DockerDrawerModel"
      @save="getDaemon"
    />
    <LogDrawer
      ref="logDrawerRef"
      @search="getDaemon"
    />
    <RebootAlert
      ref="rebootIptablesRef"
      @confirm="handleIptablesConfirmLocal"
    />
    <RebootAlert
      ref="rebootLiveRestoreRef"
      @confirm="handleLiveRestoreConfirmLocal"
    />
    <RebootAlert
      ref="rebootCgroupRef"
      @confirm="handleCgroupConfirmLocal"
    />

    <n-modal
      :show="showConfirmationModal"
      preset="dialog"
      title="配置修改"
      :loading="mirrorSaveLoading"
      positive-text="确认"
      negative-text="取消"
      @update:show="showConfirmationModal = $event"
      @positive-click="handleConfirmSaveChanges"
      @negative-click="showConfirmationModal = false"
    >
      <div>修改配置后需要重启生效。</div>
      <div>
        如果确认操作，请输入
        <span class="text-red-500">立即重启</span>
      </div>
      <n-input
        :value="confirmationInput"
        class="mt-2"
        placeholder='请输入 "立即重启"'
        @update:value="confirmationInput = $event"
      />
    </n-modal>

    <ContainerRepairModal
      :show="showRepairModal"
      :validate="validate"
      :runtime-detail-text="runtimeDetailText"
      :can-auto-repair="canAutoRepair"
      :auto-repair-loading="autoRepairLoading"
      :repair-socket-loading="repairSocketLoading"
      :repair-linger-loading="repairLingerLoading"
      @update:show="showRepairModal = $event"
      @auto-repair="autoRepair"
      @repair-socket="repairPodmanSocket"
      @repair-linger="repairLinger"
    />
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref } from "vue"
import { useMessage } from "naive-ui"
import ContainerAdvancedSettings from "./ContainerAdvancedSettings.vue"
import ContainerBasicSettings from "./ContainerBasicSettings.vue"
import ContainerDaemonStatusBar from "./ContainerDaemonStatusBar.vue"
import ContainerRepairModal from "./ContainerRepairModal.vue"
import ContainerRuntimeBanner from "./ContainerRuntimeBanner.vue"
import { useContainerSetting } from "./useContainerSetting"

const RebootAlert = defineAsyncComponent(() => import("../../../components/RebootAlert.vue"))
const DockerDrawer = defineAsyncComponent(() => import("./components/dockerDrawer.vue"))
const LogDrawer = defineAsyncComponent(() => import("./log/index.vue"))

const message = useMessage()
const DockerDrawerModel = ref()
const rebootCgroupRef = ref()
const logDrawerRef = ref()
const rebootIptablesRef = ref()
const rebootLiveRestoreRef = ref()

const {
  statusLoading,
  reloadLoading,
  daemon,
  daemonLoading,
  validate,
  currentRuntimeHost,
  runtimeBadgeText,
  runtimeDetailText,
  dockerOnly,
  canAutoRepair,
  repairHintType,
  showRepairModal,
  repairSocketLoading,
  repairLingerLoading,
  autoRepairLoading,
  updateDockerStatus,
  openRepairModal,
  autoRepair,
  repairPodmanSocket,
  repairLinger,
  getDaemon,
  updateMirrorUrls,
  saveMirrorUrls,
  openDrawer,
  showConfirmationModal,
  confirmationInput,
  mirrorSaveLoading,
  handleConfirmSaveChanges,
  dockerConf,
  handleTabChange,
  showRestartConfirm,
  saveLoading,
  onSaveFile,
  handleConfirmRestart,
  onCgroupDriverChange,
  handleCgroupConfirm,
  logPruneLoading,
  onLogSwitchChange,
  logSwitchValue,
  onIptablesChange,
  handleIptablesConfirm,
  onLiveRestoreChange,
  handleLiveRestoreConfirm
} = useContainerSetting(message)

const handleOpenDrawer = (key: string) => openDrawer(DockerDrawerModel, key)
const handleLogSwitchChange = (val: boolean) => onLogSwitchChange(val, logDrawerRef)
const handleCgroupDriverChange = (val: string) => onCgroupDriverChange(val, rebootCgroupRef)
const handleCgroupConfirmLocal = () => handleCgroupConfirm(rebootCgroupRef)
const handleIptablesToggle = (val: boolean) => onIptablesChange(val, rebootIptablesRef)
const handleIptablesConfirmLocal = () => handleIptablesConfirm(rebootIptablesRef)
const handleLiveRestoreToggle = (val: boolean) => onLiveRestoreChange(val, rebootLiveRestoreRef)
const handleLiveRestoreConfirmLocal = () => handleLiveRestoreConfirm(rebootLiveRestoreRef)

defineExpose({
  handleCgroupDriverChange,
  handleCgroupConfirm: handleCgroupConfirmLocal,
  handleConfirmRestart
})
</script>
