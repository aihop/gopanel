#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# GoPanel GitHub Releases 自动发布脚本
# ==========================================

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${PROJECT_ROOT}"

# ==========================================
# 1. 检查必要命令
# ==========================================
if ! command -v gh >/dev/null 2>&1; then
    echo "错误：未找到 gh 命令 (GitHub CLI)"
    echo "请先安装: "
    echo "  macOS: brew install gh"
    echo "  Linux: 按官方文档安装 https://github.com/cli/cli#installation"
    echo "安装后请执行 'gh auth login' 登录您的 GitHub 账号"
    exit 1
fi

# ==========================================
# 2. 检查登录状态
# ==========================================
if ! gh auth status >/dev/null 2>&1; then
    echo "错误：您还没有登录 GitHub CLI。"
    echo "请执行 'gh auth login' 进行授权登录。"
    exit 1
fi

# ==========================================
# 3. 确定版本号
# ==========================================
VERSION="${1:-}"

if [ -z "${VERSION}" ]; then
    echo "用法: $0 <版本号> [仓库名称]"
    echo "示例: $0 1.0.0"
    echo "示例: $0 1.0.0 aihop/gopanel"
    exit 1
fi

# 确保 TAG_NAME 带 v 前缀 (例如 v0.4.2)
if [[ "${VERSION}" == v* ]]; then
    TAG_NAME="${VERSION}"
    VERSION="${VERSION#v}"
else
    TAG_NAME="v${VERSION}"
fi

# 仓库名称
REPO="${2:-aihop/gopanel}"
OUTDIR="${PROJECT_ROOT}/dist/${TAG_NAME}"

echo "==========================================="
echo "即将发布版本: ${TAG_NAME} (${VERSION})"
echo "目标仓库: ${REPO}"
echo "打包目录: ${OUTDIR}"
echo "==========================================="

# ==========================================
# 4. 检查打包文件是否存在
# ==========================================
if [ ! -d "${OUTDIR}" ]; then
    echo "错误：未找到打包目录 ${OUTDIR}"
    echo "请先运行: bash build.sh ${VERSION}"
    exit 1
fi

# 收集所有需要上传的文件
ASSETS=()
while IFS=  read -r -d $'\0'; do
    ASSETS+=("$REPLY")
done < <(find "${OUTDIR}" -maxdepth 1 -name "*.tar.gz" -print0)

# 如果存在说明文档，也一并作为 Release 附件上传
if [ -f "${PROJECT_ROOT}/README.md" ]; then
    ASSETS+=("${PROJECT_ROOT}/README.md")
fi
if [ -f "${PROJECT_ROOT}/README_zh.md" ]; then
    ASSETS+=("${PROJECT_ROOT}/README_zh.md")
fi
if [ -f "${PROJECT_ROOT}/perview.png" ]; then
    ASSETS+=("${PROJECT_ROOT}/perview.png")
fi

if [ ${#ASSETS[@]} -eq 0 ]; then
    echo "错误：在 ${OUTDIR} 下没有找到任何 .tar.gz 文件。"
    echo "请先运行: bash build.sh ${VERSION}"
    exit 1
fi

echo "找到以下需要上传的发布包:"
for asset in "${ASSETS[@]}"; do
    echo "  - $(basename "$asset")"
done
echo ""

# ==========================================
# 5. 二次确认
# ==========================================
read -p "确认要创建 Release 并上传这些文件到 GitHub 吗? (y/n): " confirm
if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
    echo "已取消发布。"
    exit 0
fi

# ==========================================
# 6. 创建或更新 GitHub Release
# ==========================================
echo "正在检查是否已存在同名 Release ${TAG_NAME} ..."

# 如果已存在该 Release，则直接上传/覆盖资源；如果不存在，则创建新的 Release
if gh release view "${TAG_NAME}" --repo "${REPO}" >/dev/null 2>&1; then
    echo "Release ${TAG_NAME} 已存在，准备上传资源..."
else
    echo "创建新的 Release: ${TAG_NAME} ..."
    gh release create "${TAG_NAME}" \
        --repo "${REPO}" \
        --title "GoPanel Release ${TAG_NAME}" \
        --notes "GoPanel ${TAG_NAME} 自动发布" \
        --draft=false \
        --prerelease=false
fi

echo "正在上传文件到 GitHub..."
for asset in "${ASSETS[@]}"; do
    echo "上传: $(basename "$asset")"
    # --clobber 表示如果同名文件存在则覆盖
    gh release upload "${TAG_NAME}" "$asset" --repo "${REPO}" --clobber
done

echo "==========================================="
echo "🎉 发布成功！"
echo "您可以访问以下链接查看您的 Release："
echo "https://github.com/${REPO}/releases/tag/${TAG_NAME}"
echo "==========================================="
