<template>
  <div>
    <div
      v-if="precheck"
      class="bg-base-100 mt-3 rounded-[20px] p-4 px-6 shadow"
    >
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <n-tag
            class="uppercase"
            :type="precheck.runtimeKind === 'docker' ? 'success' : 'warning'"
          >
            Runtime: {{ precheck.runtimeKind }}
          </n-tag>
          <n-tag
            size="small"
            :type="precheck.hostPinned ? 'success' : 'default'"
          >
            {{ precheck.hostPinned ? '固定 Socket' : '自动探测' }}
          </n-tag>
          <n-tag
            v-if="precheck.runtimeKind === 'podman'"
            size="small"
            :type="precheck.rootlessHost ? 'warning' : 'info'"
          >
            {{ precheck.rootlessHost ? 'rootless' : 'rootful' }}
          </n-tag>
          <span class="text-sm text-gray-500">Current: {{ precheck.runtimeHost || '-' }}</span>
          <span class="text-sm text-gray-500">Configured: {{ precheck.configuredHost || '-' }}</span>
          <span class="text-sm text-gray-500">OS: {{ precheck.os }}/{{ precheck.arch }}</span>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <n-tag
            size="small"
            :type="precheck.compose?.ok ? 'success' : 'error'"
          >
            Compose: {{ precheck.compose?.ok ? (precheck.compose.bin + ' ' + precheck.compose.prefix) : '不可用' }}
          </n-tag>
          <n-tag
            size="small"
            :type="precheck.gpc?.reachable ? 'success' : 'warning'"
          >
            GPC: {{ precheck.gpc?.reachable ? 'OK' : '未连接' }}
          </n-tag>
        </div>
      </div>
      <div
        v-if="precheck.notes?.length"
        class="mt-3 space-y-1 text-xs text-orange-600"
      >
        <div
          v-for="(n, i) in precheck.notes"
          :key="i"
        >- {{ n }}</div>
      </div>
      <div
        v-if="dockerOnly"
        class="mt-3 rounded-lg bg-orange-50 p-3 text-xs text-orange-700"
      >
        当前运行时为 {{ precheck.runtimeKind }}，此页面的 daemon.json/iptables 等配置主要针对 Docker。Podman 模式下仅支持镜像加速（Linux 需连接 GPC；macOS 需 podman machine 可用）。
      </div>
    </div>

    <div class="bg-base-100 mt-3 rounded-[20px] p-4 px-6 shadow">
      <div class="flex items-center justify-between">
        <n-space align="center">
          <n-tag
            type="success"
            class="uppercase"
          >{{ daemon.containerType }}</n-tag>
          <n-tag
            type="warning"
            v-if="daemon.status"
          >
            {{ dockerStatusText[daemon.status] }}
          </n-tag>
          <span class="text-sm text-gray-500">版本: {{ daemon.version }}</span>
        </n-space>
        <n-space v-if="daemon.status">
          <n-button
            v-if="daemon.status === dockerStatus.Stopped"
            :loading="statusLoading"
            type="primary"
            @click="updateDockerStatus('start')"
          >
            {{ $t("container.start") }}
          </n-button>
          <n-popconfirm
            v-else
            @positive-click="updateDockerStatus('stop')"
          >
            <template #trigger>
              <n-button
                :loading="statusLoading"
                type="warning"
              >停止</n-button>
            </template>
            是否停止？
          </n-popconfirm>
          <n-popconfirm @positive-click="updateDockerStatus('restart')">
            <template #trigger>
              <n-button
                :loading="reloadLoading"
                :disabled="daemon.status === dockerStatus.Stopped"
                type="error"
              >
                重启
              </n-button>
            </template>
            是否重启
          </n-popconfirm>
          <n-button
            :disabled="!precheck"
            :type="repairHintType"
            @click="openRepairModal"
          >
            问题修复
          </n-button>
        </n-space>
      </div>
    </div>

    <div class="bg-base-100 mt-8 rounded-[28px] p-8 shadow">
      <n-tabs
        type="line"
        animated
        @update:value="handleTabChange"
      >
        <n-tab-pane
          name="basic"
          tab="基础配置"
        >
          <n-spin :show="daemonLoading">
            <n-form
              class="space-y-6 p-4"
              label-placement="left"
              :style="{
							maxWidth: '640px'
						}"
              label-width="100px"
            >
              <!-- 镜像加速 -->
              <n-form-item label="镜像加速">
                <div>
                  <div class="flex items-end gap-2">
                    <n-input
                      v-if="daemon.registryMirrors"
                      type="textarea"
                      :disabled="!(daemon.capabilities?.daemonJson || daemon.capabilities?.podmanRegistriesConf || (precheck?.os === 'linux' && precheck?.runtimeKind === 'podman' && precheck?.gpc?.reachable))"
                      :value="daemon.registryMirrors.join('\n')"
                      style="min-height: 34px"
                      placeholder="https://dockerpull.pw\nhttps://dockerhub.icu\nhttps://hub.rat.dev\nhttps://register.librax.org\nhttps://docker-0.unsee.tech"
                      :autosize="{ minRows: 1, maxRows: 5 }"
                      @update:value="updateMirrorUrls($event, 'registryMirrors')"
                    />
                    <n-button
                      :disabled="!(daemon.capabilities?.daemonJson || daemon.capabilities?.podmanRegistriesConf || (precheck?.os === 'linux' && precheck?.runtimeKind === 'podman' && precheck?.gpc?.reachable))"
                      @click="openDrawer('registryMirrors')"
                    >
                      <template #icon>
                        <Icon name="uil:setting" />
                      </template>
                      设置
                    </n-button>
                  </div>
                  <div
                    class="mt-1 text-xs text-gray-500"
                    style="line-height: 28px"
                  >
                    {{ $t("container.mirrorsHelper") }}
                  </div>
                </div>
              </n-form-item>

              <!-- 私有仓库 -->
              <!-- <n-form-item label="私有仓库">
							<div class="flex w-full items-end">
								<n-input
									v-if="daemon.insecureRegistries"
									type="textarea"
									disabled
									:value="daemon.insecureRegistries.join('\n')"
									style="min-height: 34px"
									placeholder="未设置"
									:autosize="{ minRows: 1, maxRows: 5 }"
									@update:value="updateMirrorUrls($event, 'insecureRegistries')"
								/>
								<n-input v-else value="未设置" disabled></n-input>
								<n-button @click="openPrivateRegistrySettings">
									<template #icon>
										<Icon name="uil:setting" />
									</template>
									设置
								</n-button>
							</div>
						</n-form-item> -->

              <!-- IPv6 -->
              <!-- <n-form-item label="IPv6">
							<n-switch v-model:value="daemon.ipv6" />
						</n-form-item> -->

              <!-- 日志切割 -->
              <n-form-item label="日志切割">
                <n-spin :show="logPruneLoading">
                  <n-switch
                    :value="logSwitchValue"
                    :disabled="dockerOnly"
                    @update:value="onLogSwitchChange"
                  />
                  <template v-if="logSwitchValue">
                    <n-space class="mt-2">
                      <n-tag type="info">单文件最大: {{ daemon.logMaxSize }}</n-tag>
                      <n-tag type="info">最大文件数: {{ daemon.logMaxFile }}</n-tag>
                    </n-space>
                  </template>
                </n-spin>
              </n-form-item>

              <!-- iptables -->
              <n-form-item label="iptables">
                <div class="w-full">
                  <n-switch
                    :value="daemon.iptables"
                    :disabled="dockerOnly"
                    @update:value="onIptablesChange"
                  />
                  <n-text
                    depth="3"
                    class="mt-1 block text-xs"
                  >Docker 对 iptables 规则的自动配置</n-text>
                </div>
              </n-form-item>

              <!-- Live restore -->
              <n-form-item label="Live restore">
                <div class="w-full">
                  <n-switch
                    :value="daemon.liveRestore"
                    :disabled="dockerOnly"
                    @update:value="onLiveRestoreChange"
                  />
                  <n-text
                    depth="3"
                    class="mt-1 block text-xs"
                  >
                    允许在 Docker 守护进程发生意外停机或崩溃时保留正在运行的容器状态
                  </n-text>
                </div>
              </n-form-item>

              <!-- cgroup driver -->
              <n-form-item label="cgroup driver">
                <n-radio-group
                  :value="daemon.cgroupDriver"
                  name="cgroupdriver"
                  :disabled="dockerOnly"
                  @update:value="onCgroupDriverChange"
                >
                  <n-radio-button
                    value="cgroupfs"
                    label="cgroupfs"
                  />
                  <n-radio-button
                    value="systemd"
                    label="systemd"
                  />
                </n-radio-group>
              </n-form-item>
              <!-- Socket 路径 -->
              <!-- <n-form-item label="Socket 路径">
                <n-input v-model:value="settings.socketPath" />
                     <n-button class="ml-2"  @click="openSocketPathSettings">
                                          <template #icon>
                      <Icon name="uil:setting" />
                    </template>
                      设置</n-button>
            </n-form-item>
             <div class="ml-[var(--n-label-width)] -mt-4">
                 <n-text depth="3" class="text-xs">Docker 守护进程 (Docker Daemon) 与客户端之间的通信通道</n-text>
            </div> -->
            </n-form>
          </n-spin>
        </n-tab-pane>
        <n-tab-pane
          name="advanced"
          tab="全部配置"
        >
          <div class="p-4">
            <FtEditor
              v-model="dockerConf"
              language="json"
              height="calc(100vh - 450px)"
              class="mt-[10px]"
            />
            <n-button
              :disabled="daemonLoading || dockerOnly"
              type="primary"
              style="margin-top: 5px"
              @click="onSaveFile"
            >
              {{ $t("commons.button.save") }}
            </n-button>
            <n-modal
              :show="showRestartConfirm"
              @update:show="showRestartConfirm = $event"
              preset="dialog"
              title="保存配置"
              :loading="saveLoading"
              positive-text="确认"
              negative-text="取消"
              @positive-click="handleConfirmRestart"
              @negative-click="showRestartConfirm = false"
            >
              <div>保存后将会重启 Docker 服务，是否确认？</div>
            </n-modal>
          </div>
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
      @confirm="handleIptablesConfirm"
    />
    <RebootAlert
      ref="rebootLiveRestoreRef"
      @confirm="handleLiveRestoreConfirm"
    />
    <RebootAlert
      ref="rebootCgroupRef"
      @confirm="handleCgroupConfirm"
    />

    <n-modal
      :show="showConfirmationModal"
      @update:show="showConfirmationModal = $event"
      preset="dialog"
      title="配置修改"
      positive-text="确认"
      negative-text="取消"
      @positive-click="handleConfirmSaveChanges"
      @negative-click="showConfirmationModal = false"
    >
      <div>修改配置后需要重启生效。</div>
      <div>
        如果确认操作，请输入
        <span style="color: red">立即重启</span>
      </div>
      <n-input
        :value="confirmationInput"
        @update:value="confirmationInput = $event"
        class="mt-2"
        placeholder='请输入 "立即重启"'
      />
    </n-modal>

    <n-modal
      :show="showRepairModal"
      @update:show="showRepairModal = $event"
      preset="dialog"
      title="容器运行时问题修复"
      positive-text="关闭"
      :show-icon="false"
      @positive-click="showRepairModal = false"
    >
      <div class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="text-sm text-gray-600">
            Runtime: {{ precheck?.runtimeKind || '-' }} / Current: {{ precheck?.runtimeHost || '-' }}
          </div>
          <div class="text-xs text-gray-500">
            Configured: {{ precheck?.configuredHost || '-' }} / 模式: {{ precheck?.hostPinned ? '固定 Socket' : '自动探测' }}
          </div>
          <n-button
            v-if="canAutoRepair"
            :loading="autoRepairLoading"
            :disabled="!precheck?.gpc?.reachable"
            type="primary"
            @click="autoRepair"
          >
            自动修复
          </n-button>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <n-tag :type="precheck?.runtime?.serviceActive ? 'success' : 'warning'">
            Service: {{ precheck?.runtime?.serviceActive ? 'active' : 'inactive' }}
          </n-tag>
          <n-tag :type="precheck?.runtime?.apiReady ? 'success' : 'warning'">
            API: {{ precheck?.runtime?.apiReady ? 'ready' : 'not ready' }}
          </n-tag>
          <n-tag :type="precheck?.gpc?.reachable ? 'success' : 'warning'">
            GPC: {{ precheck?.gpc?.reachable ? 'OK' : '未连接' }}
          </n-tag>
        </div>

        <div
          v-if="precheck?.notes?.length"
          class="rounded-lg bg-orange-50 p-3 text-xs text-orange-700"
        >
          <div
            v-for="(n, i) in precheck.notes"
            :key="i"
          >- {{ n }}</div>
        </div>

        <div class="flex flex-wrap gap-2">
          <n-button
            v-if="canAutoRepair"
            :loading="repairSocketLoading"
            :disabled="!precheck?.gpc?.reachable || autoRepairLoading"
            type="warning"
            @click="repairPodmanSocket"
          >
            修复 Podman Socket 权限
          </n-button>
          <n-button
            v-if="canAutoRepair"
            :loading="repairLingerLoading"
            :disabled="!precheck?.gpc?.reachable || autoRepairLoading"
            @click="repairLinger"
          >
            启用 Linger（rootless 保活）
          </n-button>
        </div>

        <div
          v-if="precheck && canAutoRepair && !precheck?.gpc?.reachable"
          class="text-xs text-gray-500"
        >
          GPC 未连接时无法执行一键修复，请先在服务器上启用/启动 gpc helper。
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref } from "vue"
import { loadDaemonFile, updateDaemonUpdate, updateDaemonByfile, containerInstanceOperateAPI, containerDaemonConfigAPI, containerPrecheck, repairPodmanSocketAPI, repairSystemdLingerAPI } from "../../../api/modules/container"
import { dockerStatus, dockerStatusText } from "../../../enums/dockerStatus.enum"
import { isSucc } from "../../../utils/is"
import { useMessage } from "naive-ui"

