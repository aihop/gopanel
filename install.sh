#!/bin/bash
set -euo pipefail
trap 'echo "错误：安装脚本在第 ${LINENO} 行中断，命令: ${BASH_COMMAND}" >&2' ERR

# ---- 基础变量 ----
APP_BRAND="${1:-GoPanel}"
API_BASE_URL="${API_BASE_URL:-https://gopanel.cn}"
API_BASE_URL="${API_BASE_URL%/}"
API_UPGRADE_URL="${API_UPGRADE_URL:-${API_BASE_URL}/api/panel/upgrade}"
API_TRACK_URL="${API_TRACK_URL:-${API_BASE_URL}/api/panel/installs/track}"
CONFIG_INSTALL_DIR="${CONFIG_INSTALL_DIR:-}"
CONFIG_PORT="${CONFIG_PORT:-5470}"
CONFIG_INSTALL_GPAGENT="${CONFIG_INSTALL_GPAGENT:-true}"
CONFIG_USER="${CONFIG_USER:-admin}"
CONFIG_PASSWORD="${CONFIG_PASSWORD:-$(openssl rand -hex 8)}"
CONFIG_SAFE_ENTER="${CONFIG_SAFE_ENTER:-$(openssl rand -hex 8)}"
CONFIG_CHANNEL="${CONFIG_CHANNEL:-${CHANNEL:-}}"
CONFIG_INSTALL_ID="${CONFIG_INSTALL_ID:-${INSTALL_ID:-}}"
RUNTIME_USER=""
RUN_AS_NORMAL_USER="false"
INVOKING_USER=""
PREEXISTING_INSTALL="false"
UPDATE_MODE="false"

os_name=""
arch_name=""
version=""
version_code=""
PACKAGE_URL=""
PACKAGE_NAME=""
GPAGENT_VERSION=""
GPAGENT_VERSION_CODE=""
GPAGENT_PACKAGE_URL=""
GPAGENT_PACKAGE_NAME=""
GPAGENT_FETCH_ERROR=""
SUDO_CMD=""
WORK_DIR=""
GPAGENT_WORK_DIR=""
BIN_GPC_PATH=""
BIN_GOPANEL_PATH=""
BIN_GPAGENT_PATH=""

log() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*" >&2; }
die() { echo "[ERROR] $*" >&2; exit 1; }

ensure_curl() {
  if command -v curl >/dev/null 2>&1; then
    return 0
  fi

  warn "检测到系统未安装 curl，正在自动安装..."

  if [ "${os_name:-}" = "darwin" ]; then
    if command -v brew >/dev/null 2>&1; then
      brew install curl
    else
      die "系统未安装 curl，且未检测到 Homebrew。请先安装 curl 后重试。"
    fi
  elif command -v apt-get >/dev/null 2>&1; then
    apt_update_soft
    run_privileged env DEBIAN_FRONTEND=noninteractive apt-get install -y curl ca-certificates
  elif command -v dnf >/dev/null 2>&1; then
    run_privileged dnf install -y curl ca-certificates
  elif command -v yum >/dev/null 2>&1; then
    run_privileged yum install -y curl ca-certificates
  elif command -v apk >/dev/null 2>&1; then
    run_privileged apk add --no-cache curl ca-certificates
  elif command -v pacman >/dev/null 2>&1; then
    run_privileged pacman -Sy --noconfirm curl ca-certificates
  elif command -v zypper >/dev/null 2>&1; then
    run_privileged zypper --non-interactive in -y curl ca-certificates
  else
    die "系统未安装 curl，且未识别到可用的包管理器（apt/yum/dnf/apk/pacman/zypper/brew）。请手动安装 curl 后重试。"
  fi

  if ! command -v curl >/dev/null 2>&1; then
    die "curl 安装失败，请手动安装后重试。"
  fi
}

cleanup() {
  if [ -n "${WORK_DIR}" ] && [ -d "${WORK_DIR}" ]; then
    rm -rf "${WORK_DIR}"
  fi
  if [ -n "${GPAGENT_WORK_DIR}" ] && [ -d "${GPAGENT_WORK_DIR}" ]; then
    rm -rf "${GPAGENT_WORK_DIR}"
  fi
}
trap cleanup EXIT

bool_is_true() {
  case "${1:-}" in
    1|true|TRUE|True|yes|YES|Yes|y|Y|on|ON|On)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

gpagent_service_name() {
  if [ "${os_name:-}" = "darwin" ]; then
    echo "io.aihop.gp-agent"
    return 0
  fi
  echo "gp-agent.service"
}

json_get() {
  local key="$1"
  local json="$2"

  # 使用 awk 提取简单的顶层 JSON 字段 (支持跨行和包含转义引号的字符串)
  # 匹配模式: "key"\s*:\s*"value" 或 "key"\s*:\s*123
  local val
  val="$(echo "$json" | awk -v k="\"$key\"" '
    BEGIN { FS=":"; RS="," }
    $1 ~ k {
      val = $2
      # 如果 value 中包含冒号(例如 URL)，将其拼凑回来
      for (i=3; i<=NF; i++) {
        val = val ":" $i
      }
      # 去掉前导和后置的空格、换行
      gsub(/^[ \t\n]+/, "", val)
      gsub(/[ \t\n]+$/, "", val)
      # 移除引号
      gsub(/^"/, "", val)
      gsub(/"$/, "", val)
      # 移除结尾的可能存在的 } 或者 ]
      gsub(/}$/, "", val)
      gsub(/]$/, "", val)
      gsub(/"$/, "", val)
      print val
      exit
    }
  ')"
  
  if [ -n "$val" ] && [ "$val" != "null" ]; then
    echo "$val"
    return 0
  fi
  
  # 如果 awk 没有提取到，且有 jq，则回退到 jq
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r ".${key} // empty"
    return 0
  fi

  return 1
}

require_cmds() {
  local missing=()
  local cmd
  for cmd in curl tar openssl awk sed find basename uname id; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      missing+=("$cmd")
    fi
  done
  if [ "${#missing[@]}" -gt 0 ]; then
    die "缺少必要命令: ${missing[*]}"
  fi
}

init_invoking_user() {
  if [ -n "${SUDO_USER:-}" ] && [ "${SUDO_USER}" != "root" ]; then
    INVOKING_USER="${SUDO_USER}"
  else
    INVOKING_USER="$(id -un 2>/dev/null || echo "root")"
  fi
}

user_home_dir() {
  local username="$1"
  if [ "$os_name" = "darwin" ]; then
    local home_dir
    home_dir="$(dscl . -read "/Users/${username}" NFSHomeDirectory 2>/dev/null | awk '{print $2}' || true)"
    if [ -n "${home_dir}" ]; then
      echo "${home_dir}"
    else
      echo "/Users/${username}"
    fi
    return 0
  fi
  eval echo "~${username}"
}

detect_platform() {
  local u m
  u="$(uname -s | tr '[:upper:]' '[:lower:]')"
  m="$(uname -m | tr '[:upper:]' '[:lower:]')"

  case "$u" in
    linux*) os_name="linux" ;;
    darwin*) os_name="darwin" ;;
    msys*|mingw*|cygwin*) die "检测到 Windows 环境，请使用 install.ps1 进行安装。" ;;
    *) die "暂不支持的系统: $u" ;;
  esac

  case "$m" in
    x86_64|amd64) arch_name="amd64" ;;
    arm64|aarch64) arch_name="arm64" ;;
    *) die "暂不支持的架构: $m" ;;
  esac

  if [ -z "${CONFIG_INSTALL_DIR}" ]; then
    if [ "$os_name" = "darwin" ]; then
      CONFIG_INSTALL_DIR="$(user_home_dir "${INVOKING_USER}")/.gopanel"
    else
      CONFIG_INSTALL_DIR="/opt/gopanel"
    fi
  fi
}

runtime_user_uid() {
  local username="$1"
  id -u "${username}" 2>/dev/null || true
}

runtime_user_runtime_dir() {
  local username="$1"
  local uid
  uid="$(runtime_user_uid "${username}")"
  if [ -z "${uid}" ]; then
    return 0
  fi
  echo "/run/user/${uid}"
}

runtime_user_podman_socket() {
  local username="$1"
  local runtime_dir
  runtime_dir="$(runtime_user_runtime_dir "${username}")"
  if [ -z "${runtime_dir}" ]; then
    return 0
  fi
  echo "unix://${runtime_dir}/podman/podman.sock"
}

detect_server_ip() {
  local ip_address=""
  if [ "${os_name:-}" = "linux" ]; then
    ip_address="$(hostname -I 2>/dev/null | awk '{print $1}' 2>/dev/null || true)"
    if [ -n "${ip_address}" ]; then
      echo "${ip_address}"
      return 0
    fi
  fi
  if [ "${os_name:-}" = "darwin" ]; then
    ip_address="$(ipconfig getifaddr en0 2>/dev/null || true)"
    if [ -z "${ip_address}" ]; then
      ip_address="$(ipconfig getifaddr en1 2>/dev/null || true)"
    fi
    if [ -n "${ip_address}" ]; then
      echo "${ip_address}"
      return 0
    fi
  fi
  ip_address="$(hostname 2>/dev/null || echo "127.0.0.1")"
  echo "${ip_address}"
}

