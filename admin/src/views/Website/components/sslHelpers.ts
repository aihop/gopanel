import type { Website } from "@/api/interface/website"
import { buildRuntimeDetailText } from "@/utils/runtime"

export type WebsiteOption = Website.WebsiteDTO

export interface SSLRow {
  id: number
  primaryDomain: string
  domains: string
  type: string
  provider: string
  organization: string
  expireDate: string
  startDate: string
  description: string
  privateKey: string
  pem: string
  status: string
  cloudAccountId: number
  dnsAccountId: number
  websites?: Array<{ id: number; name?: string; primaryDomain?: string }>
}

type SSLRowSource = Partial<Website.SSL> & {
  id?: number
  organization?: string
  dnsAccountId?: number
  websites?: Array<Partial<Website.Website>>
}

export interface CloudApplyFormState {
  primaryDomain: string
  otherDomains: string
  cloudAccountId: number | null
  cdnAccountId: number | null
  description: string
}

export interface PushCDNFormState {
  cloudAccountId: number | null
  targetDomain: string
}

export interface PushRuleFormState {
  id: number | null
  cloudAccountId: number | null
  targetDomain: string
}

export interface SyncFormState {
  websiteId: number | null
}

export interface ApplyFormState {
  websiteId: number | null
}

export interface UploadFormState {
  primaryDomain: string
  otherDomains: string
  description: string
  pem: string
  privateKey: string
}

export const createDefaultCloudApplyForm = (): CloudApplyFormState => ({
  primaryDomain: "",
  otherDomains: "",
  cloudAccountId: null,
  cdnAccountId: null,
  description: ""
})

export const createDefaultPushCDNForm = (): PushCDNFormState => ({
  cloudAccountId: null,
  targetDomain: ""
})

export const createDefaultPushRuleForm = (): PushRuleFormState => ({
  id: null,
  cloudAccountId: null,
  targetDomain: ""
})

export const createDefaultSyncForm = (): SyncFormState => ({
  websiteId: null
})

export const createDefaultApplyForm = (): ApplyFormState => ({
  websiteId: null
})

export const createDefaultUploadForm = (): UploadFormState => ({
  primaryDomain: "",
  otherDomains: "",
  description: "",
  pem: "",
  privateKey: ""
})

export function sourceLabel(row: SSLRow | { type: string }) {
  if (row.type === "caddy") return { label: "Caddy 自动 HTTPS", tagType: "success" } as const
  if (row.type === "upload") return { label: "手动上传", tagType: "warning" } as const
  if (row.type.startsWith("dns-")) {
    const providerMap: Record<string, string> = {
      aliyun: "阿里云",
      tencentcloud: "腾讯云",
      cloudflare: "Cloudflare",
      volcengine: "火山引擎",
      huaweicloud: "华为云"
    }
    const provider = row.type.replace("dns-", "")
    return { label: `云账号 (${providerMap[provider] || provider})`, tagType: "info" } as const
  }
  return { label: row.type || "未知来源", tagType: "default" } as const
}

export function formatDateTime(value: string) {
  if (!value) return "--"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString("zh-CN", { hour12: false })
}

export function isExpired(value: string) {
  if (!value) return false
  const date = new Date(value)
  return !Number.isNaN(date.getTime()) && date.getTime() < Date.now()
}

export function downloadContent(content: string, fileName: string) {
  const blob = new Blob([content || ""], { type: "text/plain;charset=utf-8" })
  const link = document.createElement("a")
  link.href = URL.createObjectURL(blob)
  link.download = fileName
  link.click()
  URL.revokeObjectURL(link.href)
}

export function normalizeSSLRow(item: SSLRowSource): SSLRow {
  return {
    id: item.id || 0,
    primaryDomain: item.primaryDomain || "",
    domains: item.domains || item.otherDomains || "",
    type: item.type || "",
    provider: item.provider || "",
    organization: item.organization || item.issuerName || "",
    expireDate: item.expireDate || "",
    startDate: item.startDate || "",
    description: item.description || "",
    privateKey: item.privateKey || "",
    pem: item.pem || "",
    status: item.status || "",
    cloudAccountId: item.cloudAccountId || 0,
    dnsAccountId: item.dnsAccountId || 0,
    websites: item.websites?.map(site => ({
      id: site.id || 0,
      name: site.alias,
      primaryDomain: site.primaryDomain
    }))
  }
}

export function buildWebsiteRuntimeText(
  item?: (Partial<WebsiteOption> & { id?: number; name?: string; primaryDomain?: string }) | null
) {
  if (!item) return ""
  const prefix = item.name ? `${item.name} · ${item.primaryDomain || "--"}` : item.primaryDomain || `#${item.id || "-"}`
  const detail = buildRuntimeDetailText(item, {
    prefix,
    kindFallback: "Runtime",
    userFallback: "镜像默认",
    runtimePrefix: "",
    runUserPrefix: "用户："
  })
  const host = String(item.runtimeHost || "").trim()
  return host ? `${detail} · Host: ${host}` : detail
}
