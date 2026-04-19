<template>
  <n-spin :show="loading">
    <div
      v-if="apps.length"
      class="apps-grid"
    >
      <div
        v-for="item in apps"
        :key="item.id"
        class="app-card"
      >
        <div class="app-card__glow"></div>
        <div class="app-card__body">
          <div class="app-card__header">
            <div class="app-card__identity">
              <div class="app-card__icon">
                <img
                  v-if="item.icon"
                  :src="item.icon"
                  alt="icon"
                  class="h-10 w-10 object-contain"
                />
                <span
                  v-else
                  class="text-base font-bold text-blue-600"
                >
                  {{ item.name?.slice(0, 1)?.toUpperCase() }}
                </span>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <div class="truncate text-base font-semibold text-slate-900">{{ item.name }}</div>
                  <n-tag
                    v-if="item.installing"
                    type="warning"
                    size="small"
                    round
                  >安装中</n-tag>
                  <n-tag
                    v-else-if="item.installed"
                    type="success"
                    size="small"
                    round
                  >已安装</n-tag>
                </div>
                <div class="mt-1 text-sm text-slate-500">{{ item.type || "应用服务" }}</div>
              </div>
            </div>
            <div class="app-card__actions">
              <n-button
                secondary
                size="small"
                @click="() => openDetailDrawer(item)"
              >{{ $t('app.detail') }}</n-button>
              <n-button
                v-if="item.installed"
                secondary
                type="info"
                size="small"
                disabled
              >{{ $t('app.install') }}</n-button>
              <n-button
                v-else
                secondary
                type="info"
                size="small"
                :disabled="item.installing"
                @click="() => handleInstallApp(item)"
              >
                {{ item.installing ? $t('commons.status.installing') : $t('commons.operate.install') }}
              </n-button>
            </div>
          </div>

          <p class="app-card__desc">{{ item.shortDescZh || item.description || "暂无应用说明，点击详情查看完整信息。" }}</p>

          <div class="app-card__meta">
            <div class="app-chip">
              <span class="app-chip__label">来源</span>
              <span class="app-chip__value">{{ item.resource || "应用商店" }}</span>
            </div>
            <div
              v-if="item.versions && item.versions.length"
              class="app-chip"
            >
              <span class="app-chip__label">版本</span>
              <span class="app-chip__value">{{ item.versions[0] }}</span>
            </div>
          </div>

          <div
            v-if="item.versions && item.versions.length > 1"
            class="app-card__versions"
          >
            <n-tag
              v-for="version in item.versions.slice(0, 3)"
              :key="version"
              size="small"
              round
              :bordered="false"
            >
              {{ version }}
            </n-tag>
            <span
              v-if="item.versions.length > 3"
              class="text-xs text-slate-400"
            >
              +{{ item.versions.length - 3 }} 个版本
            </span>
          </div>
        </div>
      </div>
    </div>
    <div
      v-else
      class="app-empty"
    >
      暂无匹配的应用
    </div>
  </n-spin>

  <n-modal
    v-model:show="showInstallModal"
    preset="dialog"
    :title="`安装 ${currentApp?.name}`"
    style="width: 600px"
  >
    <n-spin :show="installLoading">
      <n-form
        ref="formRef"
        :model="formModel"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item
          label="名称"
          path="name"
        >
          <n-input
            v-model:value="formModel.name"
            placeholder="请输入应用名称"
          />
        </n-form-item>
        <n-form-item
          label="版本"
          path="version"
        >
          <n-select
            v-model:value="formModel.version"
            :options="versionOptions"
            placeholder="请选择版本"
            @update:value="handleVersionChange"
          />
        </n-form-item>

        <template v-if="formFields && formFields.length > 0">
          <n-divider>参数配置</n-divider>
          <n-form-item
            v-for="field in formFields"
            :key="field.envKey"
            :label="field.labelZh || field.label?.zh || field.labelEn || field.envKey"
            :path="'params.' + field.envKey"
          >
            <template v-if="field.type === 'number'">
              <n-input-number
                v-model:value="formModel.params[field.envKey]"
                :min="field.min"
                :max="field.max"
                style="width: 100%"
              />
            </template>
            <template v-else-if="field.type === 'password'">
              <n-input
                v-model:value="formModel.params[field.envKey]"
                type="password"
                show-password-on="click"
              />
            </template>
            <template v-else-if="field.type === 'select'">
              <n-select
                v-model:value="formModel.params[field.envKey]"
                :options="(field.values || []).map((v) => ({ label: v.label, value: v.value }))"
              />
            </template>
            <template v-else>
              <n-input v-model:value="formModel.params[field.envKey]" />
            </template>

            <template
              #feedback
              v-if="field.description"
            >
              <div style="font-size: 12px; color: #999; margin-top: 4px;">
                {{ field.description.zh || field.description.en || field.description }}
              </div>
            </template>
          </n-form-item>
        </template>

        <n-collapse>
          <n-collapse-item
            title="高级配置"
            name="advanced"
          >
            <n-form-item label="允许端口外部访问">
              <n-switch v-model:value="formModel.allowPort" />
            </n-form-item>
            <n-form-item label="容器名称">
              <n-input
                v-model:value="formModel.containerName"
                placeholder="默认自动生成"
              />
            </n-form-item>
            <n-form-item label="CPU 限制(核)">
              <n-input-number
                v-model:value="formModel.cpuQuota"
                :min="0"
                :step="0.1"
                style="width: 100%"
              />
            </n-form-item>
            <n-form-item label="内存限制">
              <n-input-group>
                <n-input-number
                  v-model:value="formModel.memoryLimit"
                  :min="0"
                  style="width: 70%"
                />
                <n-select
                  v-model:value="formModel.memoryUnit"
                  :options="[
											{ label: 'MB', value: 'm' },
											{ label: 'GB', value: 'g' }
										]"
                  style="width: 30%"
                />
              </n-input-group>
            </n-form-item>
          </n-collapse-item>
        </n-collapse>
      </n-form>
    </n-spin>
    <template #action>
      <n-button @click="showInstallModal = false">取消</n-button>
      <n-button
        type="primary"
        :loading="installLoading"
        @click="submitInstall"
      >确认安装</n-button>
    </template>
  </n-modal>

  <n-drawer
    v-model:show="showDetailDrawer"
    :width="700"
    placement="right"
  >
    <n-drawer-content :title="(detailApp?.name + $t('app.detail')) ">
      <n-spin :show="detailLoading">
        <div
          v-if="detailApp"
          class="app-detail-container"
        >
          <div class="flex items-center mb-6">
            <img
              v-if="detailApp.icon"
              :src="detailApp.icon"
              class="w-16 h-16 mr-4"
            />
            <div>
              <div class="text-xl font-bold mb-1">{{ detailApp.name }}</div>
              <div class="text-gray-500 text-sm">{{ detailApp.shortDescZh || detailApp.description }}</div>
            </div>
          </div>

          <n-descriptions
            label-placement="left"
            :column="1"
            bordered
            size="small"
            class="mb-6"
          >
            <n-descriptions-item label="分类">{{ detailApp.type }}</n-descriptions-item>
            <n-descriptions-item
              label="可选版本"
              v-if="detailApp.versions && detailApp.versions.length"
            >
              <n-space>
                <n-tag
                  v-for="v in detailApp.versions"
                  :key="v"
                  size="small"
                  type="info"
                >{{ v }}</n-tag>
              </n-space>
            </n-descriptions-item>
            <n-descriptions-item
              label="相关链接"
              v-if="detailApp.website || detailApp.document || detailApp.github"
            >
              <n-space>
                <n-button
                  v-if="detailApp.website"
                  text
                  tag="a"
                  :href="detailApp.website"
                  target="_blank"
                  type="primary"
                >官网</n-button>
                <n-button
                  v-if="detailApp.document"
                  text
                  tag="a"
                  :href="detailApp.document"
                  target="_blank"
                  type="primary"
                >文档</n-button>
                <n-button
                  v-if="detailApp.github"
                  text
                  tag="a"
                  :href="detailApp.github"
                  target="_blank"
                  type="primary"
                >GitHub</n-button>
              </n-space>
            </n-descriptions-item>
          </n-descriptions>

          <div v-if="detailApp.readMe">
            <div class="text-lg font-bold mb-4">应用介绍</div>
            <MdPreview
              editorId="app-readme"
              :modelValue="detailApp.readMe"
            />
          </div>
        </div>
      </n-spin>
      <template #footer>
        <n-space>
          <n-button @click="showDetailDrawer = false">关闭</n-button>
          <n-button
            v-if="detailApp?.installed"
            type="info"
            secondary
            disabled
          >已安装</n-button>
          <n-button
            v-else
            type="primary"
            :disabled="detailApp?.installing"
            @click="handleInstallFromDetail"
          >{{ detailApp?.installing ? '安装中' : '去安装' }}</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>

  <!-- 安装进度弹窗 (SSE) -->
  <n-modal
    v-model:show="logModalVisible"
    preset="card"
    title="应用安装进度"
    style="width: 700px"
    :mask-closable="false"
    :closable="true"
  >
    <div
      ref="terminalRef"
      class="bg-[#1e1e1e] p-4 rounded-md h-[400px] overflow-y-auto text-[#d4d4d4] font-mono text-sm leading-relaxed"
    >
      <div
        v-for="(log, index) in logsData"
        :key="index"
        class="whitespace-pre-wrap break-all"
      >
        <span
          v-if="log.includes('ERROR')"
          class="text-red-400"
        >{{ log }}</span>
        <span
          v-else-if="log.includes('INFO')"
          class="text-blue-300"
        >{{ log }}</span>
        <span v-else>{{ log }}</span>
      </div>
    </div>
    <template #action>
      <n-button
        :disabled="!isInstallFinished"
        type="primary"
        @click="handleLogModalClose"
      >关闭</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch, reactive, nextTick } from "vue"
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
// @ts-ignore
import { appsSearchAPI, GetApp, InstallApp, GetAppDetail } from "@/api/modules/apps"
import type { AppsSearchParams } from "@/api/modules/apps"
import { useMessage } from "naive-ui"
import { useRouter } from "vue-router"
import { useAuthStore } from "@/store/auth"