# 公网 IP 探测：hostname -I 在云主机（NAT）上拿到的是私网地址，
# 面板访问地址必须给公网 IP，否则用户拿到 172.x/10.x 根本打不开。
# 端点顺序与面板内 ResolvePublicIP 一致：先国内可达，最后附一个国际兜底。
detect_public_ip() {
  local endpoints="https://myip.ipip.net https://4.ipw.cn https://ip.3322.net https://api.ipify.org"
  local url body ip
  for url in ${endpoints}; do
    body="$(curl -fsSL --max-time 3 "${url}" 2>/dev/null || true)"
    [ -z "${body}" ] && continue
    ip="$(printf '%s' "${body}" | grep -Eo '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -n 1 || true)"
    if [ -n "${ip}" ]; then
      echo "${ip}"
      return 0
    fi
  done
  echo ""
}

# 判断是否私网/保留地址（含 CGNAT 100.64/10），用于决定要不要额外显示公网地址
is_private_ip() {
  case "${1:-}" in
    10.*|127.*|192.168.*|169.254.*) return 0 ;;
    172.1[6-9].*|172.2[0-9].*|172.3[0-1].*) return 0 ;;
    100.6[4-9].*|100.[7-9][0-9].*|100.1[01][0-9].*|100.12[0-7].*) return 0 ;;
  esac
  return 1
}

detect_distro() {
  if [ "${os_name:-}" != "linux" ]; then
    echo ""
    return 0
  fi
  if [ -f /etc/os-release ]; then
    local id version_id
    id="$(awk -F= '$1=="ID"{gsub("\"","",$2);print $2}' /etc/os-release 2>/dev/null | head -n 1 || true)"
    version_id="$(awk -F= '$1=="VERSION_ID"{gsub("\"","",$2);print $2}' /etc/os-release 2>/dev/null | head -n 1 || true)"
    if [ -n "${id}" ] && [ -n "${version_id}" ]; then
      echo "${id}_${version_id}"
      return 0
    fi
    echo "${id}"
    return 0
  fi
  echo ""
}

detect_runtime() {
  if command -v docker >/dev/null 2>&1; then
    echo "docker"
    return 0
  fi
  if command -v podman >/dev/null 2>&1; then
    echo "podman"
    return 0
  fi
  echo ""
}

ensure_install_id() {
  if [ -z "${CONFIG_INSTALL_ID}" ]; then
    if [ -f "${CONFIG_INSTALL_DIR}/install_id" ]; then
      CONFIG_INSTALL_ID="$(cat "${CONFIG_INSTALL_DIR}/install_id" 2>/dev/null | tr -d ' \n\r\t' || true)"
    fi
  fi
  if [ -z "${CONFIG_INSTALL_ID}" ]; then
    CONFIG_INSTALL_ID="$(openssl rand -hex 16)"
  fi
  run_privileged mkdir -p "${CONFIG_INSTALL_DIR}"
  echo -n "${CONFIG_INSTALL_ID}" | run_privileged tee "${CONFIG_INSTALL_DIR}/install_id" >/dev/null 2>&1 || true
}

detect_preexisting_install() {
  if [ -f "${CONFIG_INSTALL_DIR}/gopanel" ] || [ -f "${CONFIG_INSTALL_DIR}/conf.yaml" ] || [ -d "${CONFIG_INSTALL_DIR}/db" ]; then
    PREEXISTING_INSTALL="true"
    return 0
  fi
  PREEXISTING_INSTALL="false"
}

prompt_update_if_installed() {
  if [ "${PREEXISTING_INSTALL}" != "true" ]; then
    return 0
  fi

  echo
  warn "检测到当前机器已经安装 GoPanel：${CONFIG_INSTALL_DIR}"
  echo "将检查最新版信息，并可直接执行原地升级。"

  fetch_upgrade_info

  local answer
  read -r -p "是否升级到最新版 ${version} (code: ${version_code})？[Y/n]: " answer || true
  answer="${answer:-Y}"
  case "${answer}" in
    y|Y|yes|YES)
      UPDATE_MODE="true"
      log "已选择升级现有 GoPanel。"
      ;;
    n|N|no|NO)
      log "已取消升级，安装流程结束。"
      exit 0
      ;;
    *)
      warn "输入无效，默认执行升级。"
      UPDATE_MODE="true"
      ;;
  esac
}

track_install_event() {
  local event="$1"
  local ver="${2:-}"
  if [ -z "${API_TRACK_URL}" ] || [ -z "${event}" ]; then
    return 0
  fi
  local channel="${CONFIG_CHANNEL}"
  if [ -z "${channel}" ]; then
    channel="unknown"
  fi
  local distro machine kernel runtime ip
  distro="$(detect_distro)"
  machine="$(uname -m 2>/dev/null || true)"
  kernel="$(uname -r 2>/dev/null || true)"
  runtime="$(detect_runtime)"
  ip="$(detect_server_ip)"

  local args=()
  args+=(--get)
  args+=(--data-urlencode "event=${event}")
  args+=(--data-urlencode "install_id=${CONFIG_INSTALL_ID}")
  args+=(--data-urlencode "channel=${channel}")
  args+=(--data-urlencode "os=${os_name}")
  args+=(--data-urlencode "arch=${arch_name}")
  args+=(--data-urlencode "ip=${ip}")
  if [ -n "${ver}" ]; then
    args+=(--data-urlencode "version=${ver}")
  fi
  if [ -n "${distro}" ]; then
    args+=(--data-urlencode "distro=${distro}")
  fi
  if [ -n "${machine}" ]; then
    args+=(--data-urlencode "machine=${machine}")
  fi
  if [ -n "${kernel}" ]; then
    args+=(--data-urlencode "kernel=${kernel}")
  fi
  if [ -n "${runtime}" ]; then
    args+=(--data-urlencode "runtime=${runtime}")
  fi

  curl -fsSL --max-time 3 "${args[@]}" "${API_TRACK_URL}" >/dev/null 2>&1 || true
}

init_privilege() {
  if [ "${EUID:-0}" -eq 0 ]; then
    SUDO_CMD=""
    log "当前已是 root 权限运行。"
    return 0
  fi

  if ! command -v sudo >/dev/null 2>&1; then
    die "当前不是 root，且系统未安装 sudo。请使用 root 账户执行，或先安装 sudo。"
  fi

  warn "检测到当前为非 root 用户，后续需要 sudo 权限。"
  warn "系统将提示输入密码以提升权限（Linux/macOS 通用）。"
  if ! sudo -v; then
    die "sudo 鉴权失败，请确认当前用户在 sudoers 中，或改用 root 执行。"
  fi
  SUDO_CMD="sudo"
}

run_privileged() {
  if [ -n "$SUDO_CMD" ]; then
    sudo "$@"
  else
    "$@"
  fi
}

# apt-get update 的软失败封装。
# 长期运行的机器上经常有失效的第三方源/已归档的 backports（Debian 11 的
# bullseye-backports 已移到 archive.debian.org，镜像站直接 404），
# 只要有一个源报 E: 整条命令就返回非 0。脚本是 set -e，会被这种
# 与本次安装无关的源拖死。这里降级为警告并把坏源打出来，继续尝试安装——
# 本地缓存里通常已有可用索引，真正装不上时后面的 apt-get install 会照常报错退出。
apt_update_soft() {
  local tmp rc
  tmp="$(mktemp -t gopanel_apt_update.XXXXXX)"
  rc=0
  run_privileged env DEBIAN_FRONTEND=noninteractive apt-get update -y 2>&1 | tee "${tmp}" || rc=$?
  if [ "${rc}" -ne 0 ]; then
    warn "apt-get update 未完全成功（退出码 ${rc}），以下软件源不可用："
    grep -E '^(E:|Err:)' "${tmp}" | head -n 10 | sed 's/^/       /' || true
    warn "若与本次安装无关可忽略，将继续尝试安装；如果随后安装失败，请先修复或移除上述软件源。"
  fi
  rm -f "${tmp}"
  return 0
}

detect_local_package() {
  if [ -n "${LOCAL_PACKAGE:-}" ]; then
    if [ -f "${LOCAL_PACKAGE}" ]; then
      PACKAGE_NAME="${LOCAL_PACKAGE}"
      PACKAGE_URL=""
      version="local"
      version_code="0"
      log "发现本地安装包（LOCAL_PACKAGE）: ${PACKAGE_NAME}，将跳过远程获取与下载。"
      return 0
    fi
    warn "已设置 LOCAL_PACKAGE，但文件不存在: ${LOCAL_PACKAGE}"
  fi

  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  local candidates=()
  local d
  for d in "$PWD" "$script_dir"; do
    if [ -d "$d" ]; then
      shopt -s nullglob
      candidates+=(
        "$d"/gopanel*"$os_name"*"$arch_name"*.tar.gz
        "$d"/gopanel*"$os_name"*"$arch_name"*.tgz
        "$d"/gopanel*.tar.gz
        "$d"/gopanel*.tgz
      )
      shopt -u nullglob
    fi
  done

  local newest=""
  local f
  for f in "${candidates[@]}"; do
    if [ -f "$f" ]; then
      if [ -z "$newest" ] || [ "$f" -nt "$newest" ]; then
        newest="$f"
      fi
    fi
  done

  if [ -n "$newest" ]; then
    PACKAGE_NAME="$newest"
    PACKAGE_URL=""
    version="local"
    version_code="0"
    log "发现本地安装包: ${PACKAGE_NAME}，将跳过远程获取与下载。"
    return 0
  fi

  return 1
}

