<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <n-drawer
    v-model:show="visible"
    :width="502"
    :mask-closable="false"
  >
    <n-drawer-content closable>
      <template #header>
        <div class="flex items-center gap-4">
          <n-button
            text
            @click="close"
          >
            <template #icon>
              <Icon name="mdi:arrow-left" />
            </template>
            返回
          </n-button>
          <n-divider vertical />
          <div>{{ title }}</div>
        </div>
      </template>

      <n-form
        ref="formRef"
        :model="form"
        :rules="rules"
        require-mark-placement="right-hanging"
      >
        <n-form-item
          label="应用类型"
          path="type"
        >
          <n-select
            v-model:value="form.type"
            :disabled="actionType === 'update'"
            :options="[
              { label: '📦 静态网站 (HTML/Vue/React)', value: 'static' },
              { label: '🚀 容器化应用 (需 Docker 镜像)', value: 'web_app' },
              { label: '🔌 纯反向代理 (不托管代码)', value: 'proxy' }
            ]"
            placeholder="请选择类型"
          />
        </n-form-item>
        <n-form-item
          :label="$t('website.primaryDomain')"
          path="primaryDomain"
        >
          <n-input
            v-model:value="form.primaryDomain"
            :disabled="actionType === 'update'"
            placeholder="请输入域名，例如 console.cn"
          />
        </n-form-item>
        <n-form-item
          label="其他域名"
          path="otherDomains"
        >
          <n-input
            type="textarea"
            v-model:value="form.otherDomains"
            placeholder="一行一个域名，支持IP地址"
          />
        </n-form-item>

        <n-checkbox
          class="mb-6"
          v-model:checked="form.IPV6"
        >监听IPV6</n-checkbox>

        <n-form-item
          label="网站目录"
          path="alias"
        >
          <n-input
            v-model:value="form.alias"
            :disabled="actionType === 'update'"
            placeholder="网站目录、代号"
          />
        </n-form-item>

        <template v-if="form.type === 'web_app' || form.type === 'static'">
          <n-form-item
            label="代码来源 / 部署方式"
            path="codeSource"
          >
            <n-radio-group v-model:value="form.codeSource">
              <n-space>
                <n-radio
                  value="upload"
                  v-if="form.type === 'static'"
                >稍后上传 / 本地目录</n-radio>
                <n-radio value="git">自定义镜像部署</n-radio>
                <n-radio value="pipeline">关联流水线</n-radio>
                <n-radio value="app_store">应用商店</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item>

          <n-form-item
            label="容器镜像地址"
            path="gitRepo"
            v-if="form.codeSource === 'git' || form.codeSource === 'docker'"
          >
            <n-auto-complete
              v-model:value="form.gitRepo"
              :options="localImageOptions"
              placeholder="例：nginx:latest 或 my-harbor/app:v1"
              :get-show="() => true"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="本地代码路径 (可选)"
            path="codeDir"
            v-if="form.codeSource === 'upload'"
          >
            <n-input
              v-model:value="form.codeDir"
              placeholder="默认：/opt/gopanel/wwwroot/{代号}/releases"
            />
          </n-form-item>

          <div
            v-if="form.codeSource === 'upload'"
            class="mb-6 text-sm text-slate-500 border p-3 rounded bg-slate-50"
          >
            创建完成后，您可以在网站列表点击 <b>[部署管理] -> [发布新版本]</b> 来上传代码压缩包。或者直接将代码放入上方目录。
          </div>

        </template>

        <n-form-item
          label="代理地址 (目标内部端口)"
          path="proxy"
          v-if="form.type === 'proxy' || form.type === 'web_app'"
        >
          <div class="w-full">
            <n-input
              v-model:value="form.proxy"
              placeholder="例：http://127.0.0.1:8080 或 8080"
            />
            <div
              v-if="form.codeSource === 'pipeline'"
              class="mt-2 text-xs text-slate-500"
            >
              这里仅作为待部署前的占位代理地址。流水线成功部署后，系统会自动写入真实的本机随机端口代理地址。
            </div>
          </div>
        </n-form-item>

        <n-form-item
          label="选择已安装应用"
          path="appInstallId"
          v-if="form.codeSource === 'app_store'"
        >
          <n-select
            v-model:value="form.appInstallId"
            :options="appInstallOptions"
            placeholder="请选择已安装的容器应用"
            @update:value="handleAppSelect"
          />
        </n-form-item>

        <n-form-item
          label="选择流水线"
          path="pipelineId"
          v-if="form.codeSource === 'pipeline'"
        >
          <n-select
            v-model:value="form.pipelineId"
            :options="pipelineOptions"
            placeholder="请选择关联的部署流水线"
            @update:value="handlePipelineSelect"
          />
        </n-form-item>

        <n-divider class="!my-6 text-slate-400">安全防护</n-divider>

        <n-form-item label="防爬虫 (拦截恶意扫描)">
          <div class="flex flex-col gap-2">
            <n-switch v-model:value="form.antiCrawler" />
            <div class="text-xs text-slate-500">开启后将在 Caddy 层拦截常见恶意扫描器、无头浏览器及部分不规范 User-Agent，降低服务器开销。</div>
          </div>
        </n-form-item>

        <n-form-item label="防盗链 (保护静态资源)">
          <div class="flex flex-col gap-2">
            <n-switch v-model:value="form.antiLeech" />
            <div class="text-xs text-slate-500">开启后仅允许本站域名引用图片、视频等静态资源，防止流量被外部网站“薅羊毛”。</div>
          </div>
        </n-form-item>

        <n-form-item label="并发与频率限制 (防 CC)">
          <n-select
            v-model:value="form.rateLimitMode"
            :options="[
              { label: '关闭 (默认)', value: 'none' },
              { label: '常规防护 (推荐)', value: 'normal' },
              { label: '严格防护 (应急)', value: 'strict' }
            ]"
          />
        </n-form-item>

        <n-form-item label="轻量级 WAF (防注入/XSS)">
          <div class="flex flex-col gap-2">
            <n-switch v-model:value="form.wafEnable" />
            <div class="text-xs text-slate-500">拦截常见的 SQL 注入、跨站脚本及系统路径穿越攻击，为建站程序提供基础防护。</div>
          </div>
        </n-form-item>

        <n-form-item label="保护敏感配置文件">
          <div class="flex flex-col gap-2">
            <n-switch v-model:value="form.blockSensitive" />
            <div class="text-xs text-slate-500">禁止外部访问 .env、.git 等隐藏文件及 .sql、.bak 等备份数据，防止源码与数据库泄露。</div>
          </div>
        </n-form-item>

        <n-divider class="!my-4" />

        <n-form-item
          label="备注"
          path="remark"
        >
          <n-input
            type="textarea"
            v-model:value="form.remark"
            placeholder="请输入备注信息"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-4">
          <n-button @click="close">取消</n-button>
          <n-button
            type="primary"
            @click="onConfirm"
            :loading="loading"
          >确定</n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>
