#!/bin/bash

# ==============================================================================
# GoPanel 发布脚本
# ==============================================================================

set -e

# --- 配置项 ---
# 注意：这里可以只写 Bucket 名字或者 Endpoint，我们在下面会强制格式化
OSS_BUCKET_NAME="ft-shoply" # 建议只写 bucket 名称
APP_NAME="${2:-gopanel}"
OSS_PREFIX="${APP_NAME}/releases" 
LOCAL_DIST_DIR="./dist"
if [ "${APP_NAME}" = "consolex" ]; then
  LOCAL_DIST_DIR="./dist/consolex"
fi
# --- 参数校验 ---
if [ -z "$1" ]; then
  echo "❌ 错误: 请指定要发布的版本号!"
  echo "用法: ./publish.sh <version>"
  exit 1
fi

VERSION=$1
TARGET_DIR="${LOCAL_DIST_DIR}/${VERSION}"

echo "========================================"
echo "🚀 开始发布 GoPanel 版本: ${VERSION}"
echo "========================================"

# 1. 检查本地目录
if [ ! -d "$TARGET_DIR" ]; then
  echo "❌ 错误: 找不到目录: $TARGET_DIR"
  exit 1
fi

# 2. 检查 ossutil 工具
if command -v ossutil &> /dev/null; then
    OSS_CMD="ossutil"
else
    echo "❌ 错误: 未找到 ossutil，请先安装。"
    exit 1
fi

# 3. 加载凭证并格式化路径
if [ -f "./.oss_env" ]; then
  echo "🔑 加载本地 .oss_env 凭证..."
  source ./.oss_env
else
  echo "❌ 错误: 找不到 .oss_env 配置文件"
  exit 1
fi

# 【关键点】强制添加 oss:// 前缀，并清理多余斜杠
CLEAN_PREFIX=$(echo "${OSS_PREFIX}" | sed 's|^/||;s|/$||')
# 确保目标路径格式为 oss://bucket-name/path/
OSS_TARGET_PATH="oss://${OSS_BUCKET_NAME}/${CLEAN_PREFIX}/${VERSION}/"

echo "📦 待发布目录: $TARGET_DIR"
echo "☁️ 目标 OSS 路径: $OSS_TARGET_PATH"

# 4. 生成临时配置文件
# 使用 .oss_env 里的变量
TEMP_CONFIG=".tmp_oss_config"
cat > $TEMP_CONFIG <<EOF
[Credentials]
    endpoint = ${OSS_ENDPOINT}
    accessKeyID = ${OSS_ACCESS_KEY_ID}
    accessKeySecret = ${OSS_ACCESS_KEY_SECRET}
    region = ${OSS_REGION}
EOF

# 5. 执行上传
echo "----------------------------------------"

# 显式指定配置文件执行
# 这样 ossutil 就会从临时文件里读取你配置好的 endpoint 和 key
$OSS_CMD cp "${TARGET_DIR}/" "${OSS_TARGET_PATH}" -r -f --config-file "$TEMP_CONFIG"

# 6. 清理
rm -f "$TEMP_CONFIG"

echo "========================================"
echo "✅ 成功: 版本 ${VERSION} 已同步至 OSS!"
echo "========================================"