is_in_china() {
  if curl -s -I -m 3 https://www.google.com >/dev/null 2>&1; then
    return 1 # Can access Google, not in China
  else
    return 0 # Cannot access Google, assume in China
  fi
}

fetch_upgrade_info() {
  local cur_version="${CUR_VERSION:-0.0.0}"
  local cur_version_code="${CUR_VERSION_CODE:-0}"
  local source="github"
  if is_in_china; then
    source="gitcode"
  fi
  local url
  url="${API_UPGRADE_URL}?versionCode=${cur_version_code}&version=${cur_version}&os=${os_name}&arch=${arch_name}&appBrand=${APP_BRAND}&source=${source}"

  log "检查最新版本..."
  local json
  if ! json="$(curl -fsSL "$url")"; then
    die "无法获取版本信息，请检查网络或接口地址: ${url}"
  fi

  local latest_name latest_code download_url
  if ! latest_name="$(json_get "latestVersionName" "$json")" \
    || ! latest_code="$(json_get "latestVersionCode" "$json")" \
    || ! download_url="$(json_get "downloadUrl" "$json")"; then
    die "版本接口返回非 JSON 或格式异常: ${url}"
  fi
  download_url="$(echo "$download_url" | sed -e 's/`//g' -e 's/^ *//g' -e 's/ *$//g')"

  if [ -z "$latest_name" ] || [ -z "$latest_code" ] || [ -z "$download_url" ]; then
    die "版本接口返回不完整: ${json}"
  fi

  version="$latest_name"
  version_code="$latest_code"
  PACKAGE_NAME="$(basename "$download_url")"
  PACKAGE_URL="$download_url"

  log "最新版本: ${version} (code: ${version_code})"
}

fetch_gpagent_upgrade_info() {
  local cur_version="${CUR_VERSION:-0.0.0}"
  local cur_version_code="${CUR_VERSION_CODE:-0}"
  local source="github"
  if is_in_china; then
    source="gitcode"
  fi
  local url
  url="${API_UPGRADE_URL}?versionCode=${cur_version_code}&version=${cur_version}&os=${os_name}&arch=${arch_name}&appBrand=${APP_BRAND}&package=gp-agent&source=${source}"

  GPAGENT_FETCH_ERROR=""
  log "获取 gp-agent 最新安装包信息..."

  local json
  if ! json="$(curl -fsSL "$url")"; then
    GPAGENT_FETCH_ERROR="无法获取版本信息，请检查网络或接口地址: ${url}"
    return 1
  fi

  local latest_name latest_code download_url
  if ! latest_name="$(json_get "latestVersionName" "$json")" \
    || ! latest_code="$(json_get "latestVersionCode" "$json")" \
    || ! download_url="$(json_get "downloadUrl" "$json")"; then
    GPAGENT_FETCH_ERROR="版本接口返回非 JSON 或格式异常: ${url}"
    return 1
  fi
  download_url="$(echo "$download_url" | sed -e 's/`//g' -e 's/^ *//g' -e 's/ *$//g')"

  if [ -z "$latest_name" ] || [ -z "$latest_code" ] || [ -z "$download_url" ]; then
    GPAGENT_FETCH_ERROR="版本接口返回不完整: ${json}"
    return 1
  fi

  GPAGENT_VERSION="$latest_name"
  GPAGENT_VERSION_CODE="$latest_code"
  GPAGENT_PACKAGE_NAME="$(basename "$download_url")"
  GPAGENT_PACKAGE_URL="$download_url"

  return 0
}

prompt_basic_config() {
  if [ "$os_name" = "linux" ]; then
    while true; do
      local input_dir
      read -r -e -p "请输入安装目录 (默认: ${CONFIG_INSTALL_DIR}): " input_dir || true
      input_dir="${input_dir:-$CONFIG_INSTALL_DIR}"
      if [[ ! "$input_dir" =~ ^/ ]]; then
        warn "安装目录必须是绝对路径。"
        continue
      fi
      if [[ "$input_dir" =~ [[:space:]] ]]; then
        warn "安装目录不能包含空格。"
        continue
      fi
      CONFIG_INSTALL_DIR="$input_dir"
      break
    done
  else
    log "macOS 默认安装目录: ${CONFIG_INSTALL_DIR}"
  fi

  while true; do
    local input_port
    read -r -e -p "请输入端口 (默认: ${CONFIG_PORT}): " input_port || true
    input_port="${input_port:-$CONFIG_PORT}"
    if [[ "$input_port" =~ ^[0-9]+$ ]] && [ "$input_port" -ge 1 ] && [ "$input_port" -le 65535 ]; then
      # 端口冲突检测
      if command -v ss >/dev/null 2>&1; then
        if ss -tuln | grep -q ":${input_port} "; then
          warn "端口 ${input_port} 似乎已被占用，请尝试其他端口。"
          continue
        fi
      elif command -v netstat >/dev/null 2>&1; then
        if netstat -tuln | grep -q ":${input_port} "; then
          warn "端口 ${input_port} 似乎已被占用，请尝试其他端口。"
          continue
        fi
      elif command -v lsof >/dev/null 2>&1; then
        if run_privileged lsof -i ":${input_port}" >/dev/null 2>&1; then
          warn "端口 ${input_port} 似乎已被占用，请尝试其他端口。"
          continue
        fi
      fi

      CONFIG_PORT="$input_port"
      break
    fi
    warn "端口必须是 1-65535 之间的整数。"
  done

  while true; do
    local input_user
    read -r -e -p "请设置登录用户名 (默认: ${CONFIG_USER}): " input_user || true
    input_user="${input_user:-$CONFIG_USER}"
    if [[ "$input_user" =~ [[:space:]] ]] || [[ "$input_user" =~ [^a-zA-Z0-9_-] ]]; then
      warn "用户名只能包含字母、数字、下划线、连字符，且不能有空格。"
      continue
    fi
    CONFIG_USER="$input_user"
    break
  done

  while true; do
    local input_password
    read -r -e -p "请设置登录密码 (默认: ${CONFIG_PASSWORD}): " input_password || true
    input_password="${input_password:-$CONFIG_PASSWORD}"
    if [[ "$input_password" =~ [[:space:]] ]]; then
      warn "密码不能包含空格。"
      continue
    fi
    CONFIG_PASSWORD="$input_password"
    break
  done

  while true; do
    local input_safe_enter
    read -r -e -p "请设置安全入口 (默认: ${CONFIG_SAFE_ENTER}): " input_safe_enter || true
    input_safe_enter="${input_safe_enter:-$CONFIG_SAFE_ENTER}"
    if [[ "$input_safe_enter" =~ [[:space:]] ]] || [[ "$input_safe_enter" =~ [^a-zA-Z0-9] ]]; then
      warn "安全入口只能包含字母数字，且不能有空格。"
      continue
    fi
    CONFIG_SAFE_ENTER="$input_safe_enter"
    break
  done
}

prompt_gpagent_install() {
  local default_install="true"
  local prompt_hint="Y/n"
  local prompt_default="Y"
  if ! bool_is_true "${CONFIG_INSTALL_GPAGENT}"; then
    default_install="false"
    prompt_hint="y/N"
    prompt_default="N"
  fi

  echo
  echo "gp-agent 提供网站托管、进程守护等能力。"
  echo "如果你准备用 GoPanel 来做网站，建议安装 gp-agent。"

  local answer
  read -r -p "是否安装 gp-agent？[${prompt_hint}]: " answer || true
  answer="${answer:-$prompt_default}"

  case "${answer}" in
    y|Y|yes|YES)
      CONFIG_INSTALL_GPAGENT="true"
      log "已选择安装 gp-agent。"
      ;;
    n|N|no|NO)
      CONFIG_INSTALL_GPAGENT="false"
      log "已跳过 gp-agent 安装。"
      ;;
    *)
      if [ "${default_install}" = "true" ]; then
        warn "输入无效，默认安装 gp-agent。"
        CONFIG_INSTALL_GPAGENT="true"
      else
        warn "输入无效，默认跳过 gp-agent。"
        CONFIG_INSTALL_GPAGENT="false"
      fi
      ;;
  esac
}

download_package() {
  if [ -f "${PACKAGE_NAME}" ]; then
    log "发现本地安装包: ${PACKAGE_NAME}，跳过下载。"
    return 0
  fi
  log "开始下载安装包: ${PACKAGE_NAME} - ${PACKAGE_URL}"
  curl -fSL -o "${PACKAGE_NAME}" "${PACKAGE_URL}" || die "下载安装包失败"
}

stop_existing_services() {
  if [ "${UPDATE_MODE}" != "true" ]; then
    return 0
  fi

  log "升级模式：先停止本机 gopanel/gpc 服务..."
  if [ "$os_name" = "linux" ]; then
    if command -v systemctl >/dev/null 2>&1; then
      run_privileged systemctl stop gopanel.service >/dev/null 2>&1 || true
      run_privileged systemctl stop gpc.service >/dev/null 2>&1 || true
    fi
  else
    run_privileged launchctl bootout system /Library/LaunchDaemons/io.aihop.gopanel.plist >/dev/null 2>&1 || true
    run_privileged launchctl bootout system /Library/LaunchDaemons/io.aihop.gpc.plist >/dev/null 2>&1 || true
  fi
}

stop_gpagent_service() {
  if [ "$os_name" = "linux" ]; then
    if command -v systemctl >/dev/null 2>&1; then
      run_privileged systemctl stop "$(gpagent_service_name)" >/dev/null 2>&1 || true
    fi
    return 0
  fi
  local plist="/Library/LaunchDaemons/$(gpagent_service_name).plist"
  run_privileged launchctl bootout system "${plist}" >/dev/null 2>&1 || true
}

