#!/bin/bash
set -euo pipefail
trap 'echo "错误：安装脚本在第 ${LINENO} 行中断，命令: ${BASH_COMMAND}" >&2' ERR

# 必要命令检查
for cmd in curl tar openssl; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "错误：缺少必要命令: $cmd"
    exit 1
  fi
done

json_get() {
  local key="$1"
  local json="$2"

  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import json,sys; key=sys.argv[1]; data=json.load(sys.stdin) if sys.stdin.readable() else {}; v=data.get(key, ""); print("" if v is None else v)' "$key" <<<"$json"
    return 0
  fi

  if command -v python >/dev/null 2>&1; then
    python -c 'import json,sys; key=sys.argv[1]; data=json.load(sys.stdin) if sys.stdin.readable() else {}; v=data.get(key, ""); print("" if v is None else v)' "$key" <<<"$json"
    return 0
  fi

  if command -v node >/dev/null 2>&1; then
    node -e 'const fs=require("fs");const key=process.argv[1];const data=JSON.parse(fs.readFileSync(0,"utf8")||"{}");const v=data[key];process.stdout.write((v===undefined||v===null)?"":String(v));' "$key" <<<"$json"
    return 0
  fi

  echo "错误：缺少 JSON 解析工具（python3/python/node 任一即可）"
  exit 1
}

# 定义默认值
CONFIG_INSTALL_DIR="${CONFIG_INSTALL_DIR:-}"
CONFIG_PORT="${CONFIG_PORT:-5470}"
CONFIG_USER="${CONFIG_USER:-admin}"
# 默认密码改为 8 字节十六进制
CONFIG_PASSWORD="${CONFIG_PASSWORD:-$(openssl rand -hex 8)}"
# 安全入口 8 字节 hex
CONFIG_SAFE_ENTER="${CONFIG_SAFE_ENTER:-$(openssl rand -hex 8)}"

# 接收第一个参数作为 APP_BRAND，如果没有则默认为 GoPanel
APP_BRAND="${1:-GoPanel}"

API_UPGRADE_URL="${API_UPGRADE_URL:-https://gopanel.cn/api/panel/upgrade}"

os_name=""
arch_name=""
PACKAGE_NAME=""
PACKAGE_URL=""
package_root=""
version=""
version_code=""

# 1. 检查系统架构
function detect_platform() {
  local u
  u="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$u" in
    linux*) os_name="linux" ;;
    darwin*) os_name="darwin" ;;
    msys*|mingw*|cygwin*)
      echo "检测到 Windows 环境，请使用 quick_start.ps1 进行安装。"
      exit 1
      ;;
    *)
      echo "暂不支持的系统: $u"
      exit 1
      ;;
  esac

  local m
  m="$(uname -m | tr '[:upper:]' '[:lower:]')"
  case "$m" in
    x86_64|amd64) arch_name="amd64" ;;
    arm64|aarch64) arch_name="arm64" ;;
    *)
      echo "暂不支持的架构: $m"
      exit 1
      ;;
  esac

  if [ -z "${CONFIG_INSTALL_DIR}" ]; then
    if [ "$os_name" = "darwin" ]; then
      CONFIG_INSTALL_DIR="${HOME}/.gopanel"
    else
      CONFIG_INSTALL_DIR="/opt/gopanel"
    fi
  fi
}

ensure_privilege_if_needed() {
  if [ "${EUID:-0}" -eq 0 ]; then
    return 0
  fi

  if [ "$os_name" = "linux" ] && [[ "$CONFIG_INSTALL_DIR" == /opt/* || "$CONFIG_INSTALL_DIR" == /usr/* || "$CONFIG_INSTALL_DIR" == /etc/* ]]; then
    echo "错误：安装目录 ${CONFIG_INSTALL_DIR} 需要 root 权限。"
    echo "请使用 sudo 运行，或设置 CONFIG_INSTALL_DIR 到用户目录（例如 ${HOME}/.gopanel）。"
    exit 1
  fi
}

fetch_upgrade_info() {
  local cur_version="${CUR_VERSION:-0.0.0}"
  local cur_version_code="${CUR_VERSION_CODE:-0}"

  local url="${API_UPGRADE_URL}?versionCode=${cur_version_code}&version=${cur_version}&os=${os_name}&arch=${arch_name}&appBrand=${APP_BRAND}"
  echo "检查最新版本..."
  local json
  if ! json="$(curl -fsSL "$url")"; then
    echo "错误：无法获取版本信息，请检查网络或接口地址"
    echo "URL: ${url}"
    exit 1
  fi

  local latest_name latest_code need_update title desc content created_at download_url
  latest_name="$(json_get "latestVersionName" "$json")"
  latest_code="$(json_get "latestVersionCode" "$json")"
  need_update="$(json_get "needUpdate" "$json")"
  title="$(json_get "title" "$json")"
  desc="$(json_get "description" "$json")"
  content="$(json_get "content" "$json")"
  created_at="$(json_get "createdAt" "$json")"
  download_url="$(json_get "downloadUrl" "$json")"

  download_url="$(echo "$download_url" | sed -e 's/`//g' -e 's/^ *//g' -e 's/ *$//g')"

  if [ -z "$latest_name" ] || [ -z "$latest_code" ] || [ -z "$download_url" ]; then
    echo "错误：版本接口返回不完整"
    echo "$json"
    exit 1
  fi

  version="$latest_name"
  version_code="$latest_code"
  PACKAGE_URL="$download_url"
  PACKAGE_NAME="$(basename "$download_url")"

  echo "最新版本: ${latest_name} (code: ${latest_code})"
  if [ -n "$title" ] || [ -n "$desc" ]; then
    echo "${title}"
    echo "${desc}"
  fi
  if [ -n "$created_at" ]; then
    echo "发布时间: ${created_at}"
  fi

  if [ "${need_update}" = "false" ]; then
    echo "当前版本无需更新（needUpdate=false）。如需强制重装，可删除安装目录后重试。"
  fi
}

# 2. 下载安装包
function download_package() {
    echo "检查安装包..."
    if [ ! -f "${PACKAGE_NAME}" ]; then
        echo "开始下载安装包: ${PACKAGE_URL}"
        if ! curl -fSL -o "${PACKAGE_NAME}" "${PACKAGE_URL}"; then
            echo "下载安装包失败，请检查网络或下载地址"
            exit 1
        fi
    else
        echo "安装包已存在，跳过下载"
    fi
}

# 3. 处理安装目录
function handle_install_directory() {
    local src_dir="${1:-.}"
    if [ -d "${CONFIG_INSTALL_DIR}" ]; then
        read -p "检测到安装目录 ${CONFIG_INSTALL_DIR} 已存在，是否删除并重新创建？(y/n): " confirm
        if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
            echo "用户取消操作"
            exit 1
        fi
        rm -rf "${CONFIG_INSTALL_DIR}"
    fi
    mkdir -p "${CONFIG_INSTALL_DIR}"
    echo "开始复制文件到安装目录: ${CONFIG_INSTALL_DIR}"
    cp -a "${src_dir}/." "${CONFIG_INSTALL_DIR}"
    verify_install_artifacts
}

# 4. 解压安装包，到当前目录
tar_filter_warnings() {
  grep -Ev "Ignoring unknown extended header keyword|LIBARCHIVE\.xattr\.com\.apple\.provenance"
}

tar_extract_compat() {
  local package_path="$1"

  if tar --warning=no-unknown-keyword -zxf "${package_path}" 2> >(tar_filter_warnings >&2); then
    return 0
  fi

  if tar --no-xattrs -zxf "${package_path}" 2> >(tar_filter_warnings >&2); then
    return 0
  fi

  tar -zxf "${package_path}" 2> >(tar_filter_warnings >&2)
}

tar_list_compat() {
  local package_path="$1"

  if tar --warning=no-unknown-keyword -tf "${package_path}" 2> >(tar_filter_warnings >&2); then
    return 0
  fi

  if tar --no-xattrs -tf "${package_path}" 2> >(tar_filter_warnings >&2); then
    return 0
  fi

  tar -tf "${package_path}" 2> >(tar_filter_warnings >&2)
}

function extract_package() {
    local package_path=$1
    echo "开始解压安装包..."
    if ! tar_extract_compat "${package_path}"; then
        echo "解压失败，安装包可能损坏"
        rm -f "${package_path}"
        exit 1
    fi

    local expected_root="${PACKAGE_NAME%.tar.gz}"
    expected_root="${expected_root%.tgz}"

    if [ -n "${expected_root}" ] && [ -d "${expected_root}" ]; then
        package_root="${expected_root}"
        echo "解压目录: ${package_root}"
        return 0
    fi

    local dirs=()
    shopt -s nullglob
    for d in */ ; do
        d="${d%/}"
        dirs+=("${d}")
    done
    shopt -u nullglob

    if [ "${#dirs[@]}" -eq 1 ]; then
        package_root="${dirs[0]}"
        echo "解压目录: ${package_root}"
        return 0
    fi

    for d in "${dirs[@]}"; do
        if [ -f "${d}/gopanel" ] || [ -f "${d}/gopanel.exe" ]; then
            package_root="${d}"
            echo "解压目录: ${package_root}"
            return 0
        fi
    done

    if [ "${#dirs[@]}" -gt 0 ]; then
        package_root="${dirs[0]}"
    else
        package_root="."
    fi
    echo "解压目录: ${package_root}"
}


function install_docker(){
    if command -v docker >/dev/null 2>&1; then
        echo "Docker 已安装，跳过"
    else
        local confirm="Y"
        if [ -t 0 ]; then
          printf "未检测到 Docker，是否安装 Docker？[Y/n]: "
          read -r confirm || true
          confirm="${confirm:-Y}"
        fi

        case "${confirm}" in
          Y|y|yes|YES|Yes)
            ;;
          N|n|no|NO|No)
            echo "已跳过 Docker 安装，继续后续流程"
            return 0
            ;;
          *)
            echo "输入无效，默认按 Yes 继续安装 Docker"
            ;;
        esac

        local uname_s=""
        uname_s="$(uname -s 2>/dev/null || true)"

        if [ "${uname_s}" = "Darwin" ]; then
          echo "检测到 macOS：将引导安装 Docker Desktop（不影响后续流程）"
          if command -v brew >/dev/null 2>&1; then
            local confirm_brew="Y"
            if [ -t 0 ]; then
              printf "是否尝试使用 Homebrew 安装 Docker Desktop？[Y/n]: "
              read -r confirm_brew || true
              confirm_brew="${confirm_brew:-Y}"
            fi
            case "${confirm_brew}" in
              N|n|no|NO|No)
                echo "已跳过自动安装，请手动安装 Docker Desktop：https://www.docker.com/products/docker-desktop/"
                return 0
                ;;
            esac
            brew install --cask docker || { echo "Docker Desktop 安装失败（将继续后续流程）"; return 0; }
            echo "Docker Desktop 已安装，请手动启动一次 Docker.app 完成初始化"
            return 0
          fi
          echo "未检测到 Homebrew，请手动安装 Docker Desktop：https://www.docker.com/products/docker-desktop/"
          return 0
        fi

        case "${uname_s}" in
          MINGW*|MSYS*|CYGWIN*)
            echo "检测到 Windows 环境：请安装 Docker Desktop 并开启 WSL2 集成：https://www.docker.com/products/docker-desktop/"
            echo "已跳过 Docker 安装，继续后续流程"
            return 0
            ;;
        esac

        echo "开始执行 Docker 安装脚本..."
        local script_url="https://gopanel.run/install_docker.sh"
        local tmp_script=""

        if command -v mktemp >/dev/null 2>&1; then
          tmp_script="$(mktemp -t gopanel_install_docker.XXXXXX)"
        else
          tmp_script="/tmp/gopanel_install_docker.$$"
        fi

        if command -v curl >/dev/null 2>&1; then
          curl -fsSL "${script_url}" -o "${tmp_script}" || { echo "Docker 安装脚本下载失败（将继续后续流程）"; rm -f "${tmp_script}"; return 0; }
        elif command -v wget >/dev/null 2>&1; then
          wget -qO "${tmp_script}" "${script_url}" || { echo "Docker 安装脚本下载失败（将继续后续流程）"; rm -f "${tmp_script}"; return 0; }
        else
          echo "错误：未找到 curl 或 wget，无法下载 Docker 安装脚本"
          rm -f "${tmp_script}"
          echo "已跳过 Docker 安装，继续后续流程"
          return 0
        fi

        chmod +x "${tmp_script}" || true
        /bin/bash "${tmp_script}" || { echo "Docker 安装失败（将继续后续流程）"; rm -f "${tmp_script}"; return 0; }
        rm -f "${tmp_script}"
    fi
}

# 写入到系统环境变量中，方便通过 gopanel 指令直接执行
function write_env(){
    local install_dir=$1
    echo "export GOPANEL_HOME=${install_dir}" >> ~/.bashrc
    echo "export PATH=\$PATH:\$GOPANEL_HOME" >> ~/.bashrc
    # 不强制 source，提示用户重启 shell
    echo "已写入 ~/.bashrc，请重启 shell 以生效"
}

# 配置服务开机自启
function set_gopanel_service(){
    local install_dir=$1
    local service_file=""
    if ! command -v systemctl >/dev/null 2>&1; then
      return 1
    fi
    if [ "${EUID:-0}" -ne 0 ]; then
      return 1
    fi

    cat >/etc/systemd/system/gopanel.service <<EOF
[Unit]
Description=GoPanel
After=network.target

[Service]
Type=simple
WorkingDirectory=${install_dir}
ExecStart=${install_dir}/gopanel
Restart=always
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable gopanel
    systemctl restart gopanel
}

verify_install_artifacts() {
    if [ ! -d "${CONFIG_INSTALL_DIR}" ]; then
        echo "错误：安装目录不存在: ${CONFIG_INSTALL_DIR}"
        exit 1
    fi

    local bin_path=""
    if [ -f "${CONFIG_INSTALL_DIR}/gopanel" ]; then
        bin_path="${CONFIG_INSTALL_DIR}/gopanel"
        chmod +x "${bin_path}" || true
    elif [ -f "${CONFIG_INSTALL_DIR}/gopanel.exe" ]; then
        bin_path="${CONFIG_INSTALL_DIR}/gopanel.exe"
    else
        echo "错误：复制后未检测到可执行文件（gopanel / gopanel.exe）"
        exit 1
    fi

    local file_count
    file_count="$(find "${CONFIG_INSTALL_DIR}" -mindepth 1 | wc -l | tr -d ' ')"
    if [ "${file_count}" -le 0 ]; then
        echo "错误：安装目录为空，文件复制失败"
        exit 1
    fi

    echo "安装文件校验通过: ${bin_path}"
}

verify_started() {
    local max_retry=15
    local i=1
    while [ "${i}" -le "${max_retry}" ]; do
        if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet gopanel; then
            echo "启动校验通过：systemd 服务运行中"
            return 0
        fi
        if pgrep -f "${CONFIG_INSTALL_DIR}/gopanel" >/dev/null 2>&1; then
            echo "启动校验通过：进程运行中"
            return 0
        fi
        sleep 1
        i=$((i+1))
    done

    echo "错误：启动后未检测到 gopanel 运行进程"
    echo "可检查日志: /tmp/gopanel.log 或 journalctl -u gopanel -n 100 --no-pager"
    exit 1
}

#  尝试获取本机公网ip
function try_get_net_ip(){
    local services=( "https://ifconfig.me" "https://icanhazip.com" "https://ident.me" "https://ipinfo.io/ip" "https://api.ipify.org" "https://ifconfig.co" )
    local external_ip=""
    for service in "${services[@]}"; do
        ip=$(curl -s --connect-timeout 3 "$service" || true)
        if [[ "$ip" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            external_ip="$ip"
            break
        fi
    done
    if [ -n "$external_ip" ]; then
        echo "外部访问地址 http://${external_ip}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
    else
        echo "无法获取公网 IP"
    fi
}

# run gopanel
function run_gopanel(){
    echo "正在启动 GoPanel"
    if [ "$os_name" = "linux" ]; then
      if set_gopanel_service "${CONFIG_INSTALL_DIR}"; then
        echo "GoPanel 已通过 systemd 启动"
      else
        echo "未检测到 systemd 或权限不足，将以前台方式启动（可自行用守护进程接管）"
        nohup "${CONFIG_INSTALL_DIR}/gopanel" >/tmp/gopanel.log 2>&1 &
      fi
    else
      nohup "${CONFIG_INSTALL_DIR}/gopanel" >/tmp/gopanel.log 2>&1 &
    fi
    verify_started
    echo "GoPanel 已启动"
    echo "用户名：${CONFIG_USER}"
    echo "密码：${CONFIG_PASSWORD}"
    ip_address="$(hostname -I 2>/dev/null | awk '{print $1}' 2>/dev/null || hostname -I 2>/dev/null || hostname)"
    echo "内网访问地址 http://${ip_address}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
    try_get_net_ip
}


function set_init_conf(){
    local init_target_dir="${1:-.}"
    # 获取安装目录
    while true; do
        read -e -p "请输入安装目录 (默认: ${CONFIG_INSTALL_DIR}): " input_dir
        # 如果用户未输入，使用默认值
        if [ -z "$input_dir" ]; then
            input_dir=${CONFIG_INSTALL_DIR}
        fi
        # 验证路径是否为绝对路径且不包含特殊字符和空格
        if [[ ! "$input_dir" =~ ^/ ]]; then
            echo "错误：安装目录必须是绝对路径，请重新输入。"
        elif [[ "$input_dir" =~ [[:space:]] ]]; then
            echo "错误：安装目录不能包含空格，请重新输入。"
        elif [[ "$input_dir" =~ [^a-zA-Z0-9_/\.-] ]]; then
            echo "错误：安装目录包含不允许的特殊字符，请重新输入。"
        else
            break
        fi
    done
    CONFIG_INSTALL_DIR="$input_dir"
    while true; do
        read -e -p "请输入端口 (默认: ${CONFIG_PORT}): " input_port
        # 如果用户未输入，使用默认值
        if [ -z "$input_port" ]; then
            CONFIG_PORT=${CONFIG_PORT}
            break
        # 检查是否为数字且在有效范围内(1-65535)
        elif [[ "$input_port" =~ ^[0-9]+$ ]] && [ "$input_port" -ge 1 ] && [ "$input_port" -le 65535 ]; then
            CONFIG_PORT=$input_port
            break
        else
            echo "错误：端口必须是1-65535之间的整数，请重新输入。"
        fi
    done
    while true; do
        read -e -p "请设置登录用户名 (默认: ${CONFIG_USER}): " input_user
        input_user=${input_user:-${CONFIG_USER}}
        if [[ "$input_user" =~ [[:space:]] ]]; then
            echo "错误：用户名不能包含空格，请重新输入。"
        elif [[ "$input_user" =~ [^a-zA-Z0-9_-] ]]; then
            echo "错误：用户名只能包含字母、数字、下划线和连字符，请重新输入。"
        else
            CONFIG_USER=$input_user
            break
        fi
    done
    while true; do
        read -e -p "请设置登录密码 (默认: ${CONFIG_PASSWORD}): " input_password
        input_password=${input_password:-${CONFIG_PASSWORD}}
        if [[ "$input_password" =~ [[:space:]] ]]; then
            echo "错误：密码不能包含空格，请重新输入。"
        else
            CONFIG_PASSWORD=$input_password
            break
        fi
    done
    while true; do
        read -e -p "请设置安全入口 (默认: ${CONFIG_SAFE_ENTER}): " input_safe_enter
        input_safe_enter=${input_safe_enter:-${CONFIG_SAFE_ENTER}}
        if [[ "$input_safe_enter" =~ [[:space:]] ]]; then
            echo "错误：安全入口不能包含空格，请重新输入。"
        elif [[ "$input_safe_enter" =~ [^a-zA-Z0-9] ]]; then
            echo "错误：安全入口只能包含字母、数字，请重新输入。"
        else
            CONFIG_SAFE_ENTER=$input_safe_enter
            break
        fi
    done

    # 生成 init.yaml 到当前安装源目录根部，避免出现嵌套目录
    mkdir -p "${init_target_dir}"
    cat <<EOF > "${init_target_dir}/init.yaml"
base_dir: "${CONFIG_INSTALL_DIR}"
port: ${CONFIG_PORT}
user: "${CONFIG_USER}"
password: "${CONFIG_PASSWORD}"
safe_enter: "${CONFIG_SAFE_ENTER}"
EOF
}


main() {
    detect_platform
    ensure_privilege_if_needed
    fetch_upgrade_info

    printf "GoPanel 安装向导 (版本: ${version}, code: ${version_code})"
    echo "=============================="


     # 如果安装目录已存在，直接使用该目录（不覆盖）
    if [ -d "${package_root}" ]; then
        cd "${package_root}" || { echo "无法进入 ${package_root}"; exit 1; }

        # 安装 docker
        install_docker

        # 初始化配置
        set_init_conf "."

        # 处理安装目录，然后将项目文件复制到安装目录
        handle_install_directory "."

        run_gopanel
        echo "使用已存在的安装目录完成启动"
        return 0
    fi

    # 安装目录不存在，优先使用本地安装包（如果存在），否则下载安装包
    if [ -f "${PACKAGE_NAME}" ]; then
        echo "发现本地安装包: ${PACKAGE_NAME}，将使用本地包进行安装"
    else
        # 下载安装包
        download_package
    fi

    # 解压安装包到当前目录
    extract_package "${PACKAGE_NAME}"

    echo "切换到解压目录: ${package_root}"
    cd "${package_root}"

    if [ "$os_name" = "linux" ]; then
      install_docker
    fi

    # 初始化配置
    set_init_conf "."

    # 处理安装目录，然后将项目文件复制到安装目录
    handle_install_directory "."

    # 运行 gopanel
    run_gopanel

    echo "GoPanel 安装完成！"
}

# 启动主流程
main
