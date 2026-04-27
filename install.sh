#!/bin/bash
set -euo pipefail
trap 'echo "错误：安装脚本在第 ${LINENO} 行中断，命令: ${BASH_COMMAND}" >&2' ERR

# ---- 基础变量 ----
APP_BRAND="${1:-GoPanel}"
API_UPGRADE_URL="${API_UPGRADE_URL:-https://gopanel.cn/api/panel/upgrade}"
API_TRACK_URL="${API_TRACK_URL:-https://gopanel.cn/api/panel/installs/track}"
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
    run_privileged env DEBIAN_FRONTEND=noninteractive apt-get update -y
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

json_get() {
  local key="$1"
  local json="$2"

  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys
key=sys.argv[1]
try:
    data=json.loads(sys.stdin.read() or "{}")
except Exception:
    sys.exit(1)
v=data.get(key, "")
print("" if v is None else v)' "$key" <<<"$json"
    return 0
  fi
  if command -v python >/dev/null 2>&1; then
    python -c 'import json,sys
key=sys.argv[1]
try:
    data=json.loads(sys.stdin.read() or "{}")
except Exception:
    sys.exit(1)
v=data.get(key, "")
print("" if v is None else v)' "$key" <<<"$json"
    return 0
  fi
  if command -v node >/dev/null 2>&1; then
    node -e 'const fs=require("fs");const key=process.argv[1];let data={};try{data=JSON.parse(fs.readFileSync(0,"utf8")||"{}")}catch(e){process.exit(1)}const v=data[key];process.stdout.write((v===undefined||v===null)?"":String(v));' "$key" <<<"$json"
    return 0
  fi
  die "缺少 JSON 解析工具（python3/python/node 任一即可）"
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

fetch_upgrade_info() {
  local cur_version="${CUR_VERSION:-0.0.0}"
  local cur_version_code="${CUR_VERSION_CODE:-0}"
  local url
  url="${API_UPGRADE_URL}?versionCode=${cur_version_code}&version=${cur_version}&os=${os_name}&arch=${arch_name}&appBrand=${APP_BRAND}"

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
  PACKAGE_URL="$download_url"
  PACKAGE_NAME="$(basename "$download_url")"

  log "最新版本: ${version} (code: ${version_code})"
}

fetch_gpagent_upgrade_info() {
  local cur_version="${CUR_VERSION:-0.0.0}"
  local cur_version_code="${CUR_VERSION_CODE:-0}"
  local url
  url="${API_UPGRADE_URL}?versionCode=${cur_version_code}&version=${cur_version}&os=${os_name}&arch=${arch_name}&appBrand=${APP_BRAND}&package=gp-agent"

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
  GPAGENT_PACKAGE_URL="$download_url"
  GPAGENT_PACKAGE_NAME="$(basename "$download_url")"
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
      run_privileged systemctl stop gp-agent.service >/dev/null 2>&1 || true
    fi
  fi
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
  run_privileged mkdir -p /usr/local/bin
  run_privileged cp "${BIN_GPC_PATH}" /usr/local/bin/gpc
  run_privileged chmod 755 /usr/local/bin/gpc
}

install_gopanel_binary() {
  log "安装 gopanel 到 ${CONFIG_INSTALL_DIR}"
  run_privileged mkdir -p "${CONFIG_INSTALL_DIR}"
  run_privileged cp "${BIN_GOPANEL_PATH}" "${CONFIG_INSTALL_DIR}/gopanel"
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
  fi
  log "安装 gp-agent 到 ${CONFIG_INSTALL_DIR}/gp-agent"
  run_privileged mkdir -p "${CONFIG_INSTALL_DIR}"
  run_privileged cp "${BIN_GPAGENT_PATH}" "${CONFIG_INSTALL_DIR}/gp-agent"
  run_privileged chmod 755 "${CONFIG_INSTALL_DIR}/gp-agent"
}

write_init_yaml() {
  if [ "${UPDATE_MODE}" = "true" ] && [ -f "${CONFIG_INSTALL_DIR}/conf.yaml" ]; then
    log "升级模式：保留现有 conf.yaml/init.yaml，不重写初始化配置。"
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
  if [ "${os_name}" = "linux" ] && [ "${RUN_AS_NORMAL_USER}" = "true" ] && command -v podman >/dev/null 2>&1; then
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

create_linux_user_if_needed() {
  local username="$1"
  if id "$username" >/dev/null 2>&1; then
    log "用户 ${username} 已存在，直接复用。"
    return 0
  fi

  if run_privileged test -x /usr/sbin/nologin; then
    run_privileged useradd --system --create-home --shell /usr/sbin/nologin "$username"
  else
    run_privileged useradd --system --create-home --shell /bin/bash "$username"
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

install_autostart_services() {
  log "写入 gpc/gopanel 开机自启动配置..."
  if [ "$os_name" = "linux" ]; then
    if ! command -v systemctl >/dev/null 2>&1; then
      die "当前 Linux 未检测到 systemd，无法完成自启动配置。"
    fi
    install_service_gpc_linux
    install_service_gopanel_linux
    install_service_gpagent_linux
  else
    install_service_gpc_macos
    install_service_gopanel_macos
  fi
}

install_podman() {
  if command -v podman >/dev/null 2>&1; then
    log "Podman 已安装，跳过。"
    return 0
  fi

  log "开始安装 Podman..."
  if [ "$os_name" = "darwin" ]; then
    if command -v brew >/dev/null 2>&1; then
      brew install podman || warn "Podman 安装失败，请手动安装。"
    else
      warn "未检测到 Homebrew，请手动安装 Podman。"
    fi
    return 0
  fi

  ensure_gnupg_for_apt() {
    if command -v gpg >/dev/null 2>&1; then
      return 0
    fi
    run_privileged apt-get update
    run_privileged apt-get install -y gnupg ca-certificates
  }

  read_os_release_value() {
    local key="$1"
    local file="/etc/os-release"
    if [ ! -f "${file}" ]; then
      echo ""
      return 0
    fi
    awk -F= -v k="${key}" '
      $1==k {
        v=$2
        gsub(/^"/,"",v); gsub(/"$/,"",v)
        print v
        exit
      }
    ' "${file}" 2>/dev/null || true
  }

  configure_podman_repo_apt_latest() {
    local id version_id repo_dist repo_url key_url keyring list_file
    id="$(read_os_release_value ID)"
    version_id="$(read_os_release_value VERSION_ID)"
    id="$(echo "${id}" | tr '[:upper:]' '[:lower:]')"

    if [ -z "${version_id}" ]; then
      warn "无法识别系统版本（/etc/os-release VERSION_ID 为空），将尝试直接 apt 安装 Podman。"
      return 1
    fi

    case "${id}" in
      debian)
        local major
        major="${version_id%%.*}"
        if [[ "${major}" =~ ^[0-9]+$ ]] && [ "${major}" -ge 13 ]; then
          warn "检测到 Debian ${version_id}，将跳过 devel:kubic:libcontainers:stable 源（该源可能缺少 Debian_${version_id}），改为直接使用 Debian 官方仓库安装 Podman。"
          return 1
        fi
        repo_dist="Debian_${version_id}"
        ;;
      ubuntu)
        repo_dist="xUbuntu_${version_id}"
        ;;
      *)
        warn "当前发行版 ${id} 未配置 Podman 官方新版本源，将尝试直接 apt 安装。"
        return 1
        ;;
    esac

    repo_url="https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/${repo_dist}/"
    key_url="${repo_url}Release.key"
    keyring="/etc/apt/keyrings/libcontainers-archive-keyring.gpg"
    list_file="/etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list"

    run_privileged mkdir -p /etc/apt/keyrings
    local tmpkey
    tmpkey="$(mktemp -t libcontainers_key.XXXXXX)"
    if ! curl -fsSL "${key_url}" -o "${tmpkey}"; then
      rm -f "${tmpkey}"
      warn "下载 Podman 源密钥失败（将回退到官方仓库安装）: ${key_url}"
      return 1
    fi
    if ! curl -fsSL "${repo_url}Release" -o /dev/null; then
      rm -f "${tmpkey}"
      warn "Podman 源不存在或不可用（将回退到官方仓库安装）: ${repo_url}"
      return 1
    fi
    run_privileged gpg --dearmor -o "${keyring}" "${tmpkey}"
    rm -f "${tmpkey}"
    run_privileged chmod 644 "${keyring}"
    printf "deb [signed-by=%s] %s /\n" "${keyring}" "${repo_url}" | run_privileged tee "${list_file}" >/dev/null
    return 0
  }

  if command -v apt-get >/dev/null 2>&1; then
    ensure_gnupg_for_apt
    if configure_podman_repo_apt_latest; then
      run_privileged apt-get update
      run_privileged apt-get install -y podman
    else
      run_privileged apt-get update
      run_privileged apt-get install -y podman
    fi
  elif command -v dnf >/dev/null 2>&1; then
    run_privileged dnf install -y podman
  elif command -v yum >/dev/null 2>&1; then
    run_privileged yum install -y podman
  elif command -v pacman >/dev/null 2>&1; then
    run_privileged pacman -Sy --noconfirm podman
  elif command -v zypper >/dev/null 2>&1; then
    run_privileged zypper --non-interactive install podman
  else
    warn "未识别的 Linux 包管理器，请手动安装 Podman。"
  fi

  ensure_podman_compose || true
  ensure_podman_socket_access || true
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
    return 0
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
  printf '[containers]\nlog_driver = "k8s-file"\n' | run_privileged tee "${user_home}/.config/containers/containers.conf" >/dev/null 2>&1 || true
  run_privileged chown -R "${runtime_uid}:${runtime_uid}" "${user_home}/.config" >/dev/null 2>&1 || true
  run_privileged su -s /bin/sh - "${RUNTIME_USER}" -c "export HOME='${user_home}'; export XDG_RUNTIME_DIR='${runtime_dir}'; export DBUS_SESSION_BUS_ADDRESS='unix:path=${runtime_dir}/bus'; systemctl --user daemon-reload >/dev/null 2>&1 || true; systemctl --user enable --now podman.socket >/dev/null 2>&1 || true"
  if ! run_privileged test -S "${runtime_dir}/podman/podman.sock"; then
    warn "用户 ${RUNTIME_USER} 的 rootless Podman socket 尚未成功创建: ${runtime_dir}/podman/podman.sock"
    warn "非 root 运行面板时将优先使用该用户的 rootless Podman；若 socket 缺失，容器/镜像/流水线可能不可用。"
    warn "请检查 linger、systemd --user 与 podman.socket，必要时手动执行: loginctl enable-linger ${RUNTIME_USER} && su -s /bin/sh - ${RUNTIME_USER} -c 'export HOME=${user_home}; export XDG_RUNTIME_DIR=${runtime_dir}; export DBUS_SESSION_BUS_ADDRESS=unix:path=${runtime_dir}/bus; systemctl --user enable --now podman.socket'"
  fi
  return 0
}

ensure_podman_compose() {
  if [ "$os_name" != "linux" ]; then
    return 0
  fi
  if ! command -v podman >/dev/null 2>&1; then
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
    run_privileged apt-get update
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
  local ip_address
  ip_address="$(detect_server_ip)"

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
  echo "访问地址: http://${ip_address}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
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
  fi

  ensure_install_id
  stop_existing_services
  download_package
  extract_and_find_binaries
  prepare_gpagent_binary
  install_gpc_binary
  install_gopanel_binary
  install_gpagent_binary
  write_init_yaml
  ensure_install_dir_owner
  install_autostart_services
  check_container_runtime
  show_result
  if [ "${PREEXISTING_INSTALL}" = "true" ]; then
    track_install_event "upgrade_success" "${version}"
  else
    track_install_event "install_success" ""
  fi
}

main
