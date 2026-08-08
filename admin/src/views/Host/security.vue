<template>
  <div class="page-container">
    <div class="page-header flex justify-between items-center mb-6">
      <div class="page-title text-2xl font-bold text-gray-800 dark:text-gray-200">主机安全体检</div>
      <div class="actions">
        <n-button type="primary" :loading="scanning" @click="startScan">
          重新扫描
        </n-button>
      </div>
    </div>

    <n-spin :show="scanning" description="正在进行全方位安全体检，请稍候...">
      <div v-if="hasScanned">
        <n-row :gutter="24">
          <n-col :span="8">
            <n-card class="text-center">
              <n-statistic label="安全得分">
                <div :class="['text-5xl font-bold mt-2', scoreColorClass]">
                  {{ score }}
                </div>
                <div class="text-gray-500 mt-2">满分 100 分</div>
              </n-statistic>
            </n-card>
          </n-col>
          <n-col :span="16">
            <n-card title="扫描结论">
              <div v-if="riskCount > 0">
                <n-alert type="error" title="发现安全风险" :show-icon="true">
                  您的服务器目前存在 {{ riskCount }} 个安全隐患，建议立即修复。
                </n-alert>
                <div class="mt-4">
                  <n-button type="error" :loading="fixing" @click="handleFixAll">一键修复所有风险</n-button>
                </div>
              </div>
              <div v-else>
                <n-alert type="success" title="服务器状态优秀" :show-icon="true">
                  您的服务器目前未发现明显的安全隐患，请继续保持！
                </n-alert>
              </div>
            </n-card>
          </n-col>
        </n-row>

        <n-divider />

        <div class="risk-items mt-6">
          <h3 class="text-lg font-bold mb-4">检查项详情</h3>
          
          <n-card class="mb-4" size="small">
            <template #header>
              <div class="flex items-center">
                <div :class="['w-3 h-3 rounded-full mr-2', sshRisks.length > 0 ? 'bg-red-500' : 'bg-green-500']"></div>
                SSH 登录防爆破扫描
              </div>
            </template>
            <template #header-extra>
              <n-button v-if="sshRisks.length > 0" size="small" type="error" :loading="fixingSSH" @click="handleFixSSH">一键修复</n-button>
              <n-tag v-else type="success">安全</n-tag>
            </template>
            
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-2">
              SSH 是黑客入侵的首选入口，绝大多数被黑的机器都是因为密码太弱或默认配置未改。
            </div>
            
            <div v-if="sshRisks.length > 0">
              <ul class="list-disc pl-5 mt-2 text-red-600 dark:text-red-400">
                <li v-for="risk in sshRisks" :key="risk">{{ risk }}</li>
              </ul>
              <div class="mt-2 p-2 bg-red-50 dark:bg-red-900/20 rounded text-red-800 dark:text-red-200 text-sm">
                <strong>大白话风险：</strong>您的服务器目前很容易被黑客通过暴力破解密码的方式入侵！一键修复将自动生成一个随机高位端口，并禁用 Root 密码登录（强烈建议提前配置好 SSH 密钥）。
              </div>
            </div>
            <div v-else class="text-green-600 dark:text-green-400">
              ✓ SSH 配置安全，未使用默认端口且已禁用高危密码登录。
            </div>
          </n-card>

          <n-card class="mb-4" size="small">
            <template #header>
              <div class="flex items-center">
                <div :class="['w-3 h-3 rounded-full mr-2', portRisks.length > 0 ? 'bg-red-500' : 'bg-green-500']"></div>
                端口与网络暴露扫描
              </div>
            </template>
            <template #header-extra>
              <n-tag v-if="portRisks.length > 0" type="error">高危</n-tag>
              <n-tag v-else type="success">安全</n-tag>
            </template>
            
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-2">
              检测系统当前监听的端口，防止内部服务（如数据库）裸奔在公网。
            </div>
            
            <div v-if="portRisks.length > 0">
              <ul class="list-disc pl-5 mt-2 text-red-600 dark:text-red-400">
                <li>发现以下数据库端口对全网 (0.0.0.0) 开放：<strong>{{ portRisks.join(", ") }}</strong></li>
              </ul>
              <div class="mt-2 p-2 bg-red-50 dark:bg-red-900/20 rounded text-red-800 dark:text-red-200 text-sm">
                <strong>大白话风险：</strong>您的内部服务正在向所有公网 IP 开放，极易被勒索病毒扫描并加密数据！建议前往防火墙模块拦截这些端口，或修改应用配置仅允许 127.0.0.1 访问。
              </div>
            </div>
            <div v-else class="text-green-600 dark:text-green-400">
              ✓ 未发现常见的数据库服务（如 3306, 6379 等）裸奔在外网。
            </div>
          </n-card>

        </div>
      </div>
      
      <div v-else class="py-20 text-center">
        <div class="text-gray-500 mb-4">您还未进行过安全体检</div>
        <n-button type="primary" size="large" @click="startScan">立即开始全面体检</n-button>
      </div>
    </n-spin>

    <n-modal v-model:show="showFixResultModal" preset="dialog" title="修复完成" type="success">
      <template #default>
        <div class="whitespace-pre-wrap rounded-md bg-slate-50 p-3 font-mono text-sm text-slate-700">
          {{ fixResultMessage }}
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useMessage, useDialog } from 'naive-ui'
import http from "@/api"