<script setup lang="ts">
// @ts-nocheck
import type { FormInst } from "naive-ui"
import type { Website } from "@/api/interface/website"
import { ref, watch, onMounted } from "vue"
import { websiteCreateAPI, websiteUpdateAPI } from "@/api/modules/website"
import { ListAppInstalled } from "@/api/modules/apps"
import { getPipelinePage } from "@/api/modules/pipeline"
import { listAllImage } from "@/api/modules/container"
import { useMessage } from "naive-ui"

const visible = ref(false)
const loading = ref(false)
const emit = defineEmits(["confirm"])
const formRef = ref<FormInst | null>(null)
const message = useMessage()
const actionType = ref("add")

const title = ref("添加域名")

const appInstallOptions = ref<{ label: string; value: number }[]>([])
const rawAppList = ref<any[]>([])

const pipelineOptions = ref<any[]>([])
const rawPipelineList = ref<any[]>([])

const localImageOptions = ref<{ label: string; value: string }[]>([])

const fetchApps = async () => {
	try {
		const res = await ListAppInstalled()
		if (res.data) {
			rawAppList.value = res.data
			appInstallOptions.value = res.data.map((app: any) => ({
				label: `${app.name} ${app.httpPort ? `(端口: ${app.httpPort})` : ''}`,
				value: app.id
			}))
		}
	} catch (e) {
		console.error("Failed to fetch installed apps", e)
	}
}

const fetchLocalImages = async () => {
	try {
		const res = await listAllImage()
		if (res.data) {
			localImageOptions.value = res.data.map((item: any) => ({
				label: item.tags && item.tags.length > 0 ? item.tags[0] : item.name,
				value: item.tags && item.tags.length > 0 ? item.tags[0] : item.name
			}))
		}
	} catch (e) {
		console.error("Failed to fetch local images", e)
	}
}

const getPipelineList = async () => {
	try {
		const res: any = await getPipelinePage({ page: 1, limit: 100 })
		rawPipelineList.value = res.data.items
		pipelineOptions.value = res.data.items.map((item: any) => {
			return {
				label: `${item.name} (#${item.id})`,
				value: item.id
			}
		})
	} catch (error) {
		console.error(error)
	}
}

const handleAppSelect = (val: number) => {
	if (!val) return
	const app = rawAppList.value.find((a: any) => a.id === val)
	if (app && app.httpPort) {
		form.value.proxy = `http://127.0.0.1:${app.httpPort}`
	} else if (app && app.httpsPort) {
		form.value.proxy = `https://127.0.0.1:${app.httpsPort}`
	} else {
		form.value.proxy = ""
	}
}

const handlePipelineSelect = (val: number) => {
	if (!val) return
	// 流水线里的 exposePort 只作为访问建议值，不应再自动回填为容器代理地址，
	// 否则容易把“外部访问端口”误解成“容器内部监听端口”。
	if (!form.value.proxy) {
		form.value.proxy = "http://127.0.0.1:80"
	}
}

onMounted(() => {
	fetchApps()
	getPipelineList()
	fetchLocalImages()
})