const FtEditor = defineAsyncComponent(() => import("../../../components/FtEditor/index.vue"))
const RebootAlert = defineAsyncComponent(() => import("../../../components/RebootAlert.vue"))
const DockerDrawer = defineAsyncComponent(() => import("./components/dockerDrawer.vue"))
const LogDrawer = defineAsyncComponent(() => import("./log/index.vue"))

const message = useMessage()
const statusLoading = ref(false)
const reloadLoading = ref(false)
async function updateDockerStatus(operation: string) {
	if (operation === "restart") {
		reloadLoading.value = true
	} else {
		statusLoading.value = true
	}
	try {
		const res: any = await containerInstanceOperateAPI({ operation })
		if (isSucc(res.code)) {
			getDaemon()
		}
	} finally {
		if (operation === "restart") {
			reloadLoading.value = false
		} else {
			statusLoading.value = false
		}
	}
}
const daemon = ref<any>({})
const daemonLoading = ref(false)
const daemonRetryCount = ref(0)
const daemonRetryTimer = ref<number | null>(null)

function clearDaemonRetry() {
	if (daemonRetryTimer.value) {
		window.clearTimeout(daemonRetryTimer.value)
		daemonRetryTimer.value = null
	}
}

const precheck = ref<any>(null)
const dockerOnly = computed(() => {
	if (!precheck.value?.runtimeKind) return false
	return precheck.value.runtimeKind !== "docker"
})
const canAutoRepair = computed(() => {
	if (!precheck.value) return false
	if (precheck.value?.os !== "linux") return false
	if (precheck.value?.runtimeKind === "podman") return true
	if (precheck.value?.runtimeKind === "docker" && !precheck.value?.cli?.docker && precheck.value?.cli?.podman) return true
	return false
})
const repairHintType = computed(() => {
	if (!precheck.value) return "default"
	if (precheck.value?.runtime?.serviceActive && !precheck.value?.runtime?.apiReady) return "warning"
	return "default"
})