const props = defineProps<{
	searchName: string
	page: number
	pageSize: number
}>()
const emits = defineEmits(["update:total"])

const router = useRouter()
const message = useMessage()
const apps = ref<any[]>([])
const loading = ref(false)
const authStore = useAuthStore()

// 详情相关
const showDetailDrawer = ref(false)
const detailApp = ref<any>(null)
const detailLoading = ref(false)

// 安装相关
const showInstallModal = ref(false)
const installLoading = ref(false)
const currentApp = ref<any>(null)
const appDetail = ref<any>(null)
const versionOptions = ref<any[]>([])

// 日志 SSE 相关
const logModalVisible = ref(false)
const logsData = ref<string[]>([])
const isInstallFinished = ref(false)
const terminalRef = ref<HTMLElement | null>(null)
let logEventSource: EventSource | null = null
const formFields = ref<any[]>([])
const formRef = ref<any>(null)

const formModel = reactive({
	name: "",
	version: "",
	params: {} as Record<string, any>,
	advanced: false,
	allowPort: false,
	containerName: "",
	cpuQuota: 0,
	memoryLimit: 0,
	memoryUnit: "m",
	hostMode: undefined as boolean | undefined
})

const rules = {
	name: { required: true, message: "请输入应用名称", trigger: "blur" },
	version: { required: true, message: "请选择版本", trigger: "change" }
}

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
		message.error("获取应用列表失败")
	} finally {
		loading.value = false
	}
}