const form = ref({
	id: undefined as number | undefined,
	type: "static",
	primaryDomain: "",
	protocol: "http",
	alias: "",
	otherDomains: "",
	proxy: "",
	IPV6: false,
	appInstallId: undefined as number | undefined,
	remark: "",
	codeSource: "upload",
	gitRepo: "",
	codeDir: "",
	pipelineId: undefined as number | undefined,
	antiCrawler: false,
	antiLeech: false,
	rateLimitMode: "none",
	wafEnable: false,
	blockSensitive: false
})

const rules = {
	type: {
		required: true,
		message: "请选择类型",
		trigger: "blur"
	},
	primaryDomain: {
		required: true,
		message: "请输入主域名",
		trigger: "blur"
	},
	alias: {
		required: true,
		message: "请输入网站目录、代号",
		trigger: "blur"
	},
	proxy: {
		required: true,
		validator(rule: any, value: string) {
			if ((form.value.type === "proxy" || form.value.type === "web_app") && !value) {
				return new Error("请输入代理地址或容器内部端口 (如: 8080)")
			}
			return true
		},
		trigger: "blur"
	},
	appInstallId: {
		required: true,
		validator(rule: any, value: number) {
			if (form.value.codeSource === "app_store" && !value) {
				return new Error("请选择已安装的容器应用")
			}
			return true
		},
		trigger: "blur"
	}
}

const onConfirm = async () => {
	try {
		await formRef.value?.validate()
	} catch (errors) {
		return
	}
	try {
		loading.value = true
		const createPayload: Website.WebSiteCreateReq = {
			type: form.value.type,
			primaryDomain: form.value.primaryDomain,
			alias: form.value.alias,
			otherDomains: form.value.otherDomains,
			proxy: form.value.proxy,
			IPV6: form.value.IPV6,
			appInstallId: form.value.appInstallId,
			remark: form.value.remark,
			codeSource: form.value.codeSource,
			gitRepo: form.value.gitRepo,
			codeDir: form.value.codeDir,
			pipelineId: form.value.pipelineId,
			antiCrawler: form.value.antiCrawler,
			antiLeech: form.value.antiLeech,
			rateLimitMode: form.value.rateLimitMode,
			wafEnable: form.value.wafEnable,
			blockSensitive: form.value.blockSensitive,
		}
		const updatePayload: Website.WebSiteUpdateReq = {
			id: form.value.id || 0,
			primaryDomain: form.value.primaryDomain,
			protocol: form.value.protocol,
			otherDomains: form.value.otherDomains,
			proxy: form.value.proxy,
			pipelineId: form.value.pipelineId,
			codeSource: form.value.codeSource,
			IPV6: form.value.IPV6,
			remark: form.value.remark,
			antiCrawler: form.value.antiCrawler,
			antiLeech: form.value.antiLeech,
			rateLimitMode: form.value.rateLimitMode,
			wafEnable: form.value.wafEnable,
			blockSensitive: form.value.blockSensitive
		}
		let res = await (actionType.value === "add" ? websiteCreateAPI(createPayload) : websiteUpdateAPI(updatePayload))
		emit("confirm", res, loading)
	} catch (error) {
		console.error(error)
	} finally {
		loading.value = false
	}
}

const open = (record?: any, action: string = "add") => {
	visible.value = true
	if (action === "add") {
		actionType.value = "add"
		title.value = "添加网站"
		form.value = {
			id: undefined,
			type: "static",
			primaryDomain: "",
			protocol: "http",
			alias: "",
			otherDomains: "",
			proxy: "",
			IPV6: false,
			appInstallId: undefined,
			remark: "",
			codeSource: "upload",
			gitRepo: "",
			codeDir: "",
			pipelineId: undefined,
			antiCrawler: false,
			antiLeech: false,
			rateLimitMode: "none",
			wafEnable: false,
			blockSensitive: false
		}
	} else {
		actionType.value = "update"
		title.value = "更新网站"
		if (record.domains && record.domains.length > 0) {
			let otherDomain = ""
			for (let i = 0; i < record.domains.length; i++) {
				if (record.domains[i].domain === record.primaryDomain) {
					continue
				}
				if (i == 0) {
					otherDomain += record.domains[i].domain
				} else {
					otherDomain += "\n" + record.domains[i].domain
				}
			}
			record.otherDomains = otherDomain
		} else {
			record.otherDomains = ""
		}
		form.value = {
			...record,
			rateLimitMode: record.rateLimitMode || "none",
			wafEnable: record.wafEnable || false,
			blockSensitive: record.blockSensitive || false
		}
	}
}
watch(
	() => form.value.primaryDomain,
	val => {
		form.value.alias = val || ""
	}
)
watch(
	() => form.value.alias,
	val => {
		if (val.startsWith("http://") || val.startsWith("https://")) {
			form.value.alias = val.replace("http://", "").replace("https://", "")
		}
	}
)
watch(
        () => form.value.type,
        val => {
                if (val === "web_app" && form.value.codeSource === "upload") {
                        form.value.codeSource = "git" // Default to custom image when switching away from static
                }
        }
)
const close = () => {
	visible.value = false
	loading.value = false
}
defineExpose({
	open,
	close
})
</script>
