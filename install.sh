#!/bin/bash
set -euo pipefail
trap 'echo "错误：安装脚本在第 ${LINENO} 行中断，命令: ${BASH_COMMAND}" >&2' ERR

# ---- 基础变量 ----
APP_BRAND="${1:-GoPanel}"
API_UPGRADE_URL="${API_UPGRADE_URL:-https://gopanel.cn/api/panel/upgrade}"
CONFIG_INSTALL_DIR="${CONFIG_INSTALL_DIR:-}"
CONFIG_PORT="${CONFIG_PORT:-5470}"
CONFIG_USER="${CONFIG_USER:-admin}"
CONFIG_PASSWORD="${CONFIG_PASSWORD:-$(openssl rand -hex 8)}"
CONFIG_SAFE_ENTER="${CONFIG_SAFE_ENTER:-$(openssl rand -hex 8)}"
RUNTIME_USER=""
RUN_AS_NORMAL_USER="false"
INVOKING_USER=""

os_name=""
arch_name=""
version=""
version_code=""
PACKAGE_URL=""
PACKAGE_NAME=""
SUDO_CMD=""
WORK_DIR=""
BIN_GPC_PATH=""
BIN_GOPANEL_PATH=""

log() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*" >&2; }
die() { echo "[ERROR] $*" >&2; exit 1; }

cleanup() {
  if [ -n "${WORK_DIR}" ] && [ -d "${WORK_DIR}" ]; then
    rm -rf "${WORK_DIR}"
  fi
}
trap cleanup EXIT

json_get() {
  local key="$1"
  local json="$2"

  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; key=sys.argv[1]; data=json.loads(sys.stdin.read() or "{}"); v=data.get(key, ""); print("" if v is None else v)' "$key" <<<"$json"
    return 0
  fi
  if command -v python >/dev/null 2>&1; then
    python -c 'import json,sys; key=sys.argv[1]; data=json.loads(sys.stdin.read() or "{}"); v=data.get(key, ""); print("" if v is None else v)' "$key" <<<"$json"
    return 0
  fi
  if command -v node >/dev/null 2>&1; then
    node -e 'const fs=require("fs");const key=process.argv[1];const data=JSON.parse(fs.readFileSync(0,"utf8")||"{}");const v=data[key];process.stdout.write((v===undefined||v===null)?"":String(v));' "$key" <<<"$json"
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
    msys*|mingw*|cygwin*) die "检测到 Windows 环境，请使用 quick_start.ps1 进行安装。" ;;
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
  latest_name="$(json_get "latestVersionName" "$json")"
  latest_code="$(json_get "latestVersionCode" "$json")"
  download_url="$(json_get "downloadUrl" "$json")"
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

download_package() {
  if [ -f "${PACKAGE_NAME}" ]; then
    log "发现本地安装包: ${PACKAGE_NAME}，跳过下载。"
    return 0
  fi
  log "开始下载安装包: ${PACKAGE_NAME} - ${PACKAGE_URL}"
  curl -fSL -o "${PACKAGE_NAME}" "${PACKAGE_URL}" || die "下载安装包失败"
}

extract_and_find_binaries() {
  WORK_DIR="$(mktemp -d -t gopanel_install.XXXXXX)"
  log "解压安装包到临时目录: ${WORK_DIR}"
  tar -zxf "${PACKAGE_NAME}" -C "${WORK_DIR}" || die "解压失败，安装包可能损坏"

  BIN_GPC_PATH="$(find "${WORK_DIR}" -type f -name gpc | head -n 1)"
  BIN_GOPANEL_PATH="$(find "${WORK_DIR}" -type f -name gopanel | head -n 1)"

  [ -n "${BIN_GPC_PATH}" ] || die "安装包中未找到 gpc 二进制文件"
  [ -n "${BIN_GOPANEL_PATH}" ] || die "安装包中未找到 gopanel 二进制文件"
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

write_init_yaml() {
  local tmp_file
  tmp_file="$(mktemp -t gopanel_init.XXXXXX)"
  cat >"${tmp_file}" <<EOF
base_dir: "${CONFIG_INSTALL_DIR}"
port: ${CONFIG_PORT}
user: "${CONFIG_USER}"
password: "${CONFIG_PASSWORD}"
safe_enter: "${CONFIG_SAFE_ENTER}"
EOF
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
    run_privileged chown -R "${RUNTIME_USER}" "${CONFIG_INSTALL_DIR}"
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
ExecStart=/usr/local/bin/gpc --base-dir ${CONFIG_INSTALL_DIR}
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
  tmp_service="$(mktemp -t gopanel.service.XXXXXX)"
  cat >"${tmp_service}" <<EOF
[Unit]
Description=GoPanel
After=network.target gpc.service
Requires=gpc.service

[Service]
Type=simple
User=${RUNTIME_USER}
WorkingDirectory=${CONFIG_INSTALL_DIR}
ExecStart=${CONFIG_INSTALL_DIR}/gopanel
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
  run_privileged cp "${tmp_service}" /etc/systemd/system/gopanel.service
  rm -f "${tmp_service}"
  run_privileged systemctl daemon-reload
  run_privileged systemctl enable gopanel.service
  run_privileged systemctl restart gopanel.service
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

  if command -v apt-get >/dev/null 2>&1; then
    run_privileged apt-get update
    run_privileged apt-get install -y podman
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

check_container_runtime() {
  if command -v docker >/dev/null 2>&1 || command -v podman >/dev/null 2>&1; then
    log "检测到容器运行时已存在，跳过安装。"
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
  ip_address="$(hostname -I 2>/dev/null | awk '{print $1}' 2>/dev/null || true)"
  if [ -z "${ip_address}" ]; then
    ip_address="$(hostname 2>/dev/null || echo "127.0.0.1")"
  fi

  echo
  echo "GoPanel 安装完成"
  echo "版本: ${version} (code: ${version_code})"
  echo "安装目录: ${CONFIG_INSTALL_DIR}"
  echo "gpc 路径: /usr/local/bin/gpc"
  echo "gopanel 运行用户: ${RUNTIME_USER}"
  echo "用户名: ${CONFIG_USER}"
  echo "密码: ${CONFIG_PASSWORD}"
  echo "访问地址: http://${ip_address}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
  echo
}

main() {
  require_cmds
  init_invoking_user
  detect_platform
  init_privilege
  fetch_upgrade_info

  echo "GoPanel 安装向导 (版本: ${version}, code: ${version_code})"
  echo "==============================================="

  prompt_basic_config
  choose_gopanel_runtime_user
  download_package
  extract_and_find_binaries
  install_gpc_binary
  install_gopanel_binary
  write_init_yaml
  ensure_install_dir_owner
  install_autostart_services
  check_container_runtime
  show_result
}

main
