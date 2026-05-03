export const createDefaultFormModel = () => ({
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

export const applyRepairHintFromText = (
  text: string,
  fallback: boolean,
  target: {
    repairTipVisible: { value: boolean }
    repairTipTitle: { value: string }
    repairTipMessage: { value: string }
    repairTipCommands: { value: string }
    repairTipAction: { value: string }
  }
) => {
  if (text.includes("no compose command found")) {
    target.repairTipVisible.value = true
    target.repairTipTitle.value = "检测到 Compose 环境缺失"
    target.repairTipMessage.value = fallback
      ? text
      : "当前主机未检测到 docker compose / podman compose / podman-compose，导致无法 pull/up。可以先一键修复（需要 root 权限），或复制命令手动执行。"
    target.repairTipCommands.value = "sudo apt-get update\nsudo apt-get install -y podman-compose"
    target.repairTipAction.value = "compose"
    return
  }

  if (text.includes("short-name") && text.includes("did not resolve")) {
    target.repairTipVisible.value = true
    target.repairTipTitle.value = "检测到 Podman 短名解析失败"
    target.repairTipMessage.value = fallback
      ? text
      : "当前容器运行时配置不允许直接拉取简写镜像名。可以先一键修复（自动向 /etc/containers/registries.conf 追加 docker.io 源）。"
    target.repairTipCommands.value = ""
    target.repairTipAction.value = "short-name"
    return
  }

  if (text.includes("insufficient UIDs or GIDs")) {
    target.repairTipVisible.value = true
    target.repairTipTitle.value = "检测到 UID/GID 映射不足"
    target.repairTipMessage.value = fallback
      ? text
      : "当前用户缺乏足够的子 UID/GID 映射，导致无法创建容器命名空间。可以点击一键修复，系统将自动配置并重置命名空间。"
    target.repairTipCommands.value = ""
    target.repairTipAction.value = "subuid"
    return
  }

  if (text.includes("cgroup-manager") || text.includes("enable-linger")) {
    target.repairTipVisible.value = true
    target.repairTipTitle.value = "建议开启用户 Linger (保活) 支持"
    target.repairTipMessage.value = fallback
      ? text
      : "检测到当前 Podman 用户会话未开启 Linger，可能导致 cgroup 限制降级或容器异常退出。可以先一键修复开启该支持。"
    target.repairTipCommands.value = ""
    target.repairTipAction.value = "linger"
    return
  }

  if (text.includes("port is already allocated") || text.includes("address already in use") || text.includes("bind: address already in use")) {
    target.repairTipVisible.value = true
    target.repairTipTitle.value = "检测到端口冲突"
    target.repairTipMessage.value = fallback
      ? text
      : "当前应用所需的端口已被其他服务占用。可以点击一键修复，系统将自动寻找可用端口并换绑。"
    target.repairTipCommands.value = ""
    target.repairTipAction.value = "port-conflict"
  }
}
