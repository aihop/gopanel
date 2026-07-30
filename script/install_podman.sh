#!/usr/bin/env bash
set -euo pipefail

GOPANEL_PODMAN_INSTALLER=1

log() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*" >&2; }
die() { echo "[ERROR] $*" >&2; exit 1; }

ensure_macos() {
  if [ "$(uname -s)" != "Darwin" ]; then
    die "当前独立安装器仅用于 macOS，Linux 请通过 GoPanel 主安装脚本安装 Podman。"
  fi
}

podman_major_version() {
  local version major
  version="$(podman version --format '{{.Client.Version}}' 2>/dev/null || true)"
  if [ -z "${version}" ]; then
    version="$(podman --version 2>/dev/null | sed -n 's/^[^0-9]*\([0-9][0-9]*\).*/\1/p' || true)"
  fi
  major="${version%%.*}"
  case "${major}" in
    ''|*[!0-9]*) echo "0" ;;
    *) echo "${major}" ;;
  esac
}

install_podman() {
  command -v brew >/dev/null 2>&1 || \
    die "未检测到 Homebrew。请先安装 Homebrew 后重新运行。"

  if command -v podman >/dev/null 2>&1; then
    local major
    major="$(podman_major_version)"
    if [ "${major}" -lt 5 ]; then
      log "当前 Podman 低于 5.x，开始升级..."
      brew upgrade podman || die "Podman 升级失败，请检查 Homebrew 输出后重试。"
    else
      log "Podman 已安装，跳过重复安装。"
    fi
  else
    log "通过 Homebrew 安装 Podman..."
    brew install podman || die "Podman 安装失败，请检查 Homebrew 输出后重试。"
  fi

  command -v podman >/dev/null 2>&1 || \
    die "Homebrew 执行完成，但仍未检测到 podman 命令。"
}

install_podman_compose() {
  if command -v podman-compose >/dev/null 2>&1 || podman compose version >/dev/null 2>&1; then
    log "Podman Compose 已可用。"
    return 0
  fi

  log "通过 Homebrew 安装 podman-compose..."
  brew install podman-compose || \
    die "podman-compose 安装失败，请检查 Homebrew 输出后重试。"

  if ! command -v podman-compose >/dev/null 2>&1 && ! podman compose version >/dev/null 2>&1; then
    die "Homebrew 执行完成，但 Podman Compose 仍不可用。"
  fi
}

ensure_podman_machine() {
  if podman info >/dev/null 2>&1; then
    log "Podman machine 已就绪。"
    return 0
  fi

  if ! podman machine inspect >/dev/null 2>&1; then
    log "初始化 Podman machine..."
    podman machine init || die "Podman machine 初始化失败。"
  fi

  log "启动 Podman machine..."
  if ! podman machine start; then
    podman info >/dev/null 2>&1 || die "Podman machine 启动失败。"
  fi
  podman info >/dev/null 2>&1 || die "Podman machine 已启动，但 podman info 检查失败。"
}

main() {
  ensure_macos
  install_podman
  install_podman_compose
  ensure_podman_machine
  log "Podman、Podman Compose 和 Podman machine 已安装并可用。"
}

main "$@"
