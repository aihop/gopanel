#!/bin/bash
set -euo pipefail

# ==========================================
# GoPanel 自动化安装脚本 (基于 GitHub Releases)
# ==========================================

# 仓库配置 (替换为你的真实 GitHub 用户名和仓库名)
GITHUB_REPO="aihop/gopanel"

# 基础命令检查
for cmd in curl tar openssl grep awk; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "错误：缺少必要命令: $cmd"
    exit 1
  fi
done

# 权限检查
if [ $EUID -ne 0 ]; then
    echo "错误：此脚本需要以 root 权限执行。请使用 sudo 或切换到 root 用户后重试。"
    exit 1
fi

# ==========================================
# 1. 识别操作系统与架构，并合理分配安装目录
# ==========================================
_uname_s="$(uname -s)"
_uname_m="$(uname -m)"

case "${_uname_s}" in
  Linux)
    OS=linux
    DEFAULT_INSTALL_DIR="/opt/gopanel"
    ;;
  Darwin)
    OS=darwin
    # macOS 下推荐安装在个人目录下的隐藏文件夹中，与后端配置保持一致
    DEFAULT_INSTALL_DIR="${HOME}/.gopanel"
    ;;
  CYGWIN*|MINGW*|MSYS*)
    OS=windows
    # Windows 下也保持与后端一致，安装在用户目录
    DEFAULT_INSTALL_DIR="${HOME}/.gopanel"
    ;;
  *)
    echo "暂不支持的操作系统: ${_uname_s}"
    exit 1
    ;;
esac

case "${_uname_m}" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  armv7l|armv7) ARCH=armv7 ;;
  i386|i686) ARCH=386 ;;
  *)
    echo "暂不支持的系统架构: ${_uname_m}"
    exit 1
    ;;
esac

# ==========================================
# 全局变量定义
# ==========================================
VERSION="${1:-latest}"
CONFIG_INSTALL_DIR="${CONFIG_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
CONFIG_PORT="${CONFIG_PORT:-5470}"
CONFIG_USER="${CONFIG_USER:-admin}"
CONFIG_PASSWORD="${CONFIG_PASSWORD:-$(openssl rand -hex 8)}"
CONFIG_SAFE_ENTER="${CONFIG_SAFE_ENTER:-$(openssl rand -hex 8)}"

TMPDIR="$(mktemp -d "${TMPDIR:-/tmp}/gopanel_install.XXXX")"
cleanup() { rm -rf "${TMPDIR}"; }
trap cleanup EXIT

# ==========================================
# 2. 获取 GitHub Releases 下载信息
# ==========================================
function fetch_download_info() {
    echo "正在从 GitHub 获取版本信息..."
    
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    if [ "${VERSION}" != "latest" ]; then
        api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${VERSION}"
    fi

    local resp
    resp=$(curl -sS "$api_url") || { echo "请求 GitHub API 失败"; return 1; }

    # 检查是否返回了 404 或限流错误
    if echo "$resp" | grep -q '"message": "Not Found"'; then
        echo "错误：未找到对应的版本 ${VERSION}，请检查版本号或仓库是否公开。"
        return 1
    fi

    # 提取版本号 (tag_name)
    REMOTE_VER=$(echo "$resp" | grep -oP '"tag_name": "\K(.*)(?=")')
    if [ -z "$REMOTE_VER" ]; then
        echo "解析版本号失败，可能是 API 达到了请求上限。"
        return 1
    fi

    # 拼接预期的安装包名称
    PACKAGE_NAME="gopanel-${OS}-${ARCH}.tar.gz"
    
    # 从 assets 数组中提取对应压缩包的下载地址
    PACKAGE_URL=$(echo "$resp" | grep -oP '"browser_download_url": "\K(.*'${PACKAGE_NAME}')(?=")')
    
    if [ -z "$PACKAGE_URL" ]; then
        echo "错误：在 Release ${REMOTE_VER} 中未找到匹配你系统架构的安装包 (${PACKAGE_NAME})。"
        echo "请检查仓库是否上传了该文件。"
        return 1
    fi

    # 可选：如果服务器在国内，可以使用 ghproxy 加速下载
    read -p "是否使用镜像加速下载 (国内服务器建议选 y)? (y/n): " use_proxy
    if [[ "${use_proxy}" == "y" || "${use_proxy}" == "Y" ]]; then
        PACKAGE_URL="https://ghproxy.com/${PACKAGE_URL}"
    fi

    export PACKAGE_URL REMOTE_VER PACKAGE_NAME
    return 0
}

