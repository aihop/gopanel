<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <n-drawer
    v-model:show="visible"
    :width="520"
    :mask-closable="false"
  >
    <n-drawer-content closable>
      <template #header>
        <div class="flex items-center gap-3">
          <div class="text-base font-semibold">安全防护</div>
          <n-tag
            v-if="website.primaryDomain"
            round
            :bordered="false"
            type="primary"
          >
            {{ website.primaryDomain }}
          </n-tag>
        </div>
      </template>

      <div class="space-y-6">
        <div class="rounded-3xl border border-slate-200 bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 px-5 py-5 text-white">
          <div class="flex items-start justify-between gap-4">
            <div>
              <div class="text-xs uppercase tracking-[0.18em] text-slate-300">Website Security Center</div>
              <div class="mt-2 text-3xl font-semibold">{{ securityScore }}</div>
              <div class="mt-2 text-sm text-slate-300">{{ securitySummaryText }}</div>
              <div
                v-if="bindingRuntimeText"
                class="mt-2 text-xs text-slate-300"
              >
                {{ bindingRuntimeText }}
              </div>
            </div>
            <div class="rounded-2xl bg-white/10 px-4 py-3 text-right">
              <div class="text-xs text-slate-300">主机风险</div>
              <div class="mt-1 text-lg font-semibold">{{ hostRiskCount }}</div>
            </div>
          </div>
        </div>

        <n-alert
          type="info"
          :show-icon="true"
        >
          这里用于集中调整站点的访问防护策略。添加/修改网站时只保留轻量预设，细粒度安全能力统一在这里维护。
        </n-alert>

        <div class="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4">
          <div class="mb-3 flex items-center justify-between">
            <div class="text-sm font-semibold text-slate-700">快速策略</div>
            <div class="text-xs text-slate-500">先用预设，再按需微调</div>
          </div>
          <div class="flex flex-wrap gap-2">
            <n-button
              ghost
              @click="applyPreset('off')"
            >
              关闭防护
            </n-button>
            <n-button
              type="primary"
              ghost
              @click="applyPreset('recommended')"
            >
              推荐策略
            </n-button>
            <n-button
              type="warning"
              ghost
              @click="applyPreset('strict')"
            >
              严格策略
            </n-button>
          </div>
        </div>

        <div class="rounded-2xl border border-slate-200 bg-white px-4 py-4">
          <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-blue-600">Website Shield</div>
          <div class="mb-4 text-sm font-semibold text-slate-700">网站防护</div>
          <div class="space-y-4">
            <n-form-item label="防爬虫">
              <div class="w-full flex flex-col gap-2">
                <n-switch v-model:value="form.antiCrawler" />
                <div class="text-xs text-slate-500">拦截常见恶意扫描器、无头浏览器和异常 User-Agent。</div>
              </div>
            </n-form-item>

            <n-form-item label="防盗链">
              <div class="w-full flex flex-col gap-2">
                <n-switch v-model:value="form.antiLeech" />
                <div class="text-xs text-slate-500">限制图片、视频等静态资源被外站直接引用。</div>
              </div>
            </n-form-item>

            <n-form-item label="频率限制">
              <div class="w-full flex flex-col gap-2">
                <n-select
                  v-model:value="form.rateLimitMode"
                  :options="rateLimitOptions"
                />
                <div class="text-xs text-slate-500">用于缓解 CC 和高频恶意请求，严格模式适合临时应急。</div>
              </div>
            </n-form-item>

            <n-form-item label="轻量 WAF">
              <div class="w-full flex flex-col gap-2">
                <n-switch v-model:value="form.wafEnable" />
                <div class="text-xs text-slate-500">拦截常见 SQL 注入、XSS 与路径穿越特征。</div>
              </div>
            </n-form-item>

            <n-form-item label="敏感文件保护">
              <div class="w-full flex flex-col gap-2">
                <n-switch v-model:value="form.blockSensitive" />
                <div class="text-xs text-slate-500">禁止访问 `.env`、`.git`、`.sql`、`.bak` 等敏感文件。</div>
              </div>
            </n-form-item>

            <n-form-item label="安全响应头">
              <div class="w-full flex flex-col gap-2">
                <n-switch v-model:value="form.securityHeader" />
                <div class="text-xs text-slate-500">启用 `X-Frame-Options`、`X-Content-Type-Options`、`Referrer-Policy` 等基础安全头。</div>
              </div>
            </n-form-item>

            <n-form-item label="HSTS">
              <div class="w-full flex flex-col gap-2">
                <n-switch
                  v-model:value="form.hstsEnabled"
                  :disabled="!isHttpsWebsite"
                />
                <div class="text-xs text-slate-500">
                  仅 HTTPS 站点可开启，强制浏览器后续优先使用 HTTPS。
                </div>
              </div>
            </n-form-item>

            <n-form-item label="IP 白名单">
              <div class="w-full flex flex-col gap-2">
                <n-input
                  v-model:value="form.ipAllowlist"
                  type="textarea"
                  placeholder="一行一个 IP / CIDR，例如 1.2.3.4 或 10.0.0.0/24"
                />
                <div class="text-xs text-slate-500">填写后仅允许列表中的 IP 访问该网站。</div>
              </div>
            </n-form-item>

            <n-form-item label="IP 黑名单">
              <div class="w-full flex flex-col gap-2">
                <n-input
                  v-model:value="form.ipBlocklist"
                  type="textarea"
                  placeholder="一行一个 IP / CIDR，例如 8.8.8.8"
                />
                <div class="text-xs text-slate-500">黑名单中的 IP 将直接被网站层拦截。</div>
              </div>
            </n-form-item>
          </div>
        </div>

        <div class="rounded-2xl border border-emerald-100 bg-emerald-50 px-4 py-4">
          <div class="mb-2 text-sm font-semibold text-emerald-700">防护摘要</div>
          <div class="flex flex-wrap gap-2">
            <n-tag
              v-for="item in enabledSummary"
              :key="item"
              round
              :bordered="false"
              type="success"
            >
              {{ item }}
            </n-tag>
            <span
              v-if="enabledSummary.length === 0"
              class="text-xs text-slate-500"
            >
              当前未启用任何网站级防护
            </span>
          </div>
        </div>

        <div class="rounded-2xl border border-slate-200 bg-white px-4 py-4">
          <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-amber-600">Host Signals</div>
          <div class="mb-3 flex items-center justify-between">
            <div class="text-sm font-semibold text-slate-700">主机联动</div>
            <n-button
              text
              type="primary"
              :loading="securityLoading"
              @click="loadSecuritySignals"
            >
              刷新
            </n-button>
          </div>
          <div class="grid gap-3 md:grid-cols-2">
            <div class="rounded-2xl border border-slate-100 bg-slate-50 px-4 py-3">
              <div class="text-xs text-slate-500">SSH 风险</div>
              <div class="mt-2 text-sm text-slate-700">
                端口：{{ sshInfo.port || "-" }}，Root 登录：{{ sshInfo.permitRootLogin || "-" }}，密码登录：{{ sshInfo.passwordAuthentication || "-" }}
              </div>
            </div>
            <div class="rounded-2xl border border-slate-100 bg-slate-50 px-4 py-3">
              <div class="text-xs text-slate-500">暴露高危端口</div>
              <div class="mt-2 text-sm text-slate-700">
                {{ exposedPortsText }}
              </div>
            </div>
          </div>
          <div class="mt-5 rounded-2xl border border-slate-100 bg-slate-50 px-4 py-4">
            <div class="mb-3 flex items-center justify-between">
              <div class="text-sm font-semibold text-slate-700">近期 SSH 异常</div>
              <div class="text-xs text-slate-500">作为主机侧异常信号展示，不属于网站规则本体</div>
            </div>
            <div class="space-y-3">
              <div
                v-for="item in sshAlerts"
                :key="`${item.createdAt}-${item.sourceIp}-${item.username}`"
                class="rounded-2xl border border-rose-100 bg-rose-50 px-4 py-3"
              >
                <div class="flex items-center justify-between gap-3">
                  <div class="text-sm font-medium text-rose-700">
                    {{ item.sourceIp }} · {{ item.username || "未知用户" }}
                  </div>
                  <div class="flex items-center gap-3">
                    <n-tag
                      v-if="isIPBlocked(item.sourceIp)"
                      size="small"
                      round
                      :bordered="false"
                      type="warning"
                    >
                      已封禁
                    </n-tag>
                    <n-button
                      size="small"
                      tertiary
                      :type="isIPBlocked(item.sourceIp) ? 'warning' : 'error'"
                      :loading="operatingIP === item.sourceIp"
                      @click="toggleIPBlock(item)"
                    >
                      {{ isIPBlocked(item.sourceIp) ? "一键解封" : "一键封禁" }}
                    </n-button>
                    <div class="text-xs text-slate-500">{{ formatTime(item.createdAt) }}</div>
                  </div>
                </div>
                <div class="mt-1 text-xs text-slate-500">
                  {{ item.authMethod || "unknown" }} / {{ item.message }}
                </div>
              </div>
              <div
                v-if="!sshAlerts.length"
                class="text-sm text-slate-400"
              >
                暂未发现近期 SSH 失败登录记录
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border border-slate-200 bg-white px-4 py-4">
          <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-emerald-600">Block Records</div>
          <div class="mb-3 flex items-center justify-between">
            <div class="text-sm font-semibold text-slate-700">封禁记录</div>
            <div class="text-xs text-slate-500">当前主机防火墙中的 IP 丢弃规则</div>
          </div>
          <div class="space-y-3">
            <div
              v-for="item in blockedRuleItems"
              :key="`${item.address}-${item.strategy}-${item.description}`"
              class="rounded-2xl border border-amber-100 bg-amber-50 px-4 py-3"
            >
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="text-sm font-medium text-amber-700">
                    {{ item.address }}
                  </div>
                  <div class="mt-1 text-xs text-slate-500">
                    {{ item.description || "无备注" }}
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <n-button
                    size="small"
                    tertiary
                    type="warning"
                    :loading="operatingIP === item.address"
                    @click="handleUnblockIP(item.address)"
                  >
                    一键解封
                  </n-button>
                  <n-tag
                    size="small"
                    round
                    :bordered="false"
                    type="warning"
                  >
                    drop
                  </n-tag>
                </div>
              </div>
            </div>
            <div
              v-if="!blockedRuleItems.length"
              class="text-sm text-slate-400"
            >
              当前没有主机级封禁记录
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <n-button @click="close">
            取消
          </n-button>
          <n-button
            type="primary"
            :loading="loading"
            @click="handleSave"
          >
            保存防护
          </n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import type { Website } from "@/api/interface/website"
