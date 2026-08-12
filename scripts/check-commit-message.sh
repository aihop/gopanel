#!/usr/bin/env bash
set -euo pipefail

# 提交信息规范见 CONTRIBUTING.md：首行 `type: 概括`。
#
# 这条闸门的动机很具体：publish.sh 直接用 `git log --pretty=%s` 拼发布说明，
# 「管理员集中提交」「调整 code」这类提交会原样出现在用户看到的版本日志里。
#
# 两种用法：
#   --file <路径>      校验单条提交信息（commit-msg 钩子走这条）
#   --base-ref <ref>   校验 <ref>..HEAD 区间内的提交（CI 走这条）
# 不给参数时校验 HEAD。

TYPES="feat|fix|refactor|style|docs|chore|test|perf|build|ci|revert|merge"

# Git 自己生成的信息不受规范约束：合并、回滚、rebase 的 fixup/squash 标记
# 都是工具产物，改写它们只会让 `git log --merges` 之类的检索失效。
EXEMPT_PATTERN='^(Merge (branch|commit|tag|remote-tracking|pull request)|Revert "|fixup! |squash! |amend! )'

MODE="head"
TARGET=""

usage() {
  echo "Usage: $0 [--file <path>] [--base-ref <git-ref>]" >&2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --file)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      MODE="file"
      TARGET="$2"
      shift
      ;;
    --base-ref)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      MODE="range"
      TARGET="$2"
      shift
      ;;
    -h|--help) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
  shift
done

# 取提交信息的首行：跳过空行和注释行。
# Git 会剥掉前导空行，所以「第一行」和「第一条非空非注释行」是一回事；
# 而 `git commit --verbose` 会把整段 diff 附在注释里，必须排除掉。
subject_of() {
  awk '
    /^#/ { next }
    /^[[:space:]]*$/ { next }
    { print; exit }
  ' "$1"
}

failures=0

check_subject() {
  local subject="$1"
  local label="$2"

  if [ -z "${subject}" ]; then
    echo "ERROR: ${label} 提交信息为空" >&2
    failures=$((failures + 1))
    return
  fi

  if printf '%s' "${subject}" | grep -qE "${EXEMPT_PATTERN}"; then
    return
  fi

  if printf '%s' "${subject}" | grep -qE '^[[:space:]]'; then
    echo "ERROR: ${label} 首行有前导空格：${subject}" >&2
    failures=$((failures + 1))
    return
  fi

  if ! printf '%s' "${subject}" | grep -qE "^(${TYPES})(\([^)]+\))?: .+"; then
    echo "ERROR: ${label} 首行不符合 \`type: 概括\`：${subject}" >&2
    failures=$((failures + 1))
    return
  fi
}

case "${MODE}" in
  file)
    [ -f "${TARGET}" ] || { echo "ERROR: 提交信息文件不存在：${TARGET}" >&2; exit 2; }
    check_subject "$(subject_of "${TARGET}")" "本次提交"
    ;;
  range)
    if ! git rev-parse --verify --quiet "${TARGET}^{commit}" >/dev/null; then
      # 首次推送分支时基线可能还不存在，这时没有可校验的区间，直接放行。
      echo "Commit-message gate skipped: base ref ${TARGET} is unavailable"
      exit 0
    fi
    while read -r commit; do
      [ -n "${commit}" ] || continue
      check_subject "$(git log -1 --format=%s "${commit}")" "${commit:0:8}"
    done <<EOF
$(git rev-list "${TARGET}..HEAD")
EOF
    ;;
  head)
    check_subject "$(git log -1 --format=%s HEAD)" "HEAD"
    ;;
esac

if [ "${failures}" -gt 0 ]; then
  cat >&2 <<'HINT'

首行格式：type: 概括
可用 type：feat fix refactor style docs chore test perf build ci revert merge
规范见 CONTRIBUTING.md。确需跳过时用 git commit --no-verify。
HINT
  exit 1
fi

echo "Commit-message gate passed."
