#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${PROJECT_ROOT}"

# 加载仓库根的 .env（存放 GOPANEL_ADMIN_KEY 等，已在 .gitignore 忽略、不入库）
ENV_FILE="${PROJECT_ROOT}/../.env"
if [ -f "${ENV_FILE}" ]; then
  set -a
  # shellcheck disable=SC1090
  . "${ENV_FILE}"
  set +a
fi

APP_NAME="gp-agent"

# 从版本名推导 version_code：1.2.1 -> 102001，1.0.2 -> 100002（major*100000+minor*1000+patch）
derive_version_code() {
  local v="${1#v}"     # 去掉前导 v
  v="${v%%-*}"         # 去掉预发布后缀
  local major minor patch
  IFS='.' read -r major minor patch <<< "${v}"
  major="${major:-0}"; minor="${minor:-0}"; patch="${patch:-0}"
  echo $(( 10#${major} * 100000 + 10#${minor} * 1000 + 10#${patch} ))
}

VERSION="${1:-1.0.0}"
[ $# -ge 1 ] && shift
# 第二个参数是纯数字则当 version_code，否则从版本名自动推导（你只传版本名即可）
if [[ "${1:-}" =~ ^[0-9]+$ ]]; then
  VERSION_CODE="$1"
  shift
else
  VERSION_CODE="$(derive_version_code "${VERSION}")"
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
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || true)"
GIT_DIRTY="false"
if [ -n "${GIT_COMMIT}" ]; then
  if [ -n "$(git status --porcelain 2>/dev/null || true)" ]; then
    GIT_DIRTY="true"
  fi
fi
LDFLAGS="-s -w -X github.com/aihop/gopanel/gp-agent/app/service.Version=${VERSION} -X github.com/aihop/gopanel/gp-agent/app/service.VersionCode=${VERSION_CODE} -X github.com/aihop/gopanel/gp-agent/app/service.BuildTime=${BUILD_TIME} -X github.com/aihop/gopanel/gp-agent/app/service.GitCommit=${GIT_COMMIT} -X github.com/aihop/gopanel/gp-agent/app/service.GitDirty=${GIT_DIRTY}"

echo "==========================================="
echo "Building Project: GoPanel Agent"
echo "Targets: ${TARGETS[*]}"
echo "==========================================="

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

sha256_file() {
  local p="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${p}" | awk '{print $1}'
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${p}" | awk '{print $1}'
    return 0
  fi
  echo ""
}

write_manifest() {
  local dist_dir="$1" goos="$2" goarch="$3" exe_name="$4"
  local exe_path="${dist_dir}/${exe_name}"
  if [ "${goos}" = "windows" ]; then exe_path="${exe_path}.exe"; fi
  local sha size
  sha="$(sha256_file "${exe_path}")"
  size="$(wc -c < "${exe_path}" | tr -d ' ')"
  cat > "${dist_dir}/manifest.json" <<EOF
{"name":"${APP_NAME}","version":"${VERSION}","version_code":"${VERSION_CODE}","build_time":"${BUILD_TIME}","git_commit":"${GIT_COMMIT}","git_dirty":"${GIT_DIRTY}","goos":"${goos}","goarch":"${goarch}","sha256":"${sha}","size_bytes":${size}}
EOF
}

# 发布 gp-agent 产物到 GitHub / GitCode releases
# 仓库默认 aihop/gopanel（可用 PUBLISH_REPO 覆盖）；tag = v${VERSION}；平台由 PUBLISH_GIT_PLATFORM 控制(all/github/gitcode)
publish_git() {
  local repo="${PUBLISH_REPO:-aihop/gopanel}"
  local tag="v${VERSION}"
  local platform="${PUBLISH_GIT_PLATFORM:-all}"
  local title="GoPanel Agent ${tag}"
  local notes="GoPanel Agent ${tag} 自动发布"
  local assets=("${ARTIFACTS[@]}" "${MANIFESTS[@]}")
  local asset fn

  # --- GitHub ---
  if [[ "${platform}" == "all" || "${platform}" == "github" ]]; then
    if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
      if ! gh release view "${tag}" --repo "${repo}" >/dev/null 2>&1; then
        echo "创建 GitHub Release ${tag} ..."
        gh release create "${tag}" --repo "${repo}" --title "${title}" --notes "${notes}" --draft=false --prerelease=false || true
      fi
      for asset in "${assets[@]}"; do
        echo "GitHub 上传: $(basename "${asset}")"
        gh release upload "${tag}" "${asset}" --repo "${repo}" --clobber || true
      done
      echo "GitHub: https://github.com/${repo}/releases/tag/${tag}"
    else
      echo "跳过 GitHub：gh 未安装或未登录。"
    fi
  fi

  # --- GitCode ---
  if [[ "${platform}" == "all" || "${platform}" == "gitcode" ]]; then
    if [ -n "${GITCODE_TOKEN:-}" ] && command -v jq >/dev/null 2>&1; then
      local info id
      info="$(curl -s -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" "https://api.gitcode.com/api/v5/repos/${repo}/releases/tags/${tag}")"
      id="$(echo "${info}" | jq -r '.id // empty')"
      if [ -z "${id}" ] || [ "${id}" = "null" ]; then
        echo "创建 GitCode Release ${tag} ..."
        local cr
        cr="$(curl -s -X POST "https://api.gitcode.com/api/v5/repos/${repo}/releases" \
          -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" -H "Content-Type: application/json" \
          -d "{\"tag_name\":\"${tag}\",\"ref\":\"main\",\"name\":\"${title}\",\"body\":\"${notes}\",\"release_status\":\"latest\"}")"
        if echo "${cr}" | grep -q '"error_code":'; then
          echo "GitCode 创建 Release 失败，跳过 GitCode: ${cr}"
          return 0
        fi
      fi
      for asset in "${assets[@]}"; do
        fn="$(basename "${asset}")"
        local up upurl
        up="$(curl -s -G -H "PRIVATE-TOKEN: ${GITCODE_TOKEN}" --data-urlencode "file_name=${fn}" "https://api.gitcode.com/api/v5/repos/${repo}/releases/${tag}/upload_url")"
        upurl="$(echo "${up}" | jq -r '.url // empty')"
        if [ -z "${upurl}" ] || [ "${upurl}" = "null" ]; then
          echo "GitCode 获取上传地址失败: ${up}"; continue
        fi
        local copts=()
        if echo "${up}" | jq -e '.headers' >/dev/null 2>&1; then
          while read -r hk hv; do [ -n "${hk}" ] && copts+=("-H" "${hk}: ${hv}"); done \
            < <(echo "${up}" | jq -r '.headers | to_entries | .[] | "\(.key) \(.value)"')
        fi
        echo "GitCode 上传: ${fn}"
        curl -s -X PUT "${copts[@]}" -T "${asset}" "${upurl}" > /dev/null
      done
      echo "GitCode: https://gitcode.com/${repo}/-/releases/${tag}"
    else
      echo "跳过 GitCode：未配置 GITCODE_TOKEN 或缺 jq。"
    fi
  fi
}

# 从 git 记录生成更新内容（HTML）：默认取「上一个 tag..HEAD」的提交标题，取不到则取最近 10 条
git_changelog_html() {
  local last range lines html line
  last="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  range="HEAD"
  [ -n "${last}" ] && range="${last}..HEAD"
  lines="$(git log ${range} --no-merges --pretty=format:'%s' 2>/dev/null | head -30)"
  [ -z "${lines}" ] && lines="$(git log -10 --no-merges --pretty=format:'%s' 2>/dev/null || true)"
  html=""
  while IFS= read -r line; do
    [ -z "${line}" ] && continue
    line="${line//&/&amp;}"; line="${line//</&lt;}"; line="${line//>/&gt;}"
    html="${html}<p>${line}</p>"
  done <<< "${lines}"
  [ -z "${html}" ] && html="<p>GoPanel Agent v${VERSION} 更新</p>"
  echo "${html}"
}

# 发版后向 gopanel.cn 登记 changelog（面板据此自动检测/更新 gp-agent）
# slug 前缀 gp-agent- 用于和主包区分；key=版本名、sort=version_code（比对用）
publish_changelog() {
  local url="${GOPANEL_ADMIN_POSTS_URL:-https://gopanel.cn/api/admin/posts}"
  local key="${GOPANEL_ADMIN_KEY:-}"
  if [ -z "${key}" ]; then
    echo "ERROR: GOPANEL_ADMIN_KEY 未设置（在仓库根 .env 里配置）。" >&2
    return 1
  fi
  if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: 需要 jq 来安全构造 JSON（brew install jq）。" >&2
    return 1
  fi
  if ! command -v curl >/dev/null 2>&1; then
    echo "ERROR: 需要 curl。" >&2
    return 1
  fi

  local slug="gp-agent-v${VERSION//./-}"
  local title="${AGENT_RELEASE_TITLE:-GoPanel Agent v${VERSION}}"
  local desc="${AGENT_RELEASE_DESC:-常规更新}"
  # 更新内容默认从 git 记录提取；也可用 AGENT_RELEASE_CONTENT 覆盖为手写内容
  local content="${AGENT_RELEASE_CONTENT:-$(git_changelog_html)}"

  local body
  body="$(jq -n \
    --arg slug "${slug}" \
    --arg title "${title}" \
    --arg description "${desc}" \
    --arg content "${content}" \
    --arg key "v${VERSION}" \
    --argjson sort "${VERSION_CODE}" \
    '{slug:$slug, title:$title, description:$description, content:$content, type:"changelog", is_active:1, key:$key, sort:$sort, meta_data:"{\"translations\":{\"zh\":{\"title\":\"\",\"description\":\"\",\"content\":\"\"}}}"}')"

  echo ">>> POST ${url}  (slug=${slug}, sort=${VERSION_CODE})"
  local resp http
  resp="$(curl -sS -o /tmp/gp_agent_post_resp.$$ -w '%{http_code}' -X POST "${url}" \
    -H "Authorization: Bearer ${key}" \
    -H "Content-Type: application/json" \
    -d "${body}")" || { echo "ERROR: 请求失败"; return 1; }
  http="${resp}"
  echo "HTTP ${http}"
  cat /tmp/gp_agent_post_resp.$$ 2>/dev/null; echo
  rm -f /tmp/gp_agent_post_resp.$$
  case "${http}" in
    2*) echo "changelog 登记成功" ;;
    *)  echo "WARN: 登记返回非 2xx（若是「已存在」需改用更新接口，或先删旧记录）"; return 1 ;;
  esac
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