const showRepairModal = ref(false)
const repairSocketLoading = ref(false)
const repairLingerLoading = ref(false)
const autoRepairLoading = ref(false)

async function openRepairModal() {
	await loadPrecheck()
	showRepairModal.value = true
	await autoRepair()
}

async function autoRepair() {
	if (autoRepairLoading.value) return
	await loadPrecheck()
	if (!precheck.value) return
	if (!canAutoRepair.value) return
	if (!precheck.value?.gpc?.reachable) return
	if (precheck.value?.runtime?.apiReady) return

	autoRepairLoading.value = true
	try {
		message.info("正在尝试自动修复…")
		const runtimeInfo: any = precheck.value?.runtime || {}
		const isRootless = !!runtimeInfo.rootless || !!precheck.value?.rootlessHost
		const notes = Array.isArray(precheck.value?.notes) ? precheck.value.notes.join(" ").toLowerCase() : ""
		const maybeRootless = typeof precheck.value?.runtimeHost === "string" && precheck.value.runtimeHost.includes("/run/user/")
		const needLinger =
			isRootless ||
			maybeRootless ||
			notes.includes("linger") ||
			notes.includes("user session") ||
			notes.includes("no medium found") ||
			notes.includes("cgroupv2")
		if (needLinger) {
			await repairLinger()
			await loadPrecheck()
			if (precheck.value?.runtime?.apiReady) {
				message.success("自动修复成功")
				return
			}
		}
		await repairPodmanSocket()
		await loadPrecheck()
		if (precheck.value?.runtime?.apiReady) {
			message.success("自动修复成功")
			return
		}
		message.warning("自动修复未完全成功，请查看提示信息")
	} finally {
		autoRepairLoading.value = false
	}
}