import { computed, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { websiteUpdateAPI } from "@/api/modules/website"
import { ListAppInstalled } from "@/api/modules/apps"
import { getSSHLoginLogs } from "@/api/modules/log"
import { buildRuntimeDetailText } from "@/utils/runtime"
import { listAllPipelines } from "@/utils/pipeline"
import { hasWebsiteRuntimeMeta, resolveWebsiteBindingMeta } from "@/utils/websiteRuntime"
import http from "@/api"
import { formatTime } from "@/utils/date"
import { Log } from "@/api/interface/log"
import { operateIPRule, searchFirewallRules } from "@/api/modules/host"
import type { Host } from "@/api/interface/host"

type SecurityPreset = "off" | "recommended" | "strict"

const emit = defineEmits(["confirm"])
const message = useMessage()
const dialog = useDialog()

const visible = ref(false)
const loading = ref(false)

const website = ref<Website.WebsiteDTO>({} as Website.WebsiteDTO)
const form = ref({
  antiCrawler: false,
  antiLeech: false,
  rateLimitMode: "none",
  wafEnable: false,
  blockSensitive: false,
  ipAllowlist: "",
  ipBlocklist: "",
  securityHeader: false,
  hstsEnabled: false,
})

const securityLoading = ref(false)
const sshInfo = ref<Record<string, any>>({})
const exposedPorts = ref<string[]>([])
const sshAlerts = ref<Log.SSHLoginLog[]>([])
const operatingIP = ref("")
const blockedIPs = ref<string[]>([])
const blockedRuleItems = ref<Host.RuleInfo[]>([])
const appInstallMap = ref<Record<number, any>>({})
const pipelineMap = ref<Record<number, any>>({})

const rateLimitOptions = [
  { label: "关闭", value: "none" },
  { label: "常规防护", value: "normal" },
  { label: "严格防护", value: "strict" },
]

const enabledSummary = computed(() => {
  const list: string[] = []
  if (form.value.antiCrawler)
    list.push("防爬虫")
  if (form.value.antiLeech)
    list.push("防盗链")
  if (form.value.rateLimitMode !== "none")
    list.push(form.value.rateLimitMode === "strict" ? "严格限流" : "常规限流")
  if (form.value.wafEnable)
    list.push("轻量 WAF")
  if (form.value.blockSensitive)
    list.push("敏感文件保护")
  if (form.value.securityHeader)
    list.push("安全响应头")
  if (form.value.hstsEnabled)
    list.push("HSTS")
  if (normalizeList(form.value.ipAllowlist).length)
    list.push("IP 白名单")
  if (normalizeList(form.value.ipBlocklist).length)
    list.push("IP 黑名单")
  return list
})

const isHttpsWebsite = computed(() => {
  return String(website.value.protocol || "").toLowerCase() === "https"
})

const hostRiskCount = computed(() => {
  let count = 0
  if (String(sshInfo.value.permitRootLogin || "").toLowerCase() !== "no")
    count++
  if (String(sshInfo.value.passwordAuthentication || "").toLowerCase() !== "no")
    count++
  count += exposedPorts.value.length
  return count
})

const securityScore = computed(() => {
  let score = 100
  if (!form.value.antiCrawler) score -= 8
  if (!form.value.antiLeech) score -= 6
  if (form.value.rateLimitMode === "none") score -= 12
  if (!form.value.wafEnable) score -= 18
  if (!form.value.blockSensitive) score -= 12
  if (!form.value.securityHeader) score -= 8
  if (isHttpsWebsite.value && !form.value.hstsEnabled) score -= 6
  if (hostRiskCount.value > 0) score -= Math.min(hostRiskCount.value * 6, 24)
  return Math.max(score, 20)
})

const securitySummaryText = computed(() => {
  if (securityScore.value >= 90)
    return "当前安全状态优秀，站点防护较完整。"
  if (securityScore.value >= 70)
    return "当前安全状态良好，仍建议继续收紧风险项。"
  return "当前存在明显风险，建议尽快启用推荐策略并处理主机风险。"
})

const exposedPortsText = computed(() => {
  return exposedPorts.value.length ? exposedPorts.value.join("、") : "未发现高危端口暴露"
})

const bindingRuntimeText = computed(() => {
  if (!website.value) return ""
  return resolveWebsiteBindingMeta(website.value, {
    appInstallMap: appInstallMap.value,
    pipelineMap: pipelineMap.value
  }, {
    sourcePrefix: "绑定目标：",
    includeSourceInDetail: true,
    kindFallback: "Runtime",
    userFallback: "镜像默认",
    runUserPrefix: "运行用户："
  })?.detail || ""
})

function normalizeIP(value: string) {
  return String(value || "").trim().toLowerCase()
}

function isIPBlocked(value: string) {
  const target = normalizeIP(value)
  return !!target && blockedIPs.value.includes(target)
}

function applyPreset(preset: SecurityPreset) {
  if (preset === "off") {
    form.value = {
      antiCrawler: false,
      antiLeech: false,
      rateLimitMode: "none",
      wafEnable: false,
      blockSensitive: false,
      ipAllowlist: "",
      ipBlocklist: "",
      securityHeader: false,
      hstsEnabled: false,
    }
    return
  }

  if (preset === "strict") {
    form.value = {
      antiCrawler: true,
      antiLeech: true,
      rateLimitMode: "strict",
      wafEnable: true,
      blockSensitive: true,
      ipAllowlist: form.value.ipAllowlist,
      ipBlocklist: form.value.ipBlocklist,
      securityHeader: true,
      hstsEnabled: isHttpsWebsite.value,
    }
    return
  }

  form.value = {
    antiCrawler: true,
    antiLeech: true,
    rateLimitMode: "normal",
    wafEnable: true,
    blockSensitive: true,
    ipAllowlist: form.value.ipAllowlist,
    ipBlocklist: form.value.ipBlocklist,
    securityHeader: true,
    hstsEnabled: isHttpsWebsite.value,
  }
}

function normalizeList(value: string) {
  return String(value || "")
    .split(/\r?\n|,|;/)
    .map(item => item.trim())
    .filter(Boolean)
}

async function loadSecuritySignals() {
  try {
    securityLoading.value = true
    const [scanRes, sshRes, firewallRes] = await Promise.all([
      http.get<any>("/security/scan"),
      getSSHLoginLogs({ page: 1, limit: 5, ip: "", status: "Failed", username: "" }),
      searchFirewallRules({ page: 1, limit: 500, info: "", status: "all", strategy: "drop", type: "ip" }),
    ])
    sshInfo.value = scanRes.data?.ssh || {}
    exposedPorts.value = scanRes.data?.port?.exposed || []
    sshAlerts.value = sshRes.data?.items || []
    blockedRuleItems.value = (firewallRes.data?.items || []) as Host.RuleInfo[]
    blockedIPs.value = blockedRuleItems.value
      .map(item => normalizeIP(item.address))
      .filter(Boolean)
  } catch (error) {
    console.error(error)
  } finally {
    securityLoading.value = false
  }
}

async function loadBindingMeta() {
  try {
    if (website.value.appInstallId) {
      const res = await ListAppInstalled()
      const list = Array.isArray(res.data) ? res.data : []
      const nextMap: Record<number, any> = {}
      for (const item of list) nextMap[item.id] = item
      appInstallMap.value = nextMap
    }
    if (website.value.pipelineId) {
      const list = await listAllPipelines()
      const nextMap: Record<number, any> = {}
      for (const item of list) nextMap[item.id] = item
      pipelineMap.value = nextMap
    }
  } catch (error) {
    appInstallMap.value = {}
    pipelineMap.value = {}
  }
}

function toggleIPBlock(item: Log.SSHLoginLog) {
  const ip = String(item.sourceIp || "").trim()
  if (!ip) {
    message.warning("当前记录缺少来源 IP，无法封禁")
    return
  }
  const blocked = isIPBlocked(ip)
  dialog.warning({
    title: blocked ? "确认解封该来源 IP？" : "确认封禁该来源 IP？",
    content: blocked
      ? `将从主机防火墙移除 ${ip} 的封禁规则，该 IP 可重新访问服务器。`
      : `将通过主机防火墙立即封禁 ${ip}，该 IP 将无法继续访问服务器。`,
    positiveText: blocked ? "确认解封" : "确认封禁",
    negativeText: "取消",
    onPositiveClick: async () => {
      operatingIP.value = ip
      try {
        const payload: Host.RuleIP = {
          operation: blocked ? "remove" : "add",
          address: ip,
          strategy: "drop",
          description: `网站安全中心封禁：SSH 异常登录（${website.value.primaryDomain || "unknown"}）`,
        }
        await operateIPRule(payload)
        const normalized = normalizeIP(ip)
        if (blocked) {
          blockedIPs.value = blockedIPs.value.filter(item => item !== normalized)
          blockedRuleItems.value = blockedRuleItems.value.filter(item => normalizeIP(item.address) !== normalized)
          message.success(`已解封 ${ip}`)
        } else {
          if (!blockedIPs.value.includes(normalized))
            blockedIPs.value.push(normalized)
          blockedRuleItems.value.unshift({
            address: ip,
            strategy: "drop",
            description: `网站安全中心封禁：SSH 异常登录（${website.value.primaryDomain || "unknown"}）`,
          } as Host.RuleInfo)
          message.success(`已封禁 ${ip}`)
        }
      } finally {
        operatingIP.value = ""
      }
    },
  })
}

function handleUnblockIP(ip: string) {
  const normalized = normalizeIP(ip)
  if (!normalized) {
    message.warning("无效的 IP 地址")
    return
  }
  dialog.warning({
    title: "确认解封该 IP？",
    content: `将从主机防火墙移除 ${ip} 的封禁规则。`,
    positiveText: "确认解封",
    negativeText: "取消",
    onPositiveClick: async () => {
      operatingIP.value = ip
      try {
        await operateIPRule({
          operation: "remove",
          address: ip,
          strategy: "drop",
          description: "",
        })
        blockedIPs.value = blockedIPs.value.filter(item => item !== normalized)
        blockedRuleItems.value = blockedRuleItems.value.filter(item => normalizeIP(item.address) !== normalized)
        message.success(`已解封 ${ip}`)
      } finally {
        operatingIP.value = ""
      }
    },
  })
}

function buildOtherDomains(row: Website.WebsiteDTO) {
  if (Array.isArray(row.domains)) {
    return row.domains
      .map((item: any) => (typeof item === "string" ? item : item?.domain))
      .filter((item: string) => item && item !== row.primaryDomain)
      .join("\n")
  }
  return row.otherDomains || ""
}

async function handleSave() {
  try {
    loading.value = true
    await websiteUpdateAPI({
      id: website.value.id,
      primaryDomain: website.value.primaryDomain,
      protocol: website.value.protocol,
      otherDomains: buildOtherDomains(website.value),
      proxy: website.value.proxy || "",
      pipelineId: website.value.pipelineId,
      codeSource: website.value.codeSource,
      IPV6: !!website.value.IPV6,
      remark: website.value.remark || "",
      antiCrawler: form.value.antiCrawler,
      antiLeech: form.value.antiLeech,
      rateLimitMode: form.value.rateLimitMode,
      wafEnable: form.value.wafEnable,
      blockSensitive: form.value.blockSensitive,
      ipAllowlist: normalizeList(form.value.ipAllowlist).join("\n"),
      ipBlocklist: normalizeList(form.value.ipBlocklist).join("\n"),
      securityHeader: form.value.securityHeader,
      hstsEnabled: isHttpsWebsite.value ? form.value.hstsEnabled : false,
    })
    message.success("安全防护已保存")
    emit("confirm")
    close()
  } finally {
    loading.value = false
  }
}

function open(record: Website.WebsiteDTO) {
  website.value = { ...record }
  form.value = {
    antiCrawler: !!record.antiCrawler,
    antiLeech: !!record.antiLeech,
    rateLimitMode: record.rateLimitMode || "none",
    wafEnable: !!record.wafEnable,
    blockSensitive: !!record.blockSensitive,
    ipAllowlist: record.ipAllowlist || "",
    ipBlocklist: record.ipBlocklist || "",
    securityHeader: !!record.securityHeader,
    hstsEnabled: !!record.hstsEnabled,
  }
  visible.value = true
  if (!hasWebsiteRuntimeMeta(record)) {
    loadBindingMeta()
  }
  loadSecuritySignals()
}

function close() {
  visible.value = false
}

defineExpose({
  open,
  close,
})
</script>