extract_and_find_binaries() {
  WORK_DIR="$(mktemp -d -t gopanel_install.XXXXXX)"
  log "解压安装包到临时目录: ${WORK_DIR}"
  tar -zxf "${PACKAGE_NAME}" -C "${WORK_DIR}" || die "解压失败，安装包可能损坏"

  BIN_GPC_PATH="$(find "${WORK_DIR}" -type f -name gpc | head -n 1)"
  BIN_GOPANEL_PATH="$(find "${WORK_DIR}" -type f -name gopanel | head -n 1)"
  BIN_GPAGENT_PATH="$(find "${WORK_DIR}" -type f -name gp-agent | head -n 1)"

  [ -n "${BIN_GPC_PATH}" ] || die "安装包中未找到 gpc 二进制文件"
  [ -n "${BIN_GOPANEL_PATH}" ] || die "安装包中未找到 gopanel 二进制文件"
}

prepare_gpagent_binary() {
  if ! bool_is_true "${CONFIG_INSTALL_GPAGENT}"; then
    BIN_GPAGENT_PATH=""
    return 0
  fi

  if fetch_gpagent_upgrade_info; then
    log "gp-agent 将使用最新安装包: ${GPAGENT_VERSION} (code: ${GPAGENT_VERSION_CODE})"
    if [ -n "${BIN_GPAGENT_PATH}" ] && [ -n "${PACKAGE_URL}" ] && [ "${GPAGENT_PACKAGE_URL}" = "${PACKAGE_URL}" ]; then
      log "gp-agent 复用当前主包中的二进制文件。"
      return 0
    fi

    GPAGENT_WORK_DIR="$(mktemp -d -t gpagent_install.XXXXXX)"
    local gpagent_archive="${GPAGENT_WORK_DIR}/${GPAGENT_PACKAGE_NAME}"
    log "开始下载 gp-agent 最新安装包: ${GPAGENT_PACKAGE_NAME}"
    curl -fSL -o "${gpagent_archive}" "${GPAGENT_PACKAGE_URL}" || die "下载 gp-agent 安装包失败"
    tar -zxf "${gpagent_archive}" -C "${GPAGENT_WORK_DIR}" || die "解压 gp-agent 安装包失败"

    local latest_gpagent_path
    latest_gpagent_path="$(find "${GPAGENT_WORK_DIR}" -type f -name gp-agent | head -n 1)"
    if [ -z "${latest_gpagent_path}" ]; then
      if [ -n "${BIN_GPAGENT_PATH}" ]; then
        warn "最新安装包中未找到 gp-agent，回退到当前主包内置的 gp-agent。"
        return 0
      fi
      if [ "${UPDATE_MODE}" = "true" ] && [ -x "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
        warn "最新安装包中未找到 gp-agent，保留本机现有 gp-agent。"
        BIN_GPAGENT_PATH=""
        return 0
      fi
      warn "最新安装包中未找到 gp-agent 二进制文件，已跳过 gp-agent 安装。"
      CONFIG_INSTALL_GPAGENT="false"
      BIN_GPAGENT_PATH=""
      return 0
    fi

    BIN_GPAGENT_PATH="${latest_gpagent_path}"
    return 0
  fi

  if [ -n "${BIN_GPAGENT_PATH}" ]; then
    warn "获取 gp-agent 最新安装包失败，回退到当前主包内置的 gp-agent。原因: ${GPAGENT_FETCH_ERROR}"
    return 0
  fi

  if [ "${UPDATE_MODE}" = "true" ] && [ -x "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
    warn "获取 gp-agent 最新安装包失败，保留本机现有 gp-agent。原因: ${GPAGENT_FETCH_ERROR}"
    BIN_GPAGENT_PATH=""
    return 0
  fi

  die "已选择安装 gp-agent，但无法获取最新安装包，且当前主包未包含 gp-agent。原因: ${GPAGENT_FETCH_ERROR}"
}

install_gpc_binary() {
  log "安装 gpc 到 /usr/local/bin/gpc"
  
  # 如果目标文件已经存在，可能是由于它正在运行导致 "Text file busy"
  if [ -f /usr/local/bin/gpc ]; then
    log "检测到旧版 gpc 存在，尝试停止服务..."
    if [ "$os_name" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
      run_privileged systemctl stop gpc.service >/dev/null 2>&1 || true
    elif [ "$os_name" = "darwin" ] && command -v launchctl >/dev/null 2>&1; then
      run_privileged launchctl bootout system /Library/LaunchDaemons/io.aihop.gpc.plist >/dev/null 2>&1 || true
    fi
    # 删除旧文件，而不是直接覆盖，能100%避免 Text file busy
    run_privileged rm -f /usr/local/bin/gpc
  fi

  run_privileged mkdir -p /usr/local/bin
  run_privileged cp -f "${BIN_GPC_PATH}" /usr/local/bin/gpc
  run_privileged chmod 755 /usr/local/bin/gpc
}

install_gopanel_binary() {
  log "安装 gopanel 到 ${CONFIG_INSTALL_DIR}"
  run_privileged mkdir -p "${CONFIG_INSTALL_DIR}"
  if [ "${UPDATE_MODE}" = "true" ] && [ -f "${CONFIG_INSTALL_DIR}/gopanel" ]; then
    log "升级模式：备份旧版 gopanel"
    run_privileged cp -f "${CONFIG_INSTALL_DIR}/gopanel" "${CONFIG_INSTALL_DIR}/gopanel.bak"
  fi
  run_privileged cp -f "${BIN_GOPANEL_PATH}" "${CONFIG_INSTALL_DIR}/gopanel"
  run_privileged chmod 755 "${CONFIG_INSTALL_DIR}/gopanel"
}

install_gpagent_binary() {
  if ! bool_is_true "${CONFIG_INSTALL_GPAGENT}"; then
    log "已按选择跳过 gp-agent 安装。"
    return 0
  fi
  if [ -z "${BIN_GPAGENT_PATH}" ]; then
    if [ "${UPDATE_MODE}" = "true" ] && [ -x "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
      log "升级模式：继续保留现有 gp-agent 二进制。"
      return 0
    fi
    die "已选择安装 gp-agent，但未找到可用的 gp-agent 二进制文件"
  fi
  if [ "${UPDATE_MODE}" = "true" ]; then
    stop_gpagent_service
    if [ -f "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
      log "升级模式：备份旧版 gp-agent"
      run_privileged cp -f "${CONFIG_INSTALL_DIR}/gp-agent" "${CONFIG_INSTALL_DIR}/gp-agent.bak"
    fi
  fi
  log "安装 gp-agent 到 ${CONFIG_INSTALL_DIR}/gp-agent"
  run_privileged mkdir -p "${CONFIG_INSTALL_DIR}"
  run_privileged cp -f "${BIN_GPAGENT_PATH}" "${CONFIG_INSTALL_DIR}/gp-agent"
  run_privileged chmod 755 "${CONFIG_INSTALL_DIR}/gp-agent"
}

write_init_yaml() {
  local should_write_podman_socket="false"
  if [ "${os_name}" = "linux" ] && [ "${RUNTIME_USER}" != "root" ] && command -v podman >/dev/null 2>&1; then
    should_write_podman_socket="true"
  fi

  if [ "${UPDATE_MODE}" = "true" ] && [ -f "${CONFIG_INSTALL_DIR}/conf.yaml" ]; then
    if [ "${should_write_podman_socket}" != "true" ]; then
      log "升级模式：保留现有 conf.yaml/init.yaml，不重写初始化配置。"
      return 0
    fi
    local tmp_file podman_socket
    tmp_file="$(mktemp -t gopanel_init.XXXXXX)"
    podman_socket="$(runtime_user_podman_socket "${RUNTIME_USER}")"
    cat >"${tmp_file}" <<EOF
base_dir: "${CONFIG_INSTALL_DIR}"
container_runtime: "podman"
docker_sock_path: "${podman_socket}"
EOF
    run_privileged cp "${tmp_file}" "${CONFIG_INSTALL_DIR}/init.yaml"
    rm -f "${tmp_file}"
    log "升级模式：已写入 rootless Podman 初始化配置，用于同步 DockerSockPath。"
    return 0
  fi

  local tmp_file
  tmp_file="$(mktemp -t gopanel_init.XXXXXX)"
  cat >"${tmp_file}" <<EOF
base_dir: "${CONFIG_INSTALL_DIR}"
port: ${CONFIG_PORT}
user: "${CONFIG_USER}"
password: "${CONFIG_PASSWORD}"
safe_enter: "${CONFIG_SAFE_ENTER}"
install_id: "${CONFIG_INSTALL_ID}"
EOF
  if [ "${should_write_podman_socket}" = "true" ]; then
    local podman_socket
    podman_socket="$(runtime_user_podman_socket "${RUNTIME_USER}")"
    cat >>"${tmp_file}" <<EOF
container_runtime: "podman"
docker_sock_path: "${podman_socket}"
EOF
  fi
  run_privileged cp "${tmp_file}" "${CONFIG_INSTALL_DIR}/init.yaml"
  rm -f "${tmp_file}"
}

# 确保运行用户具备 rootless podman 所需条件：有效可写的家目录 + subuid/subgid。
# 复用已存在的系统账号时尤其关键——这类账号家目录常是 /nonexistent，会导致
# podman 建 $HOME/.local/share/containers 时 "mkdir /nonexistent/.local: permission denied"，
# 用户级 podman.socket 起不来，整个 rootless podman 不可用。
ensure_rootless_user_ready() {
  local username="$1"
  local uhome
  uhome="$(getent passwd "${username}" 2>/dev/null | cut -d: -f6)"
  # 家目录无效（空 / /nonexistent / / 等，或目录不存在）→ 改用安装目录作为家目录并修好属主
  if [ -z "${uhome}" ] || [ "${uhome}" = "/nonexistent" ] || [ "${uhome}" = "/" ] || [ "${uhome}" = "/dev/null" ] || ! run_privileged test -d "${uhome}"; then
    warn "运行用户 ${username} 家目录无效(${uhome:-空})，rootless podman 需要可写家目录，改为 ${CONFIG_INSTALL_DIR}。"
    run_privileged mkdir -p "${CONFIG_INSTALL_DIR}"
    run_privileged usermod -d "${CONFIG_INSTALL_DIR}" "${username}" || warn "设置家目录失败，请手动执行: usermod -d ${CONFIG_INSTALL_DIR} ${username}"
    run_privileged chown "${username}:${username}" "${CONFIG_INSTALL_DIR}" 2>/dev/null || run_privileged chown "${username}" "${CONFIG_INSTALL_DIR}" 2>/dev/null || true
  fi

  # Ensure subuid and subgid for rootless podman
  if ! grep -q "^${username}:" /etc/subuid 2>/dev/null; then
    log "为用户 ${username} 配置 subuid 映射..."
    run_privileged usermod --add-subuids 100000-165535 "${username}" || warn "配置 subuid 失败，可能需要手动配置"
  fi
  if ! grep -q "^${username}:" /etc/subgid 2>/dev/null; then
    log "为用户 ${username} 配置 subgid 映射..."
    run_privileged usermod --add-subgids 100000-165535 "${username}" || warn "配置 subgid 失败，可能需要手动配置"
  fi
}

create_linux_user_if_needed() {
  local username="$1"
  if id "$username" >/dev/null 2>&1; then
    log "用户 ${username} 已存在，直接复用。"
    # 复用用户也要保证家目录/subid 有效，否则 rootless podman 会因 /nonexistent 家目录挂掉
    ensure_rootless_user_ready "$username"
    return 0
  fi

  if run_privileged test -x /usr/sbin/nologin; then
    run_privileged useradd --system --create-home --home-dir "${CONFIG_INSTALL_DIR}" --shell /usr/sbin/nologin "$username"
  else
    run_privileged useradd --system --create-home --home-dir "${CONFIG_INSTALL_DIR}" --shell /bin/bash "$username"
  fi
  # Ensure home directory ownership (useradd may skip chown if dir already exists)
  run_privileged chown "${username}:${username}" "${CONFIG_INSTALL_DIR}" 2>/dev/null || true

  ensure_rootless_user_ready "$username"
}

create_macos_user_if_needed() {
  local username="$1"
  if id "$username" >/dev/null 2>&1; then
    log "用户 ${username} 已存在，直接复用。"
    return 0
  fi

  local max_uid next_uid
  max_uid="$(
    run_privileged dscl . -list /Users UniqueID | awk '{print $2}' | sort -n | tail -1
  )"
  next_uid=$((max_uid + 1))

  run_privileged dscl . -create "/Users/${username}"
  run_privileged dscl . -create "/Users/${username}" UserShell /usr/bin/false
  run_privileged dscl . -create "/Users/${username}" RealName "GoPanel Service User"
  run_privileged dscl . -create "/Users/${username}" UniqueID "${next_uid}"
  run_privileged dscl . -create "/Users/${username}" PrimaryGroupID 20
  run_privileged dscl . -create "/Users/${username}" NFSHomeDirectory "/Users/${username}"
  local strong_pw
  strong_pw="Gp@$(openssl rand -hex 12)Aa1!"
  run_privileged dscl . -passwd "/Users/${username}" "${strong_pw}" || die "macOS 创建用户失败（密码策略校验未通过）。建议直接使用当前用户运行，或手动创建用户后重试。"
  run_privileged mkdir -p "/Users/${username}"
  run_privileged chown "${username}:staff" "/Users/${username}"
}

choose_gopanel_runtime_user() {
  if [ "$os_name" != "linux" ]; then
    RUN_AS_NORMAL_USER="true"
    local current_user="${INVOKING_USER}"
    RUNTIME_USER="${current_user}"

    local answer
    read -r -p "gopanel 是否使用其他用户运行？[y/N]: " answer || true
    case "${answer}" in
      y|Y|yes|YES)
        local target_user
        read -r -e -p "请输入运行用户名 (默认: ${current_user}): " target_user || true
        target_user="${target_user:-$current_user}"
        if [[ ! "$target_user" =~ ^[a-zA-Z_][a-zA-Z0-9_-]*$ ]]; then
          die "用户名不合法: ${target_user}"
        fi
        if [ "${target_user}" != "${current_user}" ]; then
          create_macos_user_if_needed "$target_user"
        fi
        RUNTIME_USER="$target_user"
        ;;
      *)
        ;;
    esac

    if [ "$os_name" = "darwin" ]; then
      CONFIG_INSTALL_DIR="$(user_home_dir "${RUNTIME_USER}")/.gopanel"
      log "macOS 安装目录将使用: ${CONFIG_INSTALL_DIR}"
    fi

    log "gopanel 运行用户: ${RUNTIME_USER}"
    return 0
  fi

  # 检测是否已安装 Docker 且未安装 Podman
  # 如果安装了 Docker，强制要求以 root 运行（因为 Docker 守护进程需要 root 权限，且普通用户加入 docker 组有安全风险，这里选择稳妥的强制 root）
  if command -v docker >/dev/null 2>&1 && ! command -v podman >/dev/null 2>&1; then
    warn "检测到系统已安装 Docker。"
    warn "由于 Docker 架构特性，GoPanel 需要使用 root 权限运行才能正常管理容器。"
    RUN_AS_NORMAL_USER="false"
    RUNTIME_USER="root"
    log "已强制配置 gopanel 运行用户为: root"
    return 0
  fi

  local answer
  read -r -p "是否以普通用户运行 gopanel？[Y/n]: " answer || true
  case "${answer}" in
    ""|y|Y|yes|YES)
      RUN_AS_NORMAL_USER="true"
      local default_user="gopanel"
      local target_user
      read -r -e -p "请输入运行用户名 (默认: ${default_user}): " target_user || true
      target_user="${target_user:-$default_user}"
      if [[ ! "$target_user" =~ ^[a-zA-Z_][a-zA-Z0-9_-]*$ ]]; then
        die "用户名不合法: ${target_user}"
      fi
      create_linux_user_if_needed "$target_user"
      RUNTIME_USER="$target_user"
      ;;
    n|N|no|NO)
      RUN_AS_NORMAL_USER="false"
      RUNTIME_USER="root"
      ;;
    *)
      warn "输入无效，默认按 Yes 以普通用户运行。"
      RUN_AS_NORMAL_USER="true"
      local default_user="gopanel"
      local target_user
      read -r -e -p "请输入运行用户名 (默认: ${default_user}): " target_user || true
      target_user="${target_user:-$default_user}"
      if [[ ! "$target_user" =~ ^[a-zA-Z_][a-zA-Z0-9_-]*$ ]]; then
        die "用户名不合法: ${target_user}"
      fi
      create_linux_user_if_needed "$target_user"
      RUNTIME_USER="$target_user"
      ;;
  esac

  log "gopanel 运行用户: ${RUNTIME_USER}"
}

