import type { SelectOption } from "naive-ui"
import {
  ApplySSL,
  CreateSSL,
  DeleteSSL,
  GetSSL,
  ObtainSSL,
  PushToCDNAPI,
  RenewSSL,
  SSLSearchAPI
} from "@/api/modules/ssl"
import { websiteListAPI } from "@/api/modules/website"
import { CloudCdnDomainsAPI, cloudAccountListAPI } from "@/api/modules/cloud"
import { useDialog, useMessage } from "naive-ui"
import { computed, onMounted, reactive, ref } from "vue"
import { createPushRuleColumns, createSSLColumns } from "./sslColumns"
import {
  buildWebsiteRuntimeText,
  createDefaultApplyForm,
  createDefaultCloudApplyForm,
  createDefaultPushCDNForm,
  createDefaultSyncForm,
  createDefaultUploadForm,
  downloadContent,
  normalizeSSLRow,
  type SSLRow,
  type WebsiteOption
} from "./sslHelpers"
import { useSSLLogStream } from "./useSSLLogStream"
import { useSSLPushRules } from "./useSSLPushRules"
export const useSSLManagement = () => {
  const message = useMessage()
  const dialog = useDialog()
  const loading = ref(false)
  const submitting = ref(false)
  const cdnDomainsLoading = ref(false)
  const tableData = ref<SSLRow[]>([])
  const websites = ref<WebsiteOption[]>([])
  const cloudAccounts = ref<Array<{ id: number; name: string; type: string }>>([])
  const cdnDomainsOptions = ref<Array<{ label: string; value: string }>>([])
  const syncModalVisible = ref(false)
  const uploadModalVisible = ref(false)
  const detailModalVisible = ref(false)
  const applyModalVisible = ref(false)
  const pushCDNModalVisible = ref(false)
  const cloudApplyModalVisible = ref(false)
  const pushRuleModalVisible = ref(false)
  const currentSSL = ref<SSLRow | null>(null)
  const currentApplySSLId = ref<number | null>(null)
  const currentPushSSL = ref<SSLRow | null>(null)
  const cloudApplyForm = reactive(createDefaultCloudApplyForm())
  const pushCDNForm = reactive(createDefaultPushCDNForm())
  const syncForm = reactive(createDefaultSyncForm())
  const applyForm = reactive(createDefaultApplyForm())
  const uploadForm = reactive(createDefaultUploadForm())
  const {
    pushRuleLoading,
    pushRuleData,
    pushRuleForm,
    fetchPushRules,
    resetPushRuleForm,
    editPushRule,
    deletePushRule,
    handleSavePushRule
  } = useSSLPushRules({
    currentPushSSL,
    submitting
  })
  const websiteMap = computed<Record<number, WebsiteOption>>(() =>
    websites.value.reduce<Record<number, WebsiteOption>>((acc, item) => {
      acc[item.id] = item
      return acc
    }, {})
  )
  const websiteOptions = computed<SelectOption[]>(() =>
    websites.value.map(item => ({
      label: item.primaryDomain || `#${item.id}`,
      value: item.id
    }))
  )
  const cloudAccountOptions = computed<SelectOption[]>(() =>
    cloudAccounts.value.map(item => ({
      label: `${item.name} (${item.type})`,
      value: item.id
    }))
  )
  const selectedSyncWebsiteRuntimeText = computed(() => {
    if (!syncForm.websiteId) return ""
    return buildWebsiteRuntimeText(websiteMap.value[syncForm.websiteId])
  })
  const selectedApplyWebsiteRuntimeText = computed(() => {
    if (!applyForm.websiteId) return ""
    return buildWebsiteRuntimeText(websiteMap.value[applyForm.websiteId])
  })
  const detailTitle = computed(() => (currentSSL.value ? `${currentSSL.value.primaryDomain} 证书详情` : "证书详情"))
  const buildBoundWebsiteRuntimeText = (item: { id: number; name?: string; primaryDomain?: string }) =>
    buildWebsiteRuntimeText(websiteMap.value[item.id] || item)
  const getCloudAccountLabel = (cloudAccountId: number) => {
    const account = cloudAccounts.value.find(item => item.id === cloudAccountId)
    return account ? `${account.name} (${account.type})` : `未知账号 ID: ${cloudAccountId}`
  }
  async function handleCdnAccountChange(value: number | null) {
    cloudApplyForm.cdnAccountId = value
    cloudApplyForm.primaryDomain = ""
    cdnDomainsOptions.value = []
    if (!value) return
    cdnDomainsLoading.value = true
    try {
      const res = await CloudCdnDomainsAPI(value)
      cdnDomainsOptions.value = (res.data || []).map(domain => ({ label: domain, value: domain }))
      if (!cdnDomainsOptions.value.length) {
        message.info("该云账号下暂无CDN域名")
      }
    } catch (error: any) {
      void 0
    } finally {
      cdnDomainsLoading.value = false
    }
  }
  const { logsData, logModalVisible, openLogModal, handleLogModalChange } = useSSLLogStream({
    onFinished: fetchData
  })
  const columns = createSSLColumns({
    buildBoundWebsiteRuntimeText,
    openDetail,
    openPushCDNModal,
    openPushRuleModal,
    openLogModal,
    handleRenewCertificate,
    downloadContent,
    openApplyModal,
    confirmDelete
  })
  const pushRuleColumns = createPushRuleColumns({
    getCloudAccountLabel,
    onEdit: editPushRule,
    onDelete: deletePushRule
  })
  async function fetchWebsites() {
    const res = await websiteListAPI()
    websites.value = Array.isArray(res.data?.items) ? res.data.items : []
  }
  async function fetchCloudAccounts() {
    const res = await cloudAccountListAPI({ page: 1, limit: 100 } as any)
    cloudAccounts.value = Array.isArray(res.data?.items) ? res.data.items : []
  }
  async function fetchData() {
    loading.value = true
    try {
      const res = await SSLSearchAPI({ page: 1, limit: 200, wheres: [] } as any)
      tableData.value = Array.isArray(res.data) ? res.data.map(normalizeSSLRow) : []
    } finally {
      loading.value = false
    }
  }
  function openCloudApplyModal() {
    Object.assign(cloudApplyForm, createDefaultCloudApplyForm())
    cdnDomainsOptions.value = []
    cloudApplyModalVisible.value = true
  }
  function openSyncModal() {
    Object.assign(syncForm, createDefaultSyncForm())
    syncModalVisible.value = true
  }
  function openUploadModal() {
    Object.assign(uploadForm, createDefaultUploadForm())
    uploadModalVisible.value = true
  }
  function openApplyModal(sslId: number) {
    currentApplySSLId.value = sslId
    Object.assign(applyForm, createDefaultApplyForm())
    applyModalVisible.value = true
  }
  function openPushCDNModal(row: SSLRow) {
    currentPushSSL.value = row
    pushCDNForm.cloudAccountId = row.cloudAccountId || row.dnsAccountId || null
    pushCDNForm.targetDomain = row.primaryDomain
    pushCDNModalVisible.value = true
  }
  function openPushRuleModal(row: SSLRow) {
    currentPushSSL.value = row
    resetPushRuleForm()
    pushRuleModalVisible.value = true
    void fetchPushRules()
  }
  async function openDetail(id: number) {
    const res = await GetSSL(id)
    currentSSL.value = normalizeSSLRow(res.data)
    detailModalVisible.value = true
  }
  async function handleSyncCertificate() {
    if (!syncForm.websiteId) {
      message.warning("请选择网站")
      return
    }
    submitting.value = true
    try {
      await ObtainSSL({ ID: syncForm.websiteId })
      message.success("已同步 Caddy 自动证书")
      syncModalVisible.value = false
      await fetchData()
    } finally {
      submitting.value = false
    }
  }
  async function handleRenewCertificate(row: SSLRow) {
    dialog.info({
      title: "确认重新签发证书？",
      content: `将立刻为域名 ${row.primaryDomain} 重新发起签发流程。`,
      positiveText: "确认",
      negativeText: "取消",
      onPositiveClick: async () => {
        loading.value = true
        try {
          await RenewSSL({ id: row.id })
          message.success("已提交重签请求")
          openLogModal(row.id)
        } catch (error: any) {
          void 0
        } finally {
          loading.value = false
        }
      }
    })
  }
  async function handleCloudApplyCertificate() {
    if (!cloudApplyForm.primaryDomain || !cloudApplyForm.cloudAccountId) {
      message.warning("请填写主域名并选择云账号")
      return
    }
    submitting.value = true
    try {
      const res = await CreateSSL({
        primaryDomain: cloudApplyForm.primaryDomain,
        otherDomains: cloudApplyForm.otherDomains,
        description: cloudApplyForm.description,
        dnsAccountId: cloudApplyForm.cloudAccountId,
        cloudAccountId: cloudApplyForm.cdnAccountId || 0,
        acmeAccountId: 0,
        type: "dns",
        provider: "acme-dns"
      } as any)
      message.success("已提交云账号签发请求")
      cloudApplyModalVisible.value = false
      if (res.data?.id) {
        openLogModal(res.data.id)
      } else {
        await fetchData()
      }
    } catch (error: any) {
      void 0
    } finally {
      submitting.value = false
    }
  }
  async function handleUploadCertificate() {
    if (!uploadForm.primaryDomain || !uploadForm.pem || !uploadForm.privateKey) {
      message.warning("请填写主域名、证书内容和私钥")
      return
    }
    submitting.value = true
    try {
      await CreateSSL({
        primaryDomain: uploadForm.primaryDomain,
        otherDomains: uploadForm.otherDomains,
        description: uploadForm.description,
        provider: "custom",
        acmeAccountId: 0,
        cloudAccountId: 0,
        type: "upload",
        pem: uploadForm.pem,
        privateKey: uploadForm.privateKey
      } as any)
      message.success("证书已上传")
      uploadModalVisible.value = false
      await fetchData()
    } finally {
      submitting.value = false
    }
  }
  async function handleApplyCertificate() {
    if (!applyForm.websiteId || !currentApplySSLId.value) {
      message.warning("请选择网站")
      return
    }
    submitting.value = true
    try {
      await ApplySSL({ websiteId: applyForm.websiteId, SSLId: currentApplySSLId.value })
      message.success("证书已绑定到网站")
      applyModalVisible.value = false
      await fetchData()
    } finally {
      submitting.value = false
    }
  }
  async function handlePushCDN() {
    if (!pushCDNForm.cloudAccountId || !currentPushSSL.value?.id) {
      message.warning("请选择云账号")
      return
    }
    submitting.value = true
    try {
      await PushToCDNAPI({
        sslId: currentPushSSL.value.id,
        cloudAccountId: pushCDNForm.cloudAccountId,
        targetDomain: pushCDNForm.targetDomain
      } as any)
      message.success("已成功推送到指定的 CDN")
      pushCDNModalVisible.value = false
    } catch (error: any) {
      void 0
    } finally {
      submitting.value = false
    }
  }
  function confirmDelete(row: SSLRow) {
    dialog.warning({
      title: "确认删除吗？",
      positiveText: "确认",
      negativeText: "取消",
      content:
        row.type === "caddy"
          ? "删除后仅清理面板中的证书记录，不会删除 Caddy 已签发的证书文件。"
          : "删除后将移除当前上传证书记录。",
      onPositiveClick: async () => {
        await DeleteSSL({ id: row.id })
        message.success("删除成功")
        await fetchData()
      }
    })
  }
  onMounted(async () => {
    await Promise.all([fetchData(), fetchWebsites(), fetchCloudAccounts()])
  })
  return {
    loading,
    submitting,
    pushRuleLoading,
    cdnDomainsLoading,
    tableData,
    pushRuleData,
    cdnDomainsOptions,
    logsData,
    syncModalVisible,
    uploadModalVisible,
    detailModalVisible,
    applyModalVisible,
    pushCDNModalVisible,
    cloudApplyModalVisible,
    pushRuleModalVisible,
    logModalVisible,
    currentSSL,
    currentPushSSL,
    cloudApplyForm,
    pushCDNForm,
    pushRuleForm,
    syncForm,
    applyForm,
    uploadForm,
    websiteOptions,
    cloudAccountOptions,
    selectedSyncWebsiteRuntimeText,
    selectedApplyWebsiteRuntimeText,
    detailTitle,
    columns,
    pushRuleColumns,
    buildBoundWebsiteRuntimeText,
    fetchData,
    handleCdnAccountChange,
    resetPushRuleForm,
    openCloudApplyModal,
    openSyncModal,
    openUploadModal,
    handleLogModalChange,
    handleSavePushRule,
    handleSyncCertificate,
    handleCloudApplyCertificate,
    handleUploadCertificate,
    handleApplyCertificate,
    handlePushCDN
  }
}