async function repairPodmanSocket() {
	repairSocketLoading.value = true
	try {
		const res: any = await repairPodmanSocketAPI()
		if (isSucc(res.code)) {
			message.success("已触发修复，正在刷新状态…")
			await loadPrecheck()
			getDaemon()
		} else {
			message.error(res.msg || "修复失败")
		}
	} catch (e: any) {
		message.error(e?.message || "修复失败")
	} finally {
		repairSocketLoading.value = false
	}
}

async function repairLinger() {
	repairLingerLoading.value = true
	try {
		const res: any = await repairSystemdLingerAPI()
		if (isSucc(res.code)) {
			message.success("已启用 linger")
			await loadPrecheck()
		} else {
			message.error(res.msg || "操作失败")
		}
	} catch (e: any) {
		message.error(e?.message || "操作失败")
	} finally {
		repairLingerLoading.value = false
	}
}

async function loadPrecheck() {
	try {
		const res: any = await containerPrecheck()
		if (isSucc(res.code)) {
			precheck.value = res.data
		}
	} catch (e) {}
}

function getDaemon(resetRetry = false) {
	if (resetRetry) {
		daemonRetryCount.value = 0
	}
	clearDaemonRetry()
	daemonLoading.value = true
	containerDaemonConfigAPI()
		.then((res: any) => {
			if (isSucc(res.code)) {
				daemon.value = res.data || {}
				daemonRetryCount.value = 0
			}
		})
		.catch((e: any) => {
			if (daemonRetryCount.value === 0) {
				message.warning("容器运行时配置暂时不可用，正在自动重试…")
			}
			if (daemonRetryCount.value < 8) {
				daemonRetryCount.value += 1
				daemonRetryTimer.value = window.setTimeout(() => getDaemon(false), 1500)
				return
			}
			message.error(e?.message || "获取容器运行时配置失败")
		})
		.finally(() => {
			daemonLoading.value = false
		})
}