ensure_install_dir_owner() {
  if [ -d "${CONFIG_INSTALL_DIR}" ]; then
    local grp=""
    grp="$(id -gn "${RUNTIME_USER}" 2>/dev/null || true)"
    if [ -n "${grp}" ]; then
      run_privileged chown -R "${RUNTIME_USER}:${grp}" "${CONFIG_INSTALL_DIR}"
    else
      run_privileged chown -R "${RUNTIME_USER}" "${CONFIG_INSTALL_DIR}"
    fi

    if [ "${RUNTIME_USER}" != "root" ]; then
      if [ -n "${grp}" ]; then
        run_privileged chown root:"${grp}" "${CONFIG_INSTALL_DIR}"
      else
        run_privileged chown root:"${RUNTIME_USER}" "${CONFIG_INSTALL_DIR}" || true
      fi
      run_privileged chmod 2775 "${CONFIG_INSTALL_DIR}"
    fi
  fi
}

install_service_gpc_linux() {
  local tmp_service
  tmp_service="$(mktemp -t gpc.service.XXXXXX)"
  cat >"${tmp_service}" <<EOF
[Unit]
Description=GoPanel Control (gpc)
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gpc service --base-dir ${CONFIG_INSTALL_DIR}
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
  run_privileged cp "${tmp_service}" /etc/systemd/system/gpc.service
  rm -f "${tmp_service}"
  run_privileged systemctl daemon-reload
  run_privileged systemctl enable gpc.service
  run_privileged systemctl restart gpc.service
}

install_service_gopanel_linux() {
  local tmp_service
  local runtime_home runtime_dir
  runtime_home="$(user_home_dir "${RUNTIME_USER}")"
  runtime_dir="$(runtime_user_runtime_dir "${RUNTIME_USER}")"
  tmp_service="$(mktemp -t gopanel.service.XXXXXX)"
  local unit_after="network.target gpc.service"
  local unit_wants=""
  if [ "${RUNTIME_USER}" = "root" ]; then
    unit_after="network.target docker.socket podman.socket gpc.service"
    unit_wants="docker.socket podman.socket"
  fi
  cat >"${tmp_service}" <<EOF
[Unit]
Description=GoPanel
After=${unit_after}
EOF
  if [ -n "${unit_wants}" ]; then
    cat >>"${tmp_service}" <<EOF
Wants=${unit_wants}
EOF
  fi
  cat >>"${tmp_service}" <<EOF
Requires=gpc.service

[Service]
Type=simple
User=${RUNTIME_USER}
Group=${RUNTIME_USER}
WorkingDirectory=${CONFIG_INSTALL_DIR}
ExecStart=${CONFIG_INSTALL_DIR}/gopanel
Restart=always
RestartSec=2
Environment="HOME=${runtime_home}"
EOF
  if [ "${RUNTIME_USER}" != "root" ] && [ -n "${runtime_dir}" ]; then
    cat >>"${tmp_service}" <<EOF
Environment="XDG_RUNTIME_DIR=${runtime_dir}"
Environment="DBUS_SESSION_BUS_ADDRESS=unix:path=${runtime_dir}/bus"
EOF
  fi
  cat >>"${tmp_service}" <<EOF

[Install]
WantedBy=multi-user.target
EOF
  run_privileged cp "${tmp_service}" /etc/systemd/system/gopanel.service
  rm -f "${tmp_service}"
  run_privileged systemctl daemon-reload
  run_privileged systemctl enable gopanel.service
  run_privileged systemctl restart gopanel.service
}

install_service_gpagent_linux() {
  if ! bool_is_true "${CONFIG_INSTALL_GPAGENT}"; then
    log "已跳过 gp-agent 自启动配置。"
    return 0
  fi
  if [ ! -x "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
    warn "未检测到 gp-agent 二进制，跳过 gp-agent service 配置"
    return 0
  fi
  local tmp_service
  local runtime_home runtime_dir
  runtime_home="$(user_home_dir "${RUNTIME_USER}")"
  runtime_dir="$(runtime_user_runtime_dir "${RUNTIME_USER}")"
  tmp_service="$(mktemp -t gp-agent.service.XXXXXX)"
  local unit_after="network.target"
  local unit_wants=""
  if [ "${RUNTIME_USER}" = "root" ]; then
    unit_after="network.target docker.socket podman.socket"
    unit_wants="docker.socket podman.socket"
  fi
  cat >"${tmp_service}" <<EOF
[Unit]
Description=GoPanel Agent (gp-agent)
After=${unit_after}
EOF
  if [ -n "${unit_wants}" ]; then
    cat >>"${tmp_service}" <<EOF
Wants=${unit_wants}
EOF
  fi
  cat >>"${tmp_service}" <<EOF
[Service]
Type=simple
User=${RUNTIME_USER}
Group=${RUNTIME_USER}
WorkingDirectory=${CONFIG_INSTALL_DIR}
ExecStart=${CONFIG_INSTALL_DIR}/gp-agent service --base-dir ${CONFIG_INSTALL_DIR}
Restart=always
RestartSec=2

Environment="HOME=${runtime_home}"
Environment="CADDY_DATA_DIR=${CONFIG_INSTALL_DIR}/caddy/data"
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

LimitNOFILE=65535
OOMScoreAdjust=-100
EOF
  if [ "${RUNTIME_USER}" != "root" ] && [ -n "${runtime_dir}" ]; then
    cat >>"${tmp_service}" <<EOF
Environment="XDG_RUNTIME_DIR=${runtime_dir}"
Environment="DBUS_SESSION_BUS_ADDRESS=unix:path=${runtime_dir}/bus"
EOF
  fi
  cat >>"${tmp_service}" <<EOF

[Install]
WantedBy=multi-user.target
EOF
  run_privileged cp "${tmp_service}" /etc/systemd/system/gp-agent.service
  rm -f "${tmp_service}"
  run_privileged systemctl daemon-reload
  run_privileged systemctl enable gp-agent.service
  run_privileged systemctl restart gp-agent.service
}

install_service_gpc_macos() {
  local plist="/Library/LaunchDaemons/io.aihop.gpc.plist"
  local tmp_plist
  tmp_plist="$(mktemp -t io.aihop.gpc.XXXXXX)"
  cat >"${tmp_plist}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.aihop.gpc</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/gpc</string>
    <string>--base-dir</string>
    <string>${CONFIG_INSTALL_DIR}</string>
    <string>service</string>
  </array>
  <key>UserName</key>
  <string>${RUNTIME_USER}</string>
  <key>WorkingDirectory</key>
  <string>${CONFIG_INSTALL_DIR}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/tmp/gpc.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/gpc.err.log</string>
</dict>
</plist>
EOF
  run_privileged cp "${tmp_plist}" "${plist}"
  rm -f "${tmp_plist}"
  run_privileged chmod 644 "${plist}"
  run_privileged chown root:wheel "${plist}"
  run_privileged launchctl bootout system "${plist}" >/dev/null 2>&1 || true
  run_privileged launchctl bootstrap system "${plist}"
}

install_service_gopanel_macos() {
  local plist="/Library/LaunchDaemons/io.aihop.gopanel.plist"
  local tmp_plist
  tmp_plist="$(mktemp -t io.aihop.gopanel.XXXXXX)"
  cat >"${tmp_plist}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.aihop.gopanel</string>
  <key>ProgramArguments</key>
  <array>
    <string>${CONFIG_INSTALL_DIR}/gopanel</string>
  </array>
  <key>UserName</key>
  <string>${RUNTIME_USER}</string>
  <key>WorkingDirectory</key>
  <string>${CONFIG_INSTALL_DIR}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/tmp/gopanel.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/gopanel.err.log</string>
</dict>
</plist>
EOF
  run_privileged cp "${tmp_plist}" "${plist}"
  rm -f "${tmp_plist}"
  run_privileged chmod 644 "${plist}"
  run_privileged chown root:wheel "${plist}"
  run_privileged launchctl bootout system "${plist}" >/dev/null 2>&1 || true
  run_privileged launchctl bootstrap system "${plist}"
}

install_service_gpagent_macos() {
  if ! bool_is_true "${CONFIG_INSTALL_GPAGENT}"; then
    log "已跳过 gp-agent 自启动配置。"
    return 0
  fi
  if [ ! -x "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
    warn "未检测到 gp-agent 二进制，跳过 gp-agent launchd 配置"
    return 0
  fi
  local service_name plist tmp_plist
  service_name="$(gpagent_service_name)"
  plist="/Library/LaunchDaemons/${service_name}.plist"
  tmp_plist="$(mktemp -t ${service_name}.XXXXXX)"
  cat >"${tmp_plist}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${service_name}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${CONFIG_INSTALL_DIR}/gp-agent</string>
    <string>service</string>
    <string>--base-dir</string>
    <string>${CONFIG_INSTALL_DIR}</string>
  </array>
  <key>UserName</key>
  <string>${RUNTIME_USER}</string>
  <key>WorkingDirectory</key>
  <string>${CONFIG_INSTALL_DIR}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/tmp/gp-agent.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/gp-agent.err.log</string>
</dict>
</plist>
EOF
  run_privileged cp "${tmp_plist}" "${plist}"
  rm -f "${tmp_plist}"
  run_privileged chmod 644 "${plist}"
  run_privileged chown root:wheel "${plist}"
  run_privileged launchctl bootout system "${plist}" >/dev/null 2>&1 || true
  run_privileged launchctl bootstrap system "${plist}"
}

install_autostart_services() {
  log "写入 gpc/gopanel 开机自启动配置..."
  if [ "$os_name" = "linux" ]; then
    if ! command -v systemctl >/dev/null 2>&1; then
      die "当前 Linux 未检测到 systemd，无法完成自启动配置。"
    fi
    install_service_gpc_linux
    install_service_gopanel_linux
    install_service_gpagent_linux
    
    if [ "${UPDATE_MODE}" = "true" ]; then
      log "检查升级后的服务状态..."
      sleep 3
      if ! systemctl is-active --quiet gopanel.service; then
        warn "GoPanel 升级后启动失败，开始回滚..."
        if [ -f "${CONFIG_INSTALL_DIR}/gopanel.bak" ]; then
          run_privileged cp -f "${CONFIG_INSTALL_DIR}/gopanel.bak" "${CONFIG_INSTALL_DIR}/gopanel"
          run_privileged systemctl restart gopanel.service
          if systemctl is-active --quiet gopanel.service; then
            warn "GoPanel 回滚成功，当前仍运行旧版本。"
          else
            die "GoPanel 回滚后仍然启动失败，请检查 ${CONFIG_INSTALL_DIR} 日志。"
          fi
        else
          die "无法回滚：未找到 gopanel.bak 备份。"
        fi
      fi
      if bool_is_true "${CONFIG_INSTALL_GPAGENT}" && [ -x "${CONFIG_INSTALL_DIR}/gp-agent" ]; then
        if ! systemctl is-active --quiet gp-agent.service; then
          warn "gp-agent 升级后启动失败，开始回滚..."
          if [ -f "${CONFIG_INSTALL_DIR}/gp-agent.bak" ]; then
            run_privileged cp -f "${CONFIG_INSTALL_DIR}/gp-agent.bak" "${CONFIG_INSTALL_DIR}/gp-agent"
            run_privileged systemctl restart gp-agent.service
            warn "gp-agent 回滚完成。"
          else
            warn "无法回滚：未找到 gp-agent.bak 备份。"
          fi
        fi
      fi
    fi

  else
    install_service_gpc_macos
    install_service_gopanel_macos
    install_service_gpagent_macos
  fi
}


install_podman() {
  if command -v podman >/dev/null 2>&1; then
    local pver
    pver="$(podman version --format '{{.Server.Version}}' 2>/dev/null || podman --version 2>/dev/null | grep -oP '\d+\.\d+' | head -1 || echo "0")"
    # Check if >= 5.x
    if echo "$pver" | grep -qE '^[5-9]' 2>/dev/null; then
      log "Podman ${pver} 已安装（版本 >= 5），跳过。"
      ensure_podman_compose || true
      ensure_podman_machine_macos || true
      ensure_podman_socket_access || true
      return 0
    fi
    log "当前 Podman 版本 ${pver}，低于 5.x，尝试升级..."
  fi

  if [ "$os_name" = "darwin" ]; then
    log "开始安装/升级 Podman (macOS)..."
    if command -v brew >/dev/null 2>&1; then
      brew upgrade podman 2>/dev/null || brew install podman || warn "Podman 安装失败，请手动安装。"
      brew install podman-compose || warn "podman-compose 安装失败，请手动安装。"
      ensure_podman_machine_macos || true
    else
      warn "未检测到 Homebrew，请手动安装 Podman 和 podman-compose。"
    fi
    return 0
  fi

  # ---- Linux ----
  log "开始安装 Podman 5.0+..."

  if command -v apt-get >/dev/null 2>&1; then
    install_podman_debian
  elif command -v dnf >/dev/null 2>&1; then
    run_privileged dnf install -y podman
  elif command -v yum >/dev/null 2>&1; then
    run_privileged yum install -y podman
  elif command -v pacman >/dev/null 2>&1; then
    run_privileged pacman -Sy --noconfirm podman
  elif command -v zypper >/dev/null 2>&1; then
    run_privileged zypper --non-interactive install podman
  else
    warn "未识别的 Linux 包管理器，请手动安装 Podman 5.0+。"
  fi

  ensure_podman_compose || true
  ensure_podman_socket_access || true
}

install_podman_debian() {
  log "尝试通过 OBS 仓库安装最新版本 Podman（Debian/Ubuntu）..."

  local distro_id distro_version repo_url=""
  if [ -f /etc/os-release ]; then
    distro_id="$(awk -F= '$1=="ID"{gsub("\"","",$2);print $2}' /etc/os-release | head -n 1)"
    distro_version="$(awk -F= '$1=="VERSION_ID"{gsub("\"","",$2);print $2}' /etc/os-release | head -n 1)"
  fi

  case "${distro_id}" in
    debian)
      case "${distro_version}" in
        12*) repo_url="https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/Debian_12/" ;;
        11*) repo_url="https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/Debian_11/" ;;
      esac
      ;;
    ubuntu)
      case "${distro_version}" in
        24.*) repo_url="https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_24.04/" ;;
        22.04*) repo_url="https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_22.04/" ;;
        20.04*) repo_url="https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_20.04/" ;;
      esac
      ;;
  esac

  if [ -n "${repo_url}" ]; then
    log "添加 Podman OBS 仓库: ${repo_url}"
    run_privileged mkdir -p /etc/apt/sources.list.d /usr/share/keyrings
    local keyring="/usr/share/keyrings/devel_kubic_libcontainers_stable.gpg"
    run_privileged curl -fsSL "${repo_url}/Release.key" | gpg --dearmor | run_privileged tee "${keyring}" >/dev/null 2>&1 || true
    echo "deb [signed-by=${keyring}] ${repo_url} /" | run_privileged tee "/etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list" >/dev/null 2>&1 || true
    apt_update_soft
    run_privileged apt-get install -y podman slirp4netns || {
      warn "通过 OBS 仓库安装 Podman 失败，回退到系统..."
      run_privileged apt-get install -y podman slirp4netns
    }
  else
    warn "未找到匹配的 Debian/Ubuntu 版本 OBS 仓库（${distro_id} ${distro_version}），使用系统仓库..."
    apt_update_soft
    run_privileged apt-get install -y podman slirp4netns
  fi
}
ensure_podman_machine_macos() {
  if [ "$os_name" != "darwin" ]; then
    return 0
  fi
  if ! command -v podman >/dev/null 2>&1; then
    return 0
  fi

  if ! podman machine inspect >/dev/null 2>&1; then
    log "初始化 Podman machine"
    podman machine init || warn "Podman machine 初始化失败，请手动执行: podman machine init"
  fi

  if podman machine inspect >/dev/null 2>&1; then
    if podman machine start >/dev/null 2>&1; then
      log "Podman machine 已启动"
    else
      warn "Podman machine 启动失败，请手动执行: podman machine start"
    fi
  fi
  return 0
}

