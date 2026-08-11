#!/usr/bin/env bash
set -Eeuo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${GOPANEL_BASE_DIR:-${HOME}/.gopanel}"
TARGET_BINARY="${INSTALL_DIR}/gopanel"
SERVICE_LABEL="${GOPANEL_SERVICE_LABEL:-io.aihop.gopanel}"
HEALTH_URL="${GOPANEL_HEALTH_URL:-}"
HEALTH_TIMEOUT="${GOPANEL_HEALTH_TIMEOUT:-45}"
GPC_BINARY="${GOPANEL_GPC_BINARY:-}"
NPM_CACHE_DIR="${GOPANEL_NPM_CACHE_DIR:-${INSTALL_DIR}/cache/npm-local-build}"
GO_CACHE_DIR="${GOPANEL_GO_CACHE_DIR:-${INSTALL_DIR}/cache/go-local-build}"
TEMP_DIR=""
BACKUP_BINARY=""
REPLACED=0

log() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$*"
}

fail() {
  printf '\n错误: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [ -n "${TEMP_DIR}" ] && [ -d "${TEMP_DIR}" ]; then
    rm -rf "${TEMP_DIR}"
  fi
}

trap cleanup EXIT

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1"
}

detect_arch() {
  case "$(uname -m)" in
    arm64|aarch64) printf 'arm64\n' ;;
    x86_64|amd64) printf 'amd64\n' ;;
    *) fail "不支持的 Mac 架构: $(uname -m)" ;;
  esac
}

detect_health_url() {
  if [ -n "${HEALTH_URL}" ]; then
    return
  fi
  local config_file="${INSTALL_DIR}/conf.yaml"
  local port="15470"
  if [ -f "${config_file}" ]; then
    local configured_port
    configured_port="$(awk '$1 == "port:" {gsub(/[\"\047]/, "", $2); sub(/^:/, "", $2); print $2; exit}' "${config_file}")"
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
  local health_json=""
  health_json="$(curl -fsS --max-time 2 "${HEALTH_URL}" 2>/dev/null || true)"
  local running_version
  running_version="$(printf '%s' "${health_json}" | sed -n 's/.*"appVersion":"\([^"]*\)".*/\1/p')"
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
  if [ -n "${GPC_BINARY}" ]; then
    if "${GPC_BINARY}" --base-dir "${INSTALL_DIR}" panel restart; then
      return
    fi
    log "gpc 重启失败，改用 launchctl"
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
  if [ "${REPLACED}" -ne 1 ] || [ -z "${BACKUP_BINARY}" ] || [ ! -f "${BACKUP_BINARY}" ]; then
    return
  fi
  log "新版本未能正常启动，正在回滚"
  local rollback_temp="${TARGET_BINARY}.rollback.$$"
  cp -p "${BACKUP_BINARY}" "${rollback_temp}"
  chmod 755 "${rollback_temp}"
  mv -f "${rollback_temp}" "${TARGET_BINARY}"
  restart_service || true
  wait_for_health || true
}

main() {
  [ "$(uname -s)" = "Darwin" ] || fail "此脚本只能在 macOS 上运行"
  [ -x "${TARGET_BINARY}" ] || fail "未找到本机 GoPanel: ${TARGET_BINARY}"
  [[ "${HEALTH_TIMEOUT}" =~ ^[0-9]+$ ]] || fail "GOPANEL_HEALTH_TIMEOUT 必须是整数"
  require_command go
  require_command npm
  require_command curl
  require_command tar
  require_command file

  detect_health_url
  local arch version version_code package_path extracted_binary
  arch="$(detect_arch)"
  version="$(detect_version)"
  version_code="$(derive_version_code "${version}")"
  package_path="${PROJECT_ROOT}/dist/v${version}/gopanel-darwin-${arch}.tar.gz"
  TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/gopanel-local-install.XXXXXX")"

  log "构建 GoPanel ${version} (darwin/${arch})"
  mkdir -p "${NPM_CACHE_DIR}" "${GO_CACHE_DIR}"
  (
    cd "${PROJECT_ROOT}/admin"
    npm_config_cache="${NPM_CACHE_DIR}" npm_config_fetch_retries=2 \
      npm_config_fetch_timeout=30000 npm ci --legacy-peer-deps \
      --prefer-offline --no-audit --no-fund
  )
  (
    cd "${PROJECT_ROOT}"
    npm_config_cache="${NPM_CACHE_DIR}" GOCACHE="${GO_CACHE_DIR}" \
      LOCAL_FRONTEND_BUILD=1 KEEP_DIST_DIR=0 \
      bash ./build.sh "${version}" "${version_code}" GoPanel "darwin/${arch}"
  )
  [ -f "${package_path}" ] || fail "构建完成但未找到安装包: ${package_path}"

  tar -xzf "${package_path}" -C "${TEMP_DIR}"
  extracted_binary="${TEMP_DIR}/gopanel-darwin-${arch}/gopanel"
  [ -x "${extracted_binary}" ] || fail "安装包中缺少 gopanel 可执行文件"
  file "${extracted_binary}"
  if ! file "${extracted_binary}" | grep -q "${arch/amd64/x86_64}"; then
    fail "构建产物架构与本机不匹配"
  fi

  local backup_dir install_temp
  backup_dir="${INSTALL_DIR}/backups/local-build"
  mkdir -p "${backup_dir}"
  BACKUP_BINARY="${backup_dir}/gopanel.$(date '+%Y%m%d-%H%M%S')"
  cp -p "${TARGET_BINARY}" "${BACKUP_BINARY}"

  log "替换 ${TARGET_BINARY}"
  install_temp="${TARGET_BINARY}.new.$$"
  cp "${extracted_binary}" "${install_temp}"
  chmod 755 "${install_temp}"
  mv -f "${install_temp}" "${TARGET_BINARY}"
  REPLACED=1

  log "重启 ${SERVICE_LABEL}"
  if ! restart_service; then
    rollback
    fail "服务重启失败，已尝试恢复旧版本"
  fi
  if ! wait_for_health; then
    rollback
    fail "健康检查超时，已恢复旧版本；请检查 /tmp/gopanel.err.log"
  fi

  REPLACED=0
  log "本地安装完成"
  printf '安装包: %s\n' "${package_path}"
  printf '当前程序: %s\n' "${TARGET_BINARY}"
  printf '旧版备份: %s\n' "${BACKUP_BINARY}"
  printf '健康检查: %s\n' "${HEALTH_URL}"
}

main "$@"
