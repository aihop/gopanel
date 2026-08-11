#!/usr/bin/env bash
set -Eeuo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${GOPANEL_BASE_DIR:-${HOME}/.gopanel}"
TARGET_BINARY="${INSTALL_DIR}/gopanel"
LOCAL_BINARY="${PROJECT_ROOT}/dist/local/gopanel"
SERVICE_LABEL="${GOPANEL_SERVICE_LABEL:-io.aihop.gopanel}"
HEALTH_URL="${GOPANEL_HEALTH_URL:-}"
HEALTH_TIMEOUT="${GOPANEL_HEALTH_TIMEOUT:-45}"
NPM_CACHE_DIR="${GOPANEL_NPM_CACHE_DIR:-${INSTALL_DIR}/cache/npm-local-build}"
GO_CACHE_DIR="${GOPANEL_GO_CACHE_DIR:-${INSTALL_DIR}/cache/go-local-build}"
GPC_BINARY="${GOPANEL_GPC_BINARY:-}"
BACKUP_BINARY=""
REPLACED=0

log() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

fail() {
  printf '\n错误: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1"
}

detect_health_url() {
  if [ -n "${HEALTH_URL}" ]; then
    return
  fi
  local port="15470"
  if [ -f "${INSTALL_DIR}/conf.yaml" ]; then
    local configured_port
    configured_port="$(awk '$1 == "port:" {gsub(/[\"\047]/, "", $2); sub(/^:/, "", $2); print $2; exit}' "${INSTALL_DIR}/conf.yaml")"
    if [[ "${configured_port}" =~ ^[0-9]+$ ]]; then
      port="${configured_port}"
    fi
  fi
  HEALTH_URL="http://127.0.0.1:${port}/health"
}

detect_version() {
  if [ -n "${GOPANEL_VERSION:-}" ]; then
    printf '%s\n' "${GOPANEL_VERSION#v}"
    return
  fi
  local running_version
  running_version="$(curl -fsS --max-time 2 "${HEALTH_URL}" 2>/dev/null | sed -n 's/.*"appVersion":"\([^"]*\)".*/\1/p' || true)"
  if [ -n "${running_version}" ]; then
    printf '%s\n' "${running_version#v}"
    return
  fi
  local git_version
  git_version="$(git -C "${PROJECT_ROOT}" describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || true)"
  if [ -n "${git_version}" ]; then
    printf '%s\n' "${git_version#v}"
    return
  fi
  printf '1.0.0\n'
}

derive_version_code() {
  local version="${1#v}"
  version="${version%%-*}"
  local major minor patch
  IFS='.' read -r major minor patch <<< "${version}"
  major="${major:-0}"
  minor="${minor:-0}"
  patch="${patch:-0}"
  [[ "${major}" =~ ^[0-9]+$ && "${minor}" =~ ^[0-9]+$ && "${patch}" =~ ^[0-9]+$ ]] || fail "版本号格式无效: $1"
  printf '%s\n' "$((10#${major} * 100000 + 10#${minor} * 1000 + 10#${patch}))"
}

find_gpc() {
  if [ -n "${GPC_BINARY}" ] && [ -x "${GPC_BINARY}" ]; then
    return
  fi
  local candidate
  for candidate in /usr/local/bin/gpc /opt/homebrew/bin/gpc "${INSTALL_DIR}/gpc"; do
    if [ -x "${candidate}" ]; then
      GPC_BINARY="${candidate}"
      return
    fi
  done
  GPC_BINARY=""
}

restart_service() {
  find_gpc
  if [ -n "${GPC_BINARY}" ] && "${GPC_BINARY}" --base-dir "${INSTALL_DIR}" panel restart; then
    return
  fi
  sudo launchctl kickstart -k "system/${SERVICE_LABEL}"
}

wait_for_health() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT))
  while [ "${SECONDS}" -lt "${deadline}" ]; do
    if curl -fsS --max-time 2 "${HEALTH_URL}" 2>/dev/null | grep -q '"code":0'; then
      return 0
    fi
    sleep 1
  done
  return 1
}

