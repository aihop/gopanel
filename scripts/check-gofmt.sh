#!/usr/bin/env bash
set -euo pipefail

# gofmt 增量闸门：只检查本次改动涉及的 Go 文件。
#
# 刻意不做全仓校验：现存 50 多个文件未格式化，一次性重排会产生与改动无关的巨大 diff，
# 还会和并行开发的会话撞车。增量意味着新写的代码必须干净，存量欠账由下一个
# 动到该文件的人顺手还掉。
#
# 用法：
#   --cached           校验暂存区里的 Go 文件（pre-commit 钩子走这条）
#   --base-ref <ref>   校验 <ref>..HEAD 改动的 Go 文件（CI 走这条）
# 不给参数时校验工作区相对 HEAD 的改动。

MODE="worktree"
BASE_REF=""

usage() {
  echo "Usage: $0 [--cached] [--base-ref <git-ref>]" >&2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --cached) MODE="cached" ;;
    --base-ref)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      MODE="range"
      BASE_REF="$2"
      shift
      ;;
    -h|--help) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
  shift
done

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

case "${MODE}" in
  cached) changed="$(git diff --cached --name-only --diff-filter=ACMR)" ;;
  range)
    if ! git rev-parse --verify --quiet "${BASE_REF}^{commit}" >/dev/null; then
      echo "gofmt gate skipped: base ref ${BASE_REF} is unavailable"
      exit 0
    fi
    changed="$(git diff --name-only --diff-filter=ACMR "${BASE_REF}..HEAD")"
    ;;
  *)
    # 新建但还没 git add 的文件不在 `git diff HEAD` 里，而它们恰恰是最可能没格式化的。
    changed="$(
      git diff --name-only --diff-filter=ACMR HEAD
      git ls-files --others --exclude-standard
    )"
    ;;
esac

# 链接工作区（.codux/、其他 worktree）住在仓库目录下但不属于本次改动，
# gofmt 走进去只会报出别人的文件。
targets=()
while IFS= read -r path; do
  case "${path}" in
    *.go) ;;
    *) continue ;;
  esac
  case "${path}" in
    .codux/*) continue ;;
  esac
  [ -f "${path}" ] || continue
  targets+=("${path}")
done <<EOF
${changed}
EOF

if [ "${#targets[@]}" -eq 0 ]; then
  echo "gofmt gate passed: no Go files changed."
  exit 0
fi

unformatted="$(gofmt -l "${targets[@]}")"
if [ -n "${unformatted}" ]; then
  echo "ERROR: 以下改动涉及的文件未经 gofmt 格式化：" >&2
  echo "${unformatted}" >&2
  echo "" >&2
  echo "修复：gofmt -w ${unformatted//$'\n'/ }" >&2
  exit 1
fi

echo "gofmt gate passed: ${#targets[@]} file(s) checked."