install_docker() {
  if command -v docker >/dev/null 2>&1; then
    log "Docker 已安装，跳过。"
    return 0
  fi

  if [ "$os_name" = "darwin" ]; then
    if command -v brew >/dev/null 2>&1; then
      brew install --cask docker || warn "Docker Desktop 安装失败，请手动安装。"
    else
      warn "未检测到 Homebrew，请手动安装 Docker Desktop。"
    fi
    return 0
  fi

  local tmp_script
  tmp_script="$(mktemp -t gopanel_docker_install.XXXXXX)"
  if curl -fsSL https://get.docker.com -o "${tmp_script}"; then
    run_privileged /bin/bash "${tmp_script}" || warn "Docker 安装脚本执行失败，请手动安装。"
  else
    warn "下载 Docker 安装脚本失败，请手动安装。"
  fi
  rm -f "${tmp_script}"
}

ensure_podman_socket_access() {
  if [ "$os_name" != "linux" ]; then
    return 0
  fi
  if ! command -v podman >/dev/null 2>&1; then
    return 0
  fi
  if ! command -v systemctl >/dev/null 2>&1; then
    return 0
  fi
  if [ "${RUNTIME_USER}" = "root" ]; then
    # root 场景：启用系统级 podman.socket。不能只依赖 gopanel.service 的 Wants=podman.socket，
    # 否则单独重启 podman 或单元未重建时 socket 不会被拉起。
    log "启用系统级 podman.socket"
    run_privileged systemctl enable --now podman.socket >/dev/null 2>&1 || \
      warn "系统级 podman.socket 启用失败，请手动执行: systemctl enable --now podman.socket"
    return 0
  fi

  # newuidmap/newgidmap is required by rootless Podman for user namespace mapping.
  # Without it, `podman system service` exits with code 125.
  if ! command -v newuidmap >/dev/null 2>&1 || ! command -v newgidmap >/dev/null 2>&1; then
    log "安装 uidmap（提供 newuidmap/newgidmap，rootless Podman 必需）..."
    if command -v apt-get >/dev/null 2>&1; then
      run_privileged env DEBIAN_FRONTEND=noninteractive apt-get install -y uidmap
    elif command -v dnf >/dev/null 2>&1; then
      run_privileged dnf install -y shadow-utils
    elif command -v yum >/dev/null 2>&1; then
      run_privileged yum install -y shadow-utils
    elif command -v zypper >/dev/null 2>&1; then
      run_privileged zypper --non-interactive in -y shadow-utils
    else
      warn "未安装 newuidmap/newgidmap，rootless Podman 功能可能受限。请手动安装 uidmap (Debian/Ubuntu) 或 shadow-utils (RHEL/Fedora/openSUSE)。"
    fi
  fi

  if ! command -v loginctl >/dev/null 2>&1; then
    warn "未检测到 loginctl，无法自动启用 rootless Podman 用户会话，请手动为 ${RUNTIME_USER} 启用 linger 与 podman.socket。"
    return 0
  fi

  local runtime_uid runtime_dir user_home
  runtime_uid="$(runtime_user_uid "${RUNTIME_USER}")"
  runtime_dir="$(runtime_user_runtime_dir "${RUNTIME_USER}")"
  user_home="$(user_home_dir "${RUNTIME_USER}")"
  if [ -z "${runtime_uid}" ] || [ -z "${runtime_dir}" ]; then
    warn "未能解析运行用户 ${RUNTIME_USER} 的 uid/runtime dir，跳过 rootless Podman socket 初始化。"
    return 0
  fi

  log "为用户 ${RUNTIME_USER} 启用 rootless Podman socket"
  run_privileged loginctl enable-linger "${RUNTIME_USER}" >/dev/null 2>&1 || true
  run_privileged mkdir -p "${runtime_dir}" >/dev/null 2>&1 || true
  run_privileged chown "${runtime_uid}:${runtime_uid}" "${runtime_dir}" >/dev/null 2>&1 || true
  run_privileged chmod 0700 "${runtime_dir}" >/dev/null 2>&1 || true
  run_privileged mkdir -p "${user_home}/.config/containers" >/dev/null 2>&1 || true
  printf '[containers]\nlog_driver = "k8s-file"\nshort_name_mode = "permissive"\n' | run_privileged tee "${user_home}/.config/containers/containers.conf" >/dev/null 2>&1 || true
  run_privileged chown -R "${runtime_uid}:${runtime_uid}" "${user_home}/.config" >/dev/null 2>&1 || true

  # enable-linger 拉起 user@UID.service 是异步的，必须等用户级 systemd 就绪，
  # 否则紧接着的 systemctl --user 会因连不上 bus 而静默失败（podman socket 就不会被启用）
  local wait_i
  for wait_i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
    if run_privileged test -S "${runtime_dir}/systemd/private"; then
      break
    fi
    if [ "${wait_i}" = "1" ]; then
      log "等待用户 ${RUNTIME_USER} 的 systemd user manager 启动..."
    fi
    sleep 1
  done
  if ! run_privileged test -S "${runtime_dir}/systemd/private"; then
    warn "用户 ${RUNTIME_USER} 的 systemd user manager 未就绪（${runtime_dir}/systemd/private 不存在），rootless Podman socket 可能无法启用。"
  fi

  # 启用 podman.socket，失败时重试几次（首次启动 user manager 后短暂窗口内仍可能失败）
  local try_i
  for try_i in 1 2 3; do
    run_privileged su -s /bin/sh - "${RUNTIME_USER}" -c "export HOME='${user_home}'; export XDG_RUNTIME_DIR='${runtime_dir}'; export DBUS_SESSION_BUS_ADDRESS='unix:path=${runtime_dir}/bus'; systemctl --user daemon-reload >/dev/null 2>&1 || true; systemctl --user enable --now podman.socket >/dev/null 2>&1 || true; systemctl --user enable podman-restart.service >/dev/null 2>&1 || true"
    if run_privileged test -S "${runtime_dir}/podman/podman.sock"; then
      break
    fi
    sleep 2
  done
  if run_privileged test -S "${runtime_dir}/podman/podman.sock"; then
    log "rootless Podman socket 已就绪: ${runtime_dir}/podman/podman.sock"
  else
    warn "用户 ${RUNTIME_USER} 的 rootless Podman socket 尚未成功创建: ${runtime_dir}/podman/podman.sock"
    warn "非 root 运行面板时将优先使用该用户的 rootless Podman；若 socket 缺失，容器/镜像/流水线可能不可用。"
    warn "请检查 linger、systemd --user 与 podman.socket，必要时手动执行: loginctl enable-linger ${RUNTIME_USER} && su -s /bin/sh - ${RUNTIME_USER} -c 'export HOME=${user_home}; export XDG_RUNTIME_DIR=${runtime_dir}; export DBUS_SESSION_BUS_ADDRESS=unix:path=${runtime_dir}/bus; systemctl --user enable --now podman.socket'"
  fi

  # 确保可通过短名拉取镜像（如 nginx 自动解析为 docker.io/library/nginx）
  local reg_conf="/etc/containers/registries.conf"
  if ! grep -q "unqualified-search-registries" "${reg_conf}" 2>/dev/null; then
    log "配置 Podman 短名镜像源（追加 docker.io 到 ${reg_conf}）..."
    run_privileged mkdir -p /etc/containers
    printf '\n# Added by GoPanel installer\nunqualified-search-registries = ["docker.io"]\n' | \
      run_privileged tee -a "${reg_conf}" >/dev/null 2>&1 || \
      warn "写入 ${reg_conf} 失败，短名拉取镜像可能不可用（可后续在面板中一键修复）。"
  fi

  return 0
}