function updateMirrorUrls(value: string, key: string) {
	daemon.value[key] = value
		.split("\n")
		.map(url => url.trim())
		.filter(url => url)
}

const editingMirrorUrls = ref("")
const showMirrorSettingsDrawer = ref(false)

const DockerDrawerModel = ref()
function openDrawer(key: string) {
	const needRestart = !!daemon.value?.capabilities?.daemonJson
	DockerDrawerModel.value.open(daemon.value[key], key, { needRestart })
}

const confirmationInput = ref("")
const showConfirmationModal = ref(false)
async function handleConfirmSaveChanges() {
	if (confirmationInput.value === "立即重启") {
		daemon.value.registryMirrors = editingMirrorUrls.value
			.split("\n")
			.map(url => url.trim())
			.filter(url => url)
		showMirrorSettingsDrawer.value = false
		showConfirmationModal.value = false
		message.success("镜像加速配置已更新，正在重启Docker以使配置生效...")
		updateDockerStatus("restart")
	} else {
		message.error('输入错误，请输入 "立即重启"')
	}
}
const dockerConf = ref("")

// 拉取daemon.json文件内容
async function fetchDaemonJsonFile() {
	if (!daemon.value?.capabilities?.daemonJson) {
		return
	}
	const res = await loadDaemonFile()
	if (res && res.code === 0 && typeof res.data === "string") {
		dockerConf.value = res.data
	}
}

// 页面加载时拉取一次
onMounted(() => {
	loadPrecheck()
	getDaemon(true)
})

// naive-ui n-tabs的change事件
function handleTabChange(tabName: string) {
	if (tabName === "advanced") {
		fetchDaemonJsonFile()
	}
}

const showRestartConfirm = ref(false)
const saveLoading = ref(false)

function onSaveFile() {
	showRestartConfirm.value = true
}

async function handleConfirmRestart() {
	saveLoading.value = true
	try {
		const res = await updateDaemonByfile({ file: dockerConf.value })
		if (res && res.code === 0) {
			message.success("保存成功，Docker正在重启...")
			showRestartConfirm.value = false
			fetchDaemonJsonFile()
		} else {
			message.error(res.msg || "保存失败")
		}
	} catch (e) {
		message.error("保存异常")
	} finally {
		saveLoading.value = false
	}
}

const showCgroupConfirm = ref(false)
const cgroupInput = ref("")
const rebootCgroupRef = ref()

function onCgroupDriverChange(val: string) {
	cgroupInput.value = val
	rebootCgroupRef.value.open({
		title: "cgroup driver 变更",
		input: "立即重启",
		msg: "切换 cgroup driver 后将会重启 Docker 服务。"
	})
}

async function handleCgroupConfirm() {
	rebootCgroupRef.value.close()
	daemonLoading.value = true
	try {
		const res = await updateDaemonUpdate("Driver", cgroupInput.value)
		if (res && res.code === 0) {
			message.success("cgroup driver配置已保存，Docker正在重启...")
			getDaemon()
		} else {
			message.error(res.msg || "保存失败")
		}
	} catch (e) {
		message.error("保存异常")
	} finally {
		rebootCgroupRef.value.close()
	}
}

const logDrawerRef = ref()
const logPruneLoading = ref(false)

async function onLogSwitchChange(val: boolean) {
	if (val) {
		// 打开时弹出抽屉
		logDrawerRef.value.acceptParams({
			logMaxSize: daemon.value.logMaxSize,
			logMaxFile: daemon.value.logMaxFile
		})
	} else {
		// 关闭时调用接口
		logPruneLoading.value = true
		try {
			const res = await updateDaemonUpdate("LogOption", "disable")
			if (res && res.code === 0) {
				message.success("日志切割已关闭")
				getDaemon()
			} else {
				message.error(res.msg || "操作失败")
			}
		} catch {
			message.error("操作异常")
		} finally {
			logPruneLoading.value = false
		}
	}
}

const logSwitchValue = computed(() => {
	return !!(
		daemon.value.logMaxSize &&
		daemon.value.logMaxFile &&
		daemon.value.logMaxSize !== "" &&
		daemon.value.logMaxFile !== ""
	)
})

const rebootIptablesRef = ref()
const rebootLiveRestoreRef = ref()
let iptablesTarget = false
let liveRestoreTarget = false

function onIptablesChange(val: boolean) {
	iptablesTarget = val
	rebootIptablesRef.value.open({
		title: "iptables 变更",
		input: "立即重启",
		msg: "变更 iptables 配置后需要重启 Docker 服务。"
	})
}
async function handleIptablesConfirm() {
	const value = iptablesTarget ? "enable" : "disable"
	rebootIptablesRef.value.close()
	daemonLoading.value = true
	try {
		const res = await updateDaemonUpdate("IPtables", value)
		if (res && res.code === 0) {
			message.success("iptables 配置已保存，Docker正在重启...")
			getDaemon()
		} else {
			message.error(res.msg || "保存失败")
		}
	} catch {
		message.error("保存异常")
	}
}

function onLiveRestoreChange(val: boolean) {
	liveRestoreTarget = val
	rebootLiveRestoreRef.value.open({
		title: "Live restore 变更",
		input: "立即重启",
		msg: "变更 Live restore 配置后需要重启 Docker 服务。"
	})
}
async function handleLiveRestoreConfirm() {
	const value = liveRestoreTarget ? "enable" : "disable"
	rebootLiveRestoreRef.value.close()
	daemonLoading.value = true
	try {
		const res = await updateDaemonUpdate("LiveRestore", value)
		if (res && res.code === 0) {
			message.success("Live restore 配置已保存，Docker正在重启...")
			getDaemon()
		} else {
			message.error(res.msg || "保存失败")
		}
	} catch {
		message.error("保存异常")
	}
}

defineExpose({
	onCgroupDriverChange,
	showCgroupConfirm,
	// cgroupLoading,
	handleCgroupConfirm,
	handleConfirmRestart
})
</script>

<style scoped>
/* Tailwind utility classes are used directly in the template. */
/* You can add additional styles here if needed. */
.n-form-item .n-form-item-label {
	width: 100px; /* Adjust as needed for consistent label width */
}
</style>
