export function formatNumber(val: number) {
  return Number((val || 0).toFixed(2))
}

export function parseUtil(value: string) {
  return formatNumber(Number(String(value).replace(/[^\d.]/g, "")) || 0)
}

export function shortText(value: string, max: number) {
  return value.length > max ? `${value.substring(0, max - 3)}...` : value
}

export function formatUptime(uptime: number) {
  if (!uptime) return "--"
  const days = Math.floor(uptime / 86400)
  const hours = Math.floor((uptime % 86400) / 3600)
  const minutes = Math.floor((uptime % 3600) / 60)
  const seconds = Math.floor(uptime % 60)
  const parts: string[] = []
  if (days > 0) parts.push(`${days}天`)
  if (hours > 0) parts.push(`${hours}小时`)
  if (minutes > 0) parts.push(`${minutes}分钟`)
  if (!parts.length || seconds > 0) parts.push(`${seconds}秒`)
  return parts.join("")
}

export function progressColor(value: number) {
  if (value >= 85) return "#f43f5e"
  if (value >= 65) return "#f59e0b"
  return "#2563eb"
}

export function loadStatus(val: number, t: (key: string) => string) {
  if (val < 30) return t("home.runSmoothly")
  if (val < 70) return t("home.runNormal")
  if (val < 80) return t("home.runSlowly")
  return t("home.runJam")
}

export function loadWidth(cpuShowAll: boolean, cpuPercents: number[]) {
  if (!cpuShowAll || cpuPercents.length < 24) {
    return 310
  }
  const line = Math.floor(cpuPercents.length / 16)
  return line * 141 + 28
}