ARTIFACTS=()
MANIFESTS=()

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

  write_manifest "${dist_dir}" "${GOOS}" "${GOARCH}" "${APP_NAME}"
  cp "${dist_dir}/manifest.json" "${OUTDIR}/${short_name}.manifest.json"

  # 打包
  tar -C "${OUTDIR}" -czf "${OUTDIR}/${short_name}.tar.gz" "${short_name}"
  echo "Finished: ${short_name}.tar.gz"

  ARTIFACTS+=("${OUTDIR}/${short_name}.tar.gz")
  MANIFESTS+=("${OUTDIR}/${short_name}.manifest.json")
  
  # 清理临时目录
  [ "${KEEP_DIST_DIR:-0}" = "0" ] && rm -rf "${dist_dir}"
done

echo "=== All Done ==="
ls -lh "${OUTDIR}"

PUBLISH_GIT="${PUBLISH_GIT:-}"
if [ -z "${PUBLISH_GIT}" ] && [ -t 1 ]; then
  read -r -p "发布 gp-agent 到 GitHub/GitCode? [y/N] " ans || true
  case "${ans:-}" in
    y|Y|yes|YES) PUBLISH_GIT="1" ;;
    *) PUBLISH_GIT="0" ;;
  esac
fi
if [ "${PUBLISH_GIT:-0}" = "1" ]; then
  echo "=== 发布 gp-agent 到 git ==="
  publish_git
fi

# === 登记 changelog 到 gopanel.cn（面板据此自动更新 gp-agent）===
# 默认不自动触发：交互时询问，或用 PUBLISH_POST=1 显式开启。
PUBLISH_POST="${PUBLISH_POST:-}"
if [ -z "${PUBLISH_POST}" ] && [ -t 1 ]; then
  read -r -p "登记 changelog 到 gopanel.cn（触发全网面板自动更新）? [y/N] " ans2 || true
  case "${ans2:-}" in
    y|Y|yes|YES) PUBLISH_POST="1" ;;
    *) PUBLISH_POST="0" ;;
  esac
fi
PUBLISH_POST="${PUBLISH_POST:-0}"

if [ "${PUBLISH_POST}" = "1" ]; then
  echo "=== 登记 changelog 到 gopanel.cn ==="
  publish_changelog
fi