watch([() => props.searchName, () => props.page, () => props.pageSize], fetchData, { immediate: true })

async function handleInstallApp(item: any) {
	currentApp.value = item
	const loadingMsg = message.loading("获取应用信息中...", { duration: 0 })
	try {
		const res = await GetApp(item.key)
		if (res.code === 0 && res.data) {
			appDetail.value = res.data
			versionOptions.value = (res.data.versions || []).map((v: string) => ({ label: v, value: v }))

			// 重置表单
			formModel.name = `${item.key}-${Math.random().toString(36).substring(2, 6)}`
			formModel.version = versionOptions.value.length > 0 ? versionOptions.value[0].value : ""
			formModel.params = {}
			formModel.advanced = false
			formModel.allowPort = false
			formModel.containerName = ""
			formModel.cpuQuota = 0
			formModel.memoryLimit = 0
			formModel.memoryUnit = "m"

			if (formModel.version) {
				await handleVersionChange(formModel.version)
			}

			showInstallModal.value = true
		} else {
			message.error(res.msg || "获取应用详情失败")
		}
	} catch (e) {
		message.error("获取应用详情异常")
	} finally {
		loadingMsg.destroy()
	}
}

async function openDetailDrawer(item: any) {
	detailApp.value = item
	showDetailDrawer.value = true
	detailLoading.value = true
	try {
		const res = await GetApp(item.key)
		if (res.code === 0 && res.data) {
			detailApp.value = res.data
		}
	} finally {
		detailLoading.value = false
	}
}