ensure_podman_compose() {
  if ! command -v podman >/dev/null 2>&1; then
    return 0
  fi
  if [ "$os_name" = "darwin" ]; then
    if command -v podman-compose >/dev/null 2>&1; then
      return 0
    fi
    if podman compose version >/dev/null 2>&1; then
      return 0
    fi
    log "检测到 podman compose 不可用，尝试通过 Homebrew 安装 podman-compose"
    if command -v brew >/dev/null 2>&1; then
      brew install podman-compose || warn "podman-compose 安装失败，请手动执行: brew install podman-compose"
    else
      warn "未检测到 Homebrew，无法自动安装 podman-compose，请手动安装"
    fi
    return 0
  fi
  if [ "$os_name" != "linux" ]; then
    return 0
  fi

  if command -v podman-compose >/dev/null 2>&1; then
    return 0
  fi

  if podman compose version >/dev/null 2>&1; then
    return 0
  fi

  log "检测到 podman compose 不可用，尝试安装 podman-compose"

  if command -v apt-get >/dev/null 2>&1; then
    apt_update_soft
    run_privileged apt-get install -y podman-compose
    return 0
  fi
  if command -v dnf >/dev/null 2>&1; then
    run_privileged dnf install -y podman-compose
    return 0
  fi
  if command -v yum >/dev/null 2>&1; then
    run_privileged yum install -y podman-compose
    return 0
  fi
  if command -v pacman >/dev/null 2>&1; then
    run_privileged pacman -Sy --noconfirm podman-compose
    return 0
  fi
  if command -v zypper >/dev/null 2>&1; then
    run_privileged zypper --non-interactive install podman-compose
    return 0
  fi

  warn "未识别的包管理器，无法自动安装 podman-compose，请手动安装"
  return 0
}

check_container_runtime() {
  if command -v docker >/dev/null 2>&1 || command -v podman >/dev/null 2>&1; then
    log "检测到容器运行时已存在。"
    ensure_podman_compose || true
    ensure_podman_machine_macos || true
    ensure_podman_socket_access || true
    return 0
  fi

  echo "未检测到 Docker 或 Podman。"
  if [ "$RUN_AS_NORMAL_USER" = "true" ]; then
    local choose_podman
    read -r -p "当前 gopanel 以普通用户运行，建议安装 Podman（rootless）。是否现在安装 Podman？[Y/n]: " choose_podman || true
    case "${choose_podman:-Y}" in
      N|n|no|NO)
        warn "已跳过 Podman 安装，请后续手动安装。"
        ;;
      *)
        install_podman
        ;;
    esac
    return 0
  fi

  echo "当前 gopanel 以 root 运行，可选择安装 Docker 或 Podman："
  echo "1) Docker"
  echo "2) Podman"
  echo "3) 跳过"
  local choice
  read -r -p "请输入选项 [1/2/3] (默认: 1): " choice || true
  case "${choice:-1}" in
    1) install_docker ;;
    2) install_podman ;;
    3) warn "已跳过容器运行时安装。" ;;
    *) warn "输入无效，默认安装 Docker。"; install_docker ;;
  esac
}

show_result() {
  local ip_address public_ip
  ip_address="$(detect_server_ip)"
  public_ip=""
  if is_private_ip "${ip_address}"; then
    public_ip="$(detect_public_ip)"
    if [ "${public_ip}" = "${ip_address}" ]; then
      public_ip=""
    fi
  fi

  echo
  echo "GoPanel 安装完成"
  echo "版本: ${version} (code: ${version_code})"
  echo "安装目录: ${CONFIG_INSTALL_DIR}"
  echo "gpc 路径: /usr/local/bin/gpc"
  echo "gopanel 运行用户: ${RUNTIME_USER}"
  if bool_is_true "${CONFIG_INSTALL_GPAGENT}"; then
    echo "gp-agent: 已安装（网站托管、进程守护等能力）"
  else
    echo "gp-agent: 已跳过"
  fi
  echo "用户名: ${CONFIG_USER}"
  echo "密码: ${CONFIG_PASSWORD}"
  if [ -n "${public_ip}" ]; then
    echo "访问地址(公网): http://${public_ip}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
    echo "访问地址(内网): http://${ip_address}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
    echo "提示: 公网访问需在云厂商安全组/防火墙放行 ${CONFIG_PORT} 端口"
  else
    echo "访问地址: http://${ip_address}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
  fi
  echo
}

main() {
  init_invoking_user
  detect_platform
  init_privilege
  ensure_curl
  require_cmds
  detect_preexisting_install
  prompt_update_if_installed

  if [ "${UPDATE_MODE}" != "true" ]; then
    if ! detect_local_package; then
      fetch_upgrade_info
    fi

    echo "GoPanel 安装向导 (版本: ${version}, code: ${version_code})"
    echo "==============================================="

    prompt_basic_config
    prompt_gpagent_install
    choose_gopanel_runtime_user
  else
    echo "GoPanel 升级向导 (目标版本: ${version}, code: ${version_code})"
    echo "==============================================="
    if [ -z "${RUNTIME_USER}" ]; then
      if [ "$os_name" = "darwin" ]; then
        RUNTIME_USER="${INVOKING_USER}"
      else
        RUNTIME_USER="$(stat -c '%U' "${CONFIG_INSTALL_DIR}" 2>/dev/null || echo root)"
      fi
    fi
    if [ "${RUNTIME_USER}" != "root" ]; then
      RUN_AS_NORMAL_USER="true"
    else
      RUN_AS_NORMAL_USER="false"
    fi
  fi

  ensure_install_id
  stop_existing_services
  download_package
  extract_and_find_binaries
  prepare_gpagent_binary
  install_gpc_binary
  install_gopanel_binary
  install_gpagent_binary
  # 先检查/安装容器运行时，再写 init.yaml，避免普通用户场景下
  # 因 Podman 尚未安装导致 docker_sock_path 未写入 rootless 路径。
  check_container_runtime
  write_init_yaml
  ensure_install_dir_owner
  install_autostart_services
  show_result
  if [ "${PREEXISTING_INSTALL}" = "true" ]; then
    track_install_event "upgrade_success" "${version}"
  else
    track_install_event "install_success" ""
  fi
}

main

