#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${PROJECT_ROOT}"

VERSION="${1:-1.0.0}"
VERSION_CODE="${2:-100000}"
APP_BRAND="${3:-GoPanel}"
APP_NAME="gopanel"

# 必须根据实际传入的参数个数来进行 shift
# 如果传入了参数，最多 shift 3，否则如果没传满 3 个，就会导致把默认没传的空位留给后续的 TARGETS
if [ $# -ge 3 ]; then
  shift 3
elif [ $# -ge 2 ]; then
  shift 2
elif [ $# -ge 1 ]; then
  shift 1
fi

# 默认关闭 cgo
CGO="${CGO:-0}"

# targets: format GOOS/GOARCH
if [ $# -gt 0 ]; then
  TARGETS=("$@")
else
  TARGETS=("darwin/arm64" "darwin/amd64" "linux/amd64" "linux/arm64" "windows/amd64")
fi

if [ "${APP_BRAND}" = "consolex" ]; then
  APP_BRAND="ConsoleX"
fi

if [ "${APP_BRAND}" = "ConsoleX" ]; then
  OUTDIR="${PROJECT_ROOT}/dist/consolex/v${VERSION}"
else 
  OUTDIR="${PROJECT_ROOT}/dist/v${VERSION}"
fi
MAIN_PKG="./main.go"
LDFLAGS="-s -w -X github.com/aihop/gopanel/constant.AppVersion=${VERSION} -X github.com/aihop/gopanel/constant.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X github.com/aihop/gopanel/constant.BuildVersionCode=${VERSION_CODE} -X github.com/aihop/gopanel/constant.AppBrand=${APP_BRAND}"

echo "==========================================="
echo "Building Project: ${APP_BRAND}"
echo "Targets: ${TARGETS[*]}"
echo "==========================================="

# 前端构建逻辑 (保持不变)
if [ -d "${PROJECT_ROOT}/admin" ]; then
  echo "Building frontend..."
  if [ "${APP_BRAND}" = "ConsoleX" ]; then
    (cd "${PROJECT_ROOT}/admin" && npm install && npm run build:consolex)
  else
    (cd "${PROJECT_ROOT}/admin" && npm install && npm run build)
  fi
  mkdir -p "${PROJECT_ROOT}/public"
  cp -r "${PROJECT_ROOT}/admin/dist/"* "${PROJECT_ROOT}/public/"
fi

rm -rf "${OUTDIR}"
mkdir -p "${OUTDIR}"

# --- 核心构建函数优化 ---

build_local() {
  local goos="$1" goarch="$2" outdir="$3" exe_name="$4" cgo_enabled="$5"
  echo ">>> [Build] ${goos}/${goarch} (CGO_ENABLED=${cgo_enabled})"
  
  mkdir -p "${outdir}"
  local output_path="${outdir}/${exe_name}"
  if [ "${goos}" = "windows" ]; then output_path="${output_path}.exe"; fi

  # 关键：显式清除可能存在的旧缓存干扰，并强制设置环境变量
  GOOS=${goos} GOARCH=${goarch} CGO_ENABLED=${cgo_enabled} \
  go build -trimpath -ldflags "${LDFLAGS}" -o "${output_path}" "${MAIN_PKG}"
  
  if [ "${goos}" != "windows" ]; then chmod +x "${output_path}"; fi
  
  # 验证生成的文件架构 (防止在 Debian 看到 darwin 的核心预防步骤)
  if command -v file >/dev/null; then
    echo "    Verify: $(file "${output_path}" | cut -d: -f2-)"
  fi
}

# Docker 构建逻辑保持不变，但增加平台参数确保架构正确
build_docker_linux() {
  local goarch="$1" outdir="$2" exe_name="$3"
  echo ">>> [Docker Build] linux/${goarch} (CGO_ENABLED=1)"
  mkdir -p "${outdir}"

  docker run --rm \
    --platform "linux/${goarch}" \
    -v "${PROJECT_ROOT}":/src \
    -v "${outdir}":/out \
    -w /src \
    -e CGO_ENABLED=1 \
    -e GOOS=linux \
    -e GOARCH="${goarch}" \
    golang:1.24 bash -c "
      apt-get update -qq && apt-get install -y -qq gcc libsqlite3-dev >/dev/null
      go build -trimpath -ldflags \"${LDFLAGS}\" -o /out/${exe_name} ${MAIN_PKG}
    "
}

# --- 主循环 ---

for t in "${TARGETS[@]}"; do
  IFS='/' read -r GOOS GOARCH <<< "${t}"
  short_name="${APP_NAME}-${GOOS}-${GOARCH}"
  dist_dir="${OUTDIR}/${short_name}"

  case "${GOOS}" in
    darwin)
      if [[ "$(uname -s)" != "Darwin" ]]; then
        echo "Skip darwin: Not on macOS host."
        continue
      fi
      build_local "${GOOS}" "${GOARCH}" "${dist_dir}" "${APP_NAME}" "${CGO}"
      ;;
    linux)
      if [ "${CGO}" = "0" ]; then
        build_local "linux" "${GOARCH}" "${dist_dir}" "${APP_NAME}" "0"
      else
        # 强制 Docker 构建以确保 CGO 环境正确
        build_docker_linux "${GOARCH}" "${dist_dir}" "${APP_NAME}"
      fi
      ;;
    windows)
      build_local "windows" "${GOARCH}" "${dist_dir}" "${APP_NAME}" "0"
      ;;
  esac

  # 打包
  tar -C "${OUTDIR}" -czf "${OUTDIR}/${short_name}.tar.gz" "${short_name}"
  echo "Finished: ${short_name}.tar.gz"
  
  # 清理临时目录
  [ "${KEEP_DIST_DIR:-0}" = "0" ] && rm -rf "${dist_dir}"
done

echo "=== All Done ==="
ls -lh "${OUTDIR}"
