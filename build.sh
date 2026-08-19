#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${PROJECT_ROOT}"

# 从版本名推导 version_code：1.2.3 -> 102003（major*100000+minor*1000+patch）
derive_version_code() {
  local v="${1#v}"; v="${v%%-*}"; local a b c
  IFS='.' read -r a b c <<< "${v}"
  a="${a:-0}"; b="${b:-0}"; c="${c:-0}"
  echo $(( 10#${a} * 100000 + 10#${b} * 1000 + 10#${c} ))
}

write_sha256() {
  local archive="$1" digest
  if command -v sha256sum >/dev/null 2>&1; then
    digest="$(sha256sum "${archive}" | awk '{print $1}')"
  else
    digest="$(shasum -a 256 "${archive}" | awk '{print $1}')"
  fi
  printf '%s  %s\n' "${digest}" "$(basename "${archive}")" > "${archive}.sha256"
}

VERSION="${1:-1.0.0}"
# version_code 未显式传入(第二参非纯数字)则从版本名推导，避免误用默认 100000
if [[ "${2:-}" =~ ^[0-9]+$ ]]; then
  VERSION_CODE="${2}"
else
  VERSION_CODE="$(derive_version_code "${VERSION}")"
fi
APP_BRAND="GoPanel"
APP_NAME="gopanel"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || true)"
GIT_DIRTY="false"
if [ -n "${GIT_COMMIT}" ]; then
  if [ -n "$(git status --porcelain 2>/dev/null || true)" ]; then
    GIT_DIRTY="true"
  fi
fi

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

OUTDIR="${PROJECT_ROOT}/dist/v${VERSION}"
MAIN_PKG="./main.go"
GPC_ROOT="${PROJECT_ROOT}/gpc"
GPC_MAIN_PKG="./main.go"
LDFLAGS="-s -w -X github.com/aihop/gopanel/constant.AppVersion=${VERSION} -X github.com/aihop/gopanel/constant.BuildTime=${BUILD_TIME} -X github.com/aihop/gopanel/constant.BuildVersionCode=${VERSION_CODE} -X github.com/aihop/gopanel/constant.AppBrand=${APP_BRAND}"
GPC_LDFLAGS="-s -w"

echo "==========================================="
echo "Building Project: ${APP_BRAND}"
echo "Targets: ${TARGETS[*]}"
echo "==========================================="

bash "${PROJECT_ROOT}/scripts/check-file-size.sh"

# 前端构建逻辑
if [ -d "${PROJECT_ROOT}/admin" ]; then
  echo "Building frontend..."
  (cd "${PROJECT_ROOT}/admin" && npm install && npm run build)
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

build_gpc_local() {
  local goos="$1" goarch="$2" outdir="$3"
  echo ">>> [Build] gpc ${goos}/${goarch} (CGO_ENABLED=0)"

  mkdir -p "${outdir}"
  local output_path="${outdir}/gpc"
  if [ "${goos}" = "windows" ]; then output_path="${output_path}.exe"; fi

  (
    cd "${GPC_ROOT}"
    GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build -trimpath -ldflags "${GPC_LDFLAGS}" -o "${output_path}" "${GPC_MAIN_PKG}"
  )

  if [ "${goos}" != "windows" ]; then chmod +x "${output_path}"; fi

  if command -v file >/dev/null; then
    echo "    Verify gpc: $(file "${output_path}" | cut -d: -f2-)"
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
      build_gpc_local "${GOOS}" "${GOARCH}" "${dist_dir}"
      cp "${PROJECT_ROOT}/script/install_podman.sh" "${dist_dir}/install_podman.sh"
      chmod 755 "${dist_dir}/install_podman.sh"
      ;;
    linux)
      if [ "${CGO}" = "0" ]; then
        build_local "linux" "${GOARCH}" "${dist_dir}" "${APP_NAME}" "0"
      else
        # 强制 Docker 构建以确保 CGO 环境正确
        build_docker_linux "${GOARCH}" "${dist_dir}" "${APP_NAME}"
      fi
      build_gpc_local "linux" "${GOARCH}" "${dist_dir}"
      ;;
    windows)
      build_local "windows" "${GOARCH}" "${dist_dir}" "${APP_NAME}" "0"
      cp "${PROJECT_ROOT}/install.ps1" "${dist_dir}/install.ps1"
      ;;
  esac

  # 打包
  if [ "${GOOS}" = "windows" ] && command -v zip >/dev/null 2>&1; then
    (cd "${OUTDIR}" && zip -qr "${short_name}.zip" "${short_name}")
    write_sha256 "${OUTDIR}/${short_name}.zip"
    echo "Finished: ${short_name}.zip"
  else
    tar -C "${OUTDIR}" -czf "${OUTDIR}/${short_name}.tar.gz" "${short_name}"
    write_sha256 "${OUTDIR}/${short_name}.tar.gz"
    echo "Finished: ${short_name}.tar.gz"
  fi
  
  # 清理临时目录
  [ "${KEEP_DIST_DIR:-0}" = "0" ] && rm -rf "${dist_dir}"
done

echo "=== All Done ==="
ls -lh "${OUTDIR}"