function handleInstallFromDetail() {
	showDetailDrawer.value = false
	if (detailApp.value) {
		handleInstallApp(detailApp.value)
	}
}

async function handleVersionChange(version: string) {
	if (!appDetail.value || !appDetail.value.id) return

	try {
		installLoading.value = true
		const res: any = await GetAppDetail(appDetail.value.id, version)
		if (res.code === 0) {
			const detail = res.data
			appDetail.value.appDetail = detail
			
			// Extract parameters from the returned parsed params map
			if (detail.params && detail.params.formFields) {
				formFields.value = detail.params.formFields
			} else {
				formFields.value = []
			}
			
			// Initialize default values
			formFields.value.forEach((field) => {
				formModel.params[field.envKey] = field.default !== undefined ? field.default : ""
			})
			
			// Save the appDetailId for installation
			appDetail.value.appDetailId = detail.id
			
			// Set HostMode if available
			if (detail.hostMode !== undefined) {
				formModel.hostMode = detail.hostMode
			}
		}
	} catch (e) {
		message.error("获取应用版本详情失败")
	} finally {
		installLoading.value = false
	}
}

async function submitInstall() {
	formRef.value?.validate(async (errors: any) => {
		if (!errors) {
			installLoading.value = true
			try {
				// 判断是否开启高级配置
				formModel.advanced =
					formModel.allowPort ||
					!!formModel.containerName ||
					formModel.cpuQuota > 0 ||
					formModel.memoryLimit > 0

				const reqData = {
					name: formModel.name,
					appDetailId: appDetail.value.appDetailId,
					params: formModel.params,
					advanced: formModel.advanced,
					allowPort: formModel.allowPort,
					containerName: formModel.containerName,
					cpuQuota: formModel.cpuQuota,
					memoryLimit: formModel.memoryLimit,
					memoryUnit: formModel.memoryUnit,
					pullImage: true,
					editCompose: false
				}

				const res = await InstallApp(reqData as any)
				if (res.code === 0) {
					message.success("应用开始安装")
					showInstallModal.value = false
					
					// 打开日志模态框，监听 SSE
					logModalVisible.value = true
					logsData.value = []
					isInstallFinished.value = false
					
					if (logEventSource) {
						logEventSource.close()
					}
 
					// 修改为直接使用 /api 路径，因为后端的实际路由是 /api/apps/install/:name/logs
					const apiUrl = "/api"
					const token = authStore.getAuth() || authStore.auth || ""
					logEventSource = new EventSource(`${apiUrl}/apps/install/${reqData.name}/logs?token=${token}`)
					
					// 更新状态为"安装中"
					if (currentApp.value) {
						currentApp.value.installed = false
						currentApp.value.installing = true
					}
					const appIndex = apps.value.findIndex(a => a.key === currentApp.value?.key)
					if (appIndex !== -1) {
						apps.value[appIndex].installed = false
						apps.value[appIndex].installing = true
					}

					logEventSource.onmessage = (event) => {
						if (event.data === "ping") return
						if (event.data === "EOF" || event.data === '["EOF"]') {
							logEventSource?.close()
							isInstallFinished.value = true
							logsData.value.push("\n====== 安装流程结束 ======")
							scrollToBottom()
							// 安装完成后刷新列表或更新状态
							if (currentApp.value) {
								currentApp.value.installing = false
								currentApp.value.installed = true
							}
							if (appIndex !== -1) {
								apps.value[appIndex].installing = false
								apps.value[appIndex].installed = true
							}
							fetchData() // 重新拉取最新状态
							return
						}
						logsData.value.push(event.data)
						scrollToBottom()
					}
					
					logEventSource.onerror = (err) => {
						console.error("SSE Error:", err)
						if (!isInstallFinished.value) {
							logsData.value.push("\n[系统提示] 与日志服务器的连接已断开或发生错误。")
							isInstallFinished.value = true
							// 发生异常时，恢复安装中状态
							if (currentApp.value) {
								currentApp.value.installing = false
							}
							if (appIndex !== -1) {
								apps.value[appIndex].installing = false
							}
						}
						logEventSource?.close()
					}
					
				} else {
					message.error(res.msg || "安装请求失败")
				}
			} catch (e: any) {
				message.error(e.message || "安装异常")
			} finally {
				installLoading.value = false
			}
		}
	})
}