# ==========================================
# 3. 下载与解压
# ==========================================
function download_and_extract() {
    echo "开始下载: ${PACKAGE_URL}"
    cd "${TMPDIR}"
    
    if ! curl -fSL -# -o "${PACKAGE_NAME}" "${PACKAGE_URL}"; then
        echo "下载失败，请检查网络。"
        exit 1
    fi

    echo "下载完成，开始解压..."
    tar -zxf "${PACKAGE_NAME}"
    
    # 假设打包时是将所有文件打包在一个包含可执行文件的顶层文件夹里
    local extracted_dir
    extracted_dir="$(find . -maxdepth 2 -type f \( -name 'gopanel' -o -name 'gopanel.exe' \) -exec dirname {} \; | head -n1)"
    
    if [ -z "$extracted_dir" ]; then
        echo "错误：解压包内未找到 gopanel 可执行文件！"
        exit 1
    fi
    
    export package_root="$(cd "$extracted_dir" && pwd)"
}

# ==========================================
# 4. 安装 Docker (按需)
# ==========================================
function install_docker(){
    if command -v docker >/dev/null 2>&1; then
        echo "Docker 已安装，跳过"
    else
        echo "未检测到 Docker，开始自动安装..."
        if [ -f "${package_root}/script/install_docker.sh" ]; then
            /bin/bash "${package_root}/script/install_docker.sh" || { echo "Docker 安装失败"; exit 1; }
        else
            # 直接调用官方安装脚本
            curl -fsSL https://get.docker.com | bash -s docker || { echo "Docker 安装失败"; exit 1; }
        fi
    fi
}

# ==========================================
# 5. 交互式初始化配置
# ==========================================
function set_init_conf(){
    if [ -f "${CONFIG_INSTALL_DIR}/conf.yaml" ]; then
        echo "检测到已有配置文件 ${CONFIG_INSTALL_DIR}/conf.yaml，判断为更新操作，跳过初始化配置..."
        export IS_UPDATE=1
        return 0
    fi
    
    export IS_UPDATE=0

    echo "-----------------------------------"
    echo "基础配置初始化"
    echo "-----------------------------------"
    
    read -e -p "请输入安装目录 (默认: ${CONFIG_INSTALL_DIR}): " input_dir
    CONFIG_INSTALL_DIR=${input_dir:-${CONFIG_INSTALL_DIR}}

    read -e -p "请输入服务端口 (默认: ${CONFIG_PORT}): " input_port
    CONFIG_PORT=${input_port:-${CONFIG_PORT}}

    read -e -p "请设置登录用户名 (默认: ${CONFIG_USER}): " input_user
    CONFIG_USER=${input_user:-${CONFIG_USER}}

    read -e -p "请设置登录密码 (默认随机: ${CONFIG_PASSWORD}): " input_password
    CONFIG_PASSWORD=${input_password:-${CONFIG_PASSWORD}}

    read -e -p "请设置安全入口路径 (默认随机: ${CONFIG_SAFE_ENTER}): " input_safe_enter
    CONFIG_SAFE_ENTER=${input_safe_enter:-${CONFIG_SAFE_ENTER}}

    # 写入初始密码文件，供 Go 程序启动时初始化数据库用
    cat <<EOF > "${package_root}/init.yaml"
base_dir: "${CONFIG_INSTALL_DIR}"
port: ${CONFIG_PORT}
user: "${CONFIG_USER}"
password: "${CONFIG_PASSWORD}"
safe_enter: "${CONFIG_SAFE_ENTER}"
EOF

    # 写入持久化配置
    cat <<EOF > "${package_root}/conf.yaml"
system:
  entrance: "${CONFIG_SAFE_ENTER}"
http:
  listen: ":${CONFIG_PORT}"
db:
  database: "${CONFIG_INSTALL_DIR}/db/gopanel.db"
EOF
    chmod 644 "${package_root}/conf.yaml"
}

