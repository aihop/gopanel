import http from "@/api"
import type { ResPage } from "@/api/interface"
import type { SecurityEvent, SecurityMonitoringConfig } from "@/api/interface/securityMonitoring"

export function getSecurityEvents(params: {
  page: number
  limit: number
  status?: string
  level?: string
  sourceType?: string
}) {
  return http.get<ResPage<SecurityEvent>>("/security-monitoring/events", params)
}

export function getSecurityMonitoringConfig() {
  return http.get<SecurityMonitoringConfig>("/security-monitoring/config")
}

export function saveSecurityMonitoringConfig(config: SecurityMonitoringConfig) {
  return http.put<SecurityMonitoringConfig>("/security-monitoring/config", config)
}

export function evaluateSecurityRisks() {
  return http.post("/security-monitoring/evaluate")
}

export function analyzeSecurityEvent(id: number) {
  return http.post(`/security-monitoring/events/${id}/analyze`, undefined, 120000)
}