rollback() {
  if [ "${REPLACED}" -ne 1 ] || [ ! -f "${BACKUP_BINARY}" ]; then
    return
  fi
  log "启动失败，正在恢复旧程序"
  cp -p "${BACKUP_BINARY}" "${TARGET_BINARY}.rollback"
  chmod 755 "${TARGET_BINARY}.rollback"
  mv -f "${TARGET_BINARY}.rollback" "${TARGET_BINARY}"
  restart_service || true
  wait_for_health || true
}

main() {
  [ "$(uname -s)" = "Darwin" ] || fail "此脚本只能在 macOS 上运行"
  [ -x "${TARGET_BINARY}" ] || fail "未找到本机 GoPanel: ${TARGET_BINARY}"
  [[ "${HEALTH_TIMEOUT}" =~ ^[0-9]+$ ]] || fail "GOPANEL_HEALTH_TIMEOUT 必须是整数"
  require_command go
  require_command node
  require_command npm
  require_command curl
  require_command file

  detect_health_url
  local version version_code build_time git_commit ldflags
  version="$(detect_version)"
  version_code="$(derive_version_code "${version}")"
  build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  git_commit="$(git -C "${PROJECT_ROOT}" rev-parse --short HEAD 2>/dev/null || true)"
  ldflags="-s -w -X github.com/aihop/gopanel/constant.AppVersion=${version} -X github.com/aihop/gopanel/constant.BuildTime=${build_time} -X github.com/aihop/gopanel/constant.BuildVersionCode=${version_code} -X github.com/aihop/gopanel/constant.AppBrand=GoPanel"

  log "构建前端资源"
  mkdir -p "${NPM_CACHE_DIR}" "${GO_CACHE_DIR}"
  if [ ! -f "${PROJECT_ROOT}/admin/node_modules/vite/bin/vite.js" ]; then
    (
      cd "${PROJECT_ROOT}/admin"
      npm_config_cache="${NPM_CACHE_DIR}" npm_config_fetch_retries=2 \
        npm_config_fetch_timeout=30000 npm ci --legacy-peer-deps \
        --prefer-offline --no-audit --no-fund
    )
  fi
  (
    cd "${PROJECT_ROOT}/admin"
    node node_modules/vite/bin/vite.js build --mode prod --outDir ../public --emptyOutDir
  )

  log "构建当前 Mac 使用的单一 gopanel 程序"
  mkdir -p "$(dirname "${LOCAL_BINARY}")"
  (
    cd "${PROJECT_ROOT}"
    GOCACHE="${GO_CACHE_DIR}" CGO_ENABLED=0 \
      go build -trimpath -ldflags "${ldflags}" -o "${LOCAL_BINARY}" ./main.go
  )
  chmod 755 "${LOCAL_BINARY}"
  file "${LOCAL_BINARY}"

  local backup_dir install_temp
  backup_dir="${INSTALL_DIR}/backups/local-build"
  mkdir -p "${backup_dir}"
  BACKUP_BINARY="${backup_dir}/gopanel.$(date '+%Y%m%d-%H%M%S')"
  cp -p "${TARGET_BINARY}" "${BACKUP_BINARY}"

  log "替换并重启本机 GoPanel"
  install_temp="${TARGET_BINARY}.new.$$"
  cp "${LOCAL_BINARY}" "${install_temp}"
  chmod 755 "${install_temp}"
  mv -f "${install_temp}" "${TARGET_BINARY}"
  REPLACED=1
  if ! restart_service || ! wait_for_health; then
    rollback
    fail "本机服务未能正常启动，已尝试恢复旧程序"
  fi

  REPLACED=0
  log "完成"
  printf '开发产物: %s\n' "${LOCAL_BINARY}"
  printf '本机程序: %s\n' "${TARGET_BINARY}"
  printf '旧版备份: %s\n' "${BACKUP_BINARY}"
  printf '提交版本: %s\n' "${git_commit}"
}

main "$@"
