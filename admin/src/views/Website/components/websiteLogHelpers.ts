export type WebsiteLogType = "access" | "error"
export type StatusFilter = "all" | "2xx" | "3xx" | "4xx" | "5xx"

export type ParsedLogEntry = {
  raw: string
  parsed: boolean
  formattedRaw: string
  timeText: string
  method: string
  path: string
  status?: number
  statusText: string
  ip: string
  host: string
  durationText: string
  sizeText: string
  userAgent: string
  userAgentFull: string
  referer: string
}

type ExtractedLogPayload = {
  timestamp: string
  jsonText: string
}

export const statusFilters: Array<{ label: string; value: StatusFilter }> = [
  { label: "全部", value: "all" },
  { label: "2xx", value: "2xx" },
  { label: "3xx", value: "3xx" },
  { label: "4xx", value: "4xx" },
  { label: "5xx", value: "5xx" }
]

export function getErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === "object") {
    const maybe = error as { message?: string }
    if (typeof maybe.message === "string" && maybe.message.trim()) {
      return maybe.message
    }
  }
  return fallback
}

export function parseLogLine(line: string): ParsedLogEntry | null {
  const raw = line.trim()
  if (!raw) return null
  const extracted = extractLogPayload(raw)
  try {
    const payload = JSON.parse(extracted?.jsonText || raw)
    const request = payload?.request || {}
    const headers = request?.headers || {}
    const status = readNumber(payload?.status)
    return {
      raw,
      parsed: true,
      formattedRaw: formatStructuredRaw(payload),
      timeText: formatLogTime(payload?.ts || extracted?.timestamp),
      method: readString(request?.method) || "-",
      path: readString(request?.uri) || readString(request?.path) || "/",
      status,
      statusText: status ? String(status) : "--",
      ip: readString(request?.remote_ip) || readString(request?.client_ip) || "-",
      host: readString(request?.host) || getHeaderValue(headers, "Host"),
      durationText: formatDuration(payload?.duration),
      sizeText: formatBytes(payload?.size),
      userAgent: simplifyUserAgent(getHeaderValue(headers, "User-Agent")),
      userAgentFull: getHeaderValue(headers, "User-Agent"),
      referer: getHeaderValue(headers, "Referer")
    }
  } catch {
    return {
      raw,
      parsed: false,
      formattedRaw: raw,
      timeText: formatLogTime(extracted?.timestamp),
      method: "LOG",
      path: raw,
      status: undefined,
      statusText: "--",
      ip: "-",
      host: "",
      durationText: "--",
      sizeText: "",
      userAgent: "",
      userAgentFull: "",
      referer: ""
    }
  }
}

export function getStatusTagType(status?: number): "default" | "info" | "success" | "warning" | "error" {
  if (!status) return "default"
  if (status >= 500) return "error"
  if (status >= 400) return "warning"
  if (status >= 300) return "info"
  if (status >= 200) return "success"
  return "default"
}

export function matchStatusFilter(item: ParsedLogEntry, currentFilter: StatusFilter) {
  if (currentFilter === "all") return true
  if (!item.status) return false
  return String(item.status).startsWith(currentFilter[0] || "")
}

export function matchSearchKeyword(item: ParsedLogEntry, keywordValue: string) {
  const keyword = keywordValue.trim().toLowerCase()
  if (!keyword) return true
  return [item.raw, item.path, item.ip, item.host, item.userAgent, item.userAgentFull, item.referer, item.statusText].some(value =>
    value.toLowerCase().includes(keyword)
  )
}

function extractLogPayload(raw: string): ExtractedLogPayload | null {
  const clean = stripAnsi(raw)
  const jsonStart = clean.indexOf("{")
  if (jsonStart < 0) return null
  const prefix = clean.slice(0, jsonStart).trim()
  const jsonText = clean.slice(jsonStart).trim()
  return {
    timestamp: extractTimestamp(prefix),
    jsonText
  }
}

function stripAnsi(value: string) {
  return value.replace(/\u001b\[[0-9;]*m/g, "")
}

function extractTimestamp(prefix: string) {
  const match = prefix.match(/\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}(?:\.\d+)?/)
  return match?.[0] || ""
}

function readString(value: unknown) {
  return typeof value === "string" ? value : ""
}

function readNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined
}

function getHeaderValue(headers: Record<string, unknown>, name: string) {
  const target = headers?.[name]
  if (Array.isArray(target)) {
    return typeof target[0] === "string" ? target[0] : ""
  }
  return typeof target === "string" ? target : ""
}

function formatLogTime(value: unknown) {
  if (typeof value === "number" && Number.isFinite(value)) {
    return new Date(value * 1000).toLocaleTimeString("zh-CN", {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit"
    })
  }
  if (typeof value === "string" && value) {
    const normalized = value.includes("/") ? value.replace(/\//g, "-") : value
    const date = new Date(normalized)
    if (!Number.isNaN(date.getTime())) {
      return date.toLocaleTimeString("zh-CN", {
        hour12: false,
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit"
      })
    }
  }
  return "--:--:--"
}

function formatDuration(value: unknown) {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
    return "--"
  }
  if (value >= 1) {
    return `${value.toFixed(value >= 10 ? 1 : 2)}s`
  }
  const ms = value * 1000
  if (ms >= 1) {
    return `${Math.round(ms)}ms`
  }
  return `${Math.max(1, Math.round(ms * 1000))}us`
}

function formatBytes(value: unknown) {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
    return ""
  }
  if (value < 1024) {
    return `${Math.round(value)}B`
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)}KB`
  }
  return `${(value / 1024 / 1024).toFixed(1)}MB`
}

function simplifyUserAgent(ua: string) {
  if (!ua) return ""
  const browser = detectBrowser(ua)
  const os = detectOS(ua)
  if (browser && os) return `${browser} / ${os}`
  if (browser) return browser
  return ua.length > 36 ? `${ua.slice(0, 36)}...` : ua
}

function detectBrowser(ua: string) {
  if (/curl\//i.test(ua)) return "curl"
  if (/PostmanRuntime/i.test(ua)) return "Postman"
  if (/Go-http-client/i.test(ua)) return "Go HTTP"
  if (/Edg\//i.test(ua)) return "Edge"
  if (/Chrome\//i.test(ua) && !/Edg\//i.test(ua)) return "Chrome"
  if (/Firefox\//i.test(ua)) return "Firefox"
  if (/Safari\//i.test(ua) && /Version\//i.test(ua) && !/Chrome\//i.test(ua)) return "Safari"
  return ""
}

function detectOS(ua: string) {
  if (/iPhone/i.test(ua)) return "iPhone"
  if (/iPad/i.test(ua)) return "iPad"
  if (/Android/i.test(ua)) return "Android"
  if (/Mac OS X|Macintosh/i.test(ua)) return "macOS"
  if (/Windows/i.test(ua)) return "Windows"
  if (/Linux/i.test(ua)) return "Linux"
  return ""
}

function formatStructuredRaw(payload: unknown) {
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload ?? "")
  }
}