function handleLogModalClose() {
	logModalVisible.value = false
	if (logEventSource) {
		logEventSource.close()
	}
}

const scrollToBottom = () => {
	nextTick(() => {
		if (terminalRef.value) {
			terminalRef.value.scrollTop = terminalRef.value.scrollHeight
		}
	})
}
</script>

<style scoped>
.app-card {
	position: relative;
	overflow: hidden;
	border-radius: 24px;
	border: 1px solid rgba(226, 232, 240, 0.88);
	background:
		radial-gradient(circle at top right, rgba(59, 130, 246, 0.12), transparent 28%),
		linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.92));
	box-shadow: 0 14px 36px rgba(15, 23, 42, 0.06);
	transition: transform 0.26s ease, box-shadow 0.26s ease, border-color 0.26s ease;
}

.app-card:hover {
	transform: translateY(-4px);
	border-color: rgba(59, 130, 246, 0.22);
	box-shadow: 0 22px 44px rgba(15, 23, 42, 0.1);
}

.apps-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
	gap: 18px;
}

.app-card__glow {
	position: absolute;
	top: -40px;
	right: -32px;
	width: 120px;
	height: 120px;
	border-radius: 9999px;
	background: rgba(59, 130, 246, 0.14);
	filter: blur(20px);
	pointer-events: none;
}

.app-card__body {
	position: relative;
	z-index: 1;
	display: flex;
	flex-direction: column;
	gap: 18px;
	padding: 22px;
	height: 100%;
}

.app-card__header {
	display: flex;
	justify-content: space-between;
	gap: 14px;
}

.app-card__identity {
	display: flex;
	align-items: flex-start;
	gap: 12px;
	min-width: 0;
	flex: 1;
}

.app-card__icon {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 54px;
	height: 54px;
	flex-shrink: 0;
	border-radius: 18px;
	border: 1px solid rgba(219, 234, 254, 0.9);
	background: linear-gradient(135deg, rgba(239, 246, 255, 0.95), rgba(255, 255, 255, 0.75));
	box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.65);
}

.app-card__actions {
	display: flex;
	align-items: center;
	gap: 8px;
	flex-wrap: wrap;
	justify-content: flex-end;
}

.app-card__desc {
	min-height: 66px;
	margin: 0;
	font-size: 0.92rem;
	line-height: 1.7;
	color: rgb(100 116 139);
}

.app-card__meta {
	display: grid;
	grid-template-columns: repeat(2, minmax(0, 1fr));
	gap: 10px;
}

.app-chip {
	padding: 10px 12px;
	border-radius: 16px;
	background: rgba(248, 250, 252, 0.95);
	border: 1px solid rgba(226, 232, 240, 0.9);
}

.app-chip__label {
	display: block;
	font-size: 0.72rem;
	color: rgb(148 163 184);
	margin-bottom: 4px;
}

.app-chip__value {
	display: block;
	font-size: 0.88rem;
	font-weight: 600;
	color: rgb(30 41 59);
	word-break: break-word;
}

.app-card__versions {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	gap: 8px;
	padding-top: 2px;
}

.app-empty {
	padding: 64px 20px;
	text-align: center;
	font-size: 0.95rem;
	color: rgb(148 163 184);
}

@media (max-width: 640px) {
	.apps-grid {
		grid-template-columns: 1fr;
	}

	.app-card__header {
		flex-direction: column;
	}

	.app-card__actions {
		justify-content: flex-start;
	}

	.app-card__meta {
		grid-template-columns: 1fr;
	}
}
</style>
