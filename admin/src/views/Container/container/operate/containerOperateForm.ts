import type { Container } from "@/api/interface/container"
import { checkIpV4V6, checkPort } from "@/utils/util"

export const createDefaultContainerHelper = (): Container.ContainerHelper => ({
  containerID: "",
  memoryItem: 0,
  name: "",
  image: "",
  imageInput: false,
  forcePull: false,
  publishAllPorts: false,
  exposedPorts: [],
  network: "",
  ipv4: "",
  ipv6: "",
  volumes: [],
  cmdStr: "",
  entrypointStr: "",
  autoRemove: false,
  privileged: false,
  tty: false,
  openStdin: false,
  restartPolicy: "no",
  cpuShares: 0,
  nanoCPUs: 0,
  memory: 0,
  labelsStr: "",
  envStr: "",
  cmd: [],
  entrypoint: [],
  labels: [],
  env: []
})

export const cloneContainerHelper = (rowData?: Container.ContainerHelper) =>
  JSON.parse(JSON.stringify(rowData || createDefaultContainerHelper())) as Container.ContainerHelper

export const escapeQuotes = (input: string): string => {
  if (!input) return ""
  const placeholder = "___TEMP_ESCAPED_QUOTE___"
  let result = input.replace(/\\"/g, placeholder)
  result = result.replace(/"/g, '\\"')
  result = result.replace(new RegExp(placeholder, "g"), '\\"')
  return result
}

export const splitStringIgnoringQuotes = (input: string): string[] => {
  if (!input) return []
  input = input.replace(/\\"/g, "<quota>")
  const regex = /"([^"]*)"|(\S+)/g
  const result: string[] = []
  let match

  while ((match = regex.exec(input)) !== null) {
    if (match[1]) {
      result.push(match[1].replace(/<quota>/g, '\\"'))
    } else if (match[2]) {
      result.push(match[2].replace(/<quota>/g, '\\"'))
    }
  }
  return result
}

export const hydrateContainerFormForEdit = (rowData: Container.ContainerHelper) => {
  const currentMemory = rowData.memory
  rowData.memory = Number(typeof currentMemory === "number" ? currentMemory.toFixed(2) : 0)

  rowData.cmd = rowData?.cmd || []
  rowData.cmdStr = rowData.cmd
    .map(item => (item.indexOf(" ") !== -1 ? `"${escapeQuotes(item)}"` : item))
    .join(" ")

  rowData.entrypoint = rowData?.entrypoint || []
  rowData.entrypointStr = rowData.entrypoint
    .map(item => (item.indexOf(" ") !== -1 ? `"${escapeQuotes(item)}"` : item))
    .join(" ")

  rowData.labels = rowData.labels || []
  rowData.env = rowData.env || []
  rowData.labelsStr = rowData.labels.join("\n")
  rowData.envStr = rowData.env.join("\n")
  rowData.exposedPorts = rowData.exposedPorts || []
  for (const item of rowData.exposedPorts) {
    item.host = item.hostIP ? `${item.hostIP}:${item.hostPort}` : item.hostPort
  }
  rowData.volumes = rowData.volumes || []
}

export const createEmptyExposedPort = () => ({
  host: "",
  hostIP: "",
  containerPort: "",
  hostPort: "",
  protocol: "tcp"
})

export const createEmptyVolume = () => ({
  type: "bind",
  sourceDir: "",
  containerDir: "",
  mode: "rw"
})

export const validateExposedPorts = (
  ports: Array<any>,
  onError: (message: string) => void,
  t: (key: string) => string
) => {
  if (ports.length === 0) {
    return true
  }
  for (const port of ports) {
    if (port.host.indexOf(":") !== -1) {
      port.hostIP = port.host.substring(0, port.host.lastIndexOf(":"))
      if (checkIpV4V6(port.hostIP)) {
        onError(t("firewall.addressFormatError"))
        return false
      }
      port.hostPort = port.host.substring(port.host.lastIndexOf(":") + 1)
    } else {
      port.hostIP = ""
      port.hostPort = port.host
    }
    if (port.hostPort.indexOf("-") !== -1) {
      if (checkPort(port.hostPort.split("-")[0]) || checkPort(port.hostPort.split("-")[1])) {
        onError(t("firewall.portFormatError"))
        return false
      }
    } else if (checkPort(port.hostPort)) {
      onError(t("firewall.portFormatError"))
      return false
    }

    if (port.containerPort.indexOf("-") !== -1) {
      if (checkPort(port.containerPort.split("-")[0]) || checkPort(port.containerPort.split("-")[1])) {
        onError(t("firewall.portFormatError"))
        return false
      }
    } else if (checkPort(port.containerPort)) {
      onError(t("firewall.portFormatError"))
      return false
    }
  }
  return true
}

export const isFromApp = (rowData: Container.ContainerHelper) =>
  !!(rowData && rowData.labels && rowData.labels.indexOf("createdBy=Apps") > -1)

export const buildSubmitPayload = (rowData: Container.ContainerHelper) => {
  const payload = cloneContainerHelper(rowData)

  payload.env = payload.envStr ? payload.envStr.split("\n") : []
  payload.labels = payload.labelsStr ? payload.labelsStr.split("\n") : []
  payload.cmd = payload.cmdStr ? splitStringIgnoringQuotes(payload.cmdStr) : []
  payload.entrypoint = payload.entrypointStr ? splitStringIgnoringQuotes(payload.entrypointStr) : []
  payload.exposedPorts = payload.publishAllPorts ? [] : payload.exposedPorts
  payload.memory = Number(payload.memory)
  payload.nanoCPUs = Number(payload.nanoCPUs)

  return payload
}
