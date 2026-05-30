#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# gp-agent GitHub & GitCode Releases 自动发布脚本
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

if ! command -v jq >/dev/null 2>&1; then
    echo "警告：未找到 jq 命令，发布到 GitCode 时需要使用 jq 解析 JSON 响应。"
    echo "如果不需要发布到 GitCode，请忽略此警告。"
    echo "请先安装: "
    echo "  macOS: brew install jq"
    echo "  Linux: apt-get install jq"
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

# 为了避免和主项目的 tag 冲突，如果用户希望 gp-agent 使用不同的 tag 前缀，
# 可以在此处修改。此处默认保持和主项目一致的 tag 格式，但在 release title 中标明 gp-agent。

echo "==========================================="
echo "即将发布 gp-agent 版本: ${TAG_NAME} (${VERSION})"
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

# 收集所有需要上传的文件 (.tar.gz 和 .manifest.json)
ASSETS=()
while IFS=  read -r -d $'\0'; do
    ASSETS+=("$REPLY")
done < <(find "${OUTDIR}" -maxdepth 1 \( -name "*.tar.gz" -o -name "*.manifest.json" \) -print0)

if [ ${#ASSETS[@]} -eq 0 ]; then
    echo "错误：在 ${OUTDIR} 下没有找到任何 .tar.gz 或 .manifest.json 文件。"
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
read -p "确认要创建 Release 并上传这些文件到 GitHub (如配置了 GITCODE_TOKEN 还会发布到 GitCode) 吗? (y/n): " confirm
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
        --title "gp-agent Release ${TAG_NAME}" \
        --notes "gp-agent ${TAG_NAME} 自动发布" \
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
echo "🎉 GitHub 发布成功！"
echo "您可以访问以下链接查看您的 Release："
echo "https://github.com/${REPO}/releases/tag/${TAG_NAME}"
echo "==========================================="

# ==========================================
# 7. 创建或更新 GitCode Release
# ==========================================
if [ -n "${GITCODE_TOKEN:-}" ]; then
    if ! command -v jq >/dev/null 2>&1; then
        echo "错误：未找到 jq 命令，无法发布到 GitCode。请先安装 jq。"
    else
        echo "==========================================="
        echo "正在检查是否已存在同名 GitCode Release ${TAG_NAME} ..."
        
        # GitCode API 获取 Release 信息
        GC_RELEASE_INFO=$(curl -s -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" "https://api.gitcode.com/api/v5/repos/${REPO}/releases/tags/${TAG_NAME}")
        GC_RELEASE_ID=$(echo "$GC_RELEASE_INFO" | jq -r '.id // empty')
        
        if [ -n "$GC_RELEASE_ID" ] && [ "$GC_RELEASE_ID" != "null" ]; then
            echo "GitCode Release ${TAG_NAME} 已存在，准备上传资源..."
        else
            echo "创建新的 GitCode Release: ${TAG_NAME} ..."
            CREATE_RES=$(curl -s -X POST "https://api.gitcode.com/api/v5/repos/${REPO}/releases" \
                -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" \
                -H "Content-Type: application/json" \
                -d '{
                    "tag_name": "'"${TAG_NAME}"'",
                    "name": "gp-agent Release '"${TAG_NAME}"'",
                    "body": "gp-agent '"${TAG_NAME}"' 自动发布",
                    "release_status": "latest"
                }')
            
            # 检查是否创建成功或因 Tag 不存在而失败
            # GitCode 在 Tag 不存在时创建 Release 会失败，这里我们加上 target_commitish 参数，让 GitCode 自动创建 Tag
            if echo "$CREATE_RES" | grep -q '"error_code":'; then
                echo "提示：创建 Release 失败，可能是 Tag 不存在。尝试携带 target_commitish 参数重新创建..."
                
                curl -s -X POST "https://api.gitcode.com/api/v5/repos/${REPO}/releases" \
                    -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" \
                    -H "Content-Type: application/json" \
                    -d '{
                        "tag_name": "'"${TAG_NAME}"'",
                        "target_commitish": "main",
                        "name": "gp-agent Release '"${TAG_NAME}"'",
                        "body": "gp-agent '"${TAG_NAME}"' 自动发布",
                        "release_status": "latest"
                    }' > /dev/null
            fi
        fi
        
        echo "正在上传文件到 GitCode..."
        for asset in "${ASSETS[@]}"; do
            filename=$(basename "$asset")
            echo "正在获取 GitCode 上传地址: $filename"
            
            upload_info=$(curl -s -G -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" --data-urlencode "file_name=${filename}" "https://api.gitcode.com/api/v5/repos/${REPO}/releases/${TAG_NAME}/upload_url")
            upload_url=$(echo "$upload_info" | jq -r '.url // empty')
            
            if [ -z "$upload_url" ] || [ "$upload_url" == "null" ]; then
                echo "获取 GitCode 上传地址失败: $upload_info"
                continue
            fi
            
            # 提取 headers 构造成 curl 的 -H 参数
            curl_opts=()
            while read -r header_key header_val; do
                curl_opts+=("-H" "$header_key: $header_val")
            done < <(echo "$upload_info" | jq -r '.headers | to_entries | .[] | "\(.key) \(.value)"')
            
            echo "上传到 GitCode: $filename"
            curl -s -X PUT "${curl_opts[@]}" -T "$asset" "$upload_url" > /dev/null
        done
        
        echo "==========================================="
        echo "🎉 GitCode 发布成功！"
        echo "您可以访问以下链接查看您的 Release："
        echo "https://gitcode.com/${REPO}/-/releases/${TAG_NAME}"
        echo "==========================================="
    fi
else
    echo "提示：未配置 GITCODE_TOKEN 环境变量，已跳过 GitCode 发布。"
fi