# ==========================================
# 6. 文件部署与服务拉起
# ==========================================
function deploy_and_start() {
    echo "正在部署文件到 ${CONFIG_INSTALL_DIR} ..."
    
    mkdir -p "${CONFIG_INSTALL_DIR}"

    # 更新操作：如果存在旧服务，先停止
    if [ "$OS" = "linux" ] && command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet gopanel; then
        systemctl stop gopanel || true
    else
        killall gopanel >/dev/null 2>&1 || true
        killall gopanel.exe >/dev/null 2>&1 || true
    fi

    # 复制所有文件到目标目录，覆盖已有的执行文件和前端资源（但不要覆盖已经存在且修改过的 conf.yaml/db）
    # 为了避免覆盖 db/ 里的数据，如果是更新，不要盲目覆盖 db 文件夹，除非它是空的。
    if [ "${IS_UPDATE}" = "1" ]; then
        echo "更新核心文件与静态资源..."
        # 移除安装包中可能自带的初始化数据，防止覆盖已有数据
        rm -f "${package_root}/init.yaml"
        rm -f "${package_root}/db/gopanel.db"
    fi
    
    cp -a "${package_root}/." "${CONFIG_INSTALL_DIR}/"
    [ -f "${CONFIG_INSTALL_DIR}/gopanel" ] && chmod +x "${CONFIG_INSTALL_DIR}/gopanel"
    [ -f "${CONFIG_INSTALL_DIR}/gopanel.exe" ] && chmod +x "${CONFIG_INSTALL_DIR}/gopanel.exe"

    # 根据系统配置守护进程
    if [ "$OS" = "linux" ]; then
        local service_file="${CONFIG_INSTALL_DIR}/script/gopanel.service"
        if [ -f "${service_file}" ]; then
            sed -i "s|WorkingDirectory=/opt/gopanel|WorkingDirectory=${CONFIG_INSTALL_DIR}|g" "${service_file}"
            sed -i "s|ExecStart=/opt/gopanel/gopanel|ExecStart=${CONFIG_INSTALL_DIR}/gopanel|g" "${service_file}"
            
            cp "${service_file}" /etc/systemd/system/
            systemctl daemon-reload
            systemctl enable gopanel
            systemctl restart gopanel
            echo "已配置 systemd 开机自启服务。"
        else
            echo "警告：未找到 systemd 服务模板文件，启动可能会失败。"
        fi
    elif [ "$OS" = "darwin" ]; then
        # macOS 建议直接后台运行或使用 launchd
        cd "${CONFIG_INSTALL_DIR}"
        nohup ./gopanel > /dev/null 2>&1 &
        echo "已在 macOS 后台启动 gopanel。"
    elif [ "$OS" = "windows" ]; then
        # Windows (Git Bash/Cygwin) 下后台运行，可执行文件通常带 .exe 后缀
        cd "${CONFIG_INSTALL_DIR}"
        if [ -f "./gopanel.exe" ]; then
            nohup ./gopanel.exe > /dev/null 2>&1 &
        else
            nohup ./gopanel > /dev/null 2>&1 &
        fi
        echo "已在 Windows 后台启动 gopanel。"
        echo "注意：Windows 下若需开机自启，建议将启动快捷方式放入【启动】文件夹或使用 NSSM 注册为系统服务。"
    fi
}

# ==========================================
# 7. 打印访问信息
# ==========================================
function show_success_info() {
    echo "================================================="
    if [ "${IS_UPDATE}" = "1" ]; then
        echo "🎉 GoPanel 更新成功！(版本: ${REMOTE_VER})"
        echo "================================================="
        echo "面板目录: ${CONFIG_INSTALL_DIR}"
        echo "服务已重启，请刷新浏览器访问。"
    else
        echo "🎉 GoPanel 安装成功！"
        echo "================================================="
        echo "面板目录: ${CONFIG_INSTALL_DIR}"
        echo "用户名:   ${CONFIG_USER}"
        echo "密码:     ${CONFIG_PASSWORD}"
        echo ""
        
        local ip_address
        ip_address="$(hostname -I 2>/dev/null | awk '{print $1}' || echo '127.0.0.1')"
        echo "👉 内网访问: http://${ip_address}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
        
        # 尝试获取公网 IP
        local public_ip
        public_ip=$(curl -s --connect-timeout 3 https://ifconfig.me || echo "")
        if [ -n "$public_ip" ]; then
            echo "👉 公网访问: http://${public_ip}:${CONFIG_PORT}/${CONFIG_SAFE_ENTER}"
        fi
    fi
    echo "================================================="
}

# ==========================================
# 主流程
# ==========================================
main() {
    echo "欢迎使用 GoPanel 安装向导 (版本: ${VERSION})"
    echo "============================================="
    
    fetch_download_info
    echo "最新版本: ${REMOTE_VER}"
    
    download_and_extract
    install_docker
    set_init_conf
    deploy_and_start
    show_success_info
}

main