const message = useMessage()
const dialog = useDialog()

const scanning = ref(false)
const hasScanned = ref(false)
const fixing = ref(false)
const fixingSSH = ref(false)

const scanResult = ref<any>(null)
const showFixResultModal = ref(false)
const fixResultMessage = ref("")

const sshRisks = computed(() => {
  if (!scanResult.value || !scanResult.value.ssh) return []
  const risks = []
  const ssh = scanResult.value.ssh
  if (ssh.port == "22" || ssh.port == 22) {
    risks.push("正在使用默认的 22 端口，容易被针对性扫描")
  }
  if (ssh.permitRootLogin === "yes") {
    risks.push("允许 root 用户直接密码登录")
  }
  if (ssh.passwordAuthentication === "yes") {
    risks.push("允许密码认证（建议改用更安全的 SSH 密钥认证）")
  }
  return risks
})

const portRisks = computed(() => {
  if (!scanResult.value || !scanResult.value.port || !scanResult.value.port.exposed) return []
  return scanResult.value.port.exposed
})

const riskCount = computed(() => {
  let count = 0
  if (sshRisks.value.length > 0) count++
  if (portRisks.value.length > 0) count++
  return count
})

const score = computed(() => {
  if (!hasScanned.value) return 100
  let s = 100
  s -= sshRisks.value.length * 15
  s -= portRisks.value.length * 20
  return s < 0 ? 0 : s
})

const scoreColorClass = computed(() => {
  if (score.value >= 90) return 'text-green-500'
  if (score.value >= 60) return 'text-orange-500'
  return 'text-red-500'
})

const startScan = async () => {
  scanning.value = true
  try {
    const res = await http.get<any>("/security/scan")
    if (res.code === 0) {
      scanResult.value = res.data
      hasScanned.value = true
      message.success("体检完成")
    } else {
      message.error(res.msg || "扫描失败")
    }
  } catch (error: any) {
    // 错误提示由请求拦截器统一处理
  } finally {
    scanning.value = false
  }
}

const handleFixSSH = () => {
  dialog.warning({
    title: "高危操作确认",
    content: "一键修复将自动为您生成一个随机的 SSH 高位端口，并禁用 Root 密码登录。修复后请务必记下新端口并使用密钥登录，否则您将可能无法连接服务器！确认要继续吗？",
    positiveText: "我已了解风险并确认修复",
    negativeText: "取消",
    onPositiveClick: async () => {
      fixingSSH.value = true
      try {
        const res = await http.post<any>("/security/fix/ssh")
        if (res.code === 0) {
          fixResultMessage.value = `SSH 修复成功！\n\n新端口：${res.data.newPort}\n请牢记此端口，并在您的云服务商防火墙（安全组）中放行该端口。`
          showFixResultModal.value = true
          // 重新扫描以更新状态
          await startScan()
        } else {
          message.error(res.msg || "修复失败")
        }
      } catch (error: any) {
        // 错误提示由请求拦截器统一处理
      } finally {
        fixingSSH.value = false
      }
    }
  })
}

const handleFixAll = () => {
  if (sshRisks.value.length > 0) {
    handleFixSSH()
  } else {
    message.info("暂无自动化修复脚本，请根据提示手动处理风险项。")
  }
}

onMounted(() => {
  // 可选：页面加载时自动扫描
  // startScan()
})
</script>

<style scoped>
.page-container {
  padding: 24px;
}
</style>
