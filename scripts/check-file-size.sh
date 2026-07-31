#!/usr/bin/env bash
set -euo pipefail

MAX_LINES="${FILE_SIZE_MAX_LINES:-500}"
BASELINE_PATH="${FILE_SIZE_BASELINE:-.file-size-baseline}"
SOURCE_MODE="worktree"
BASE_REF=""
REPO_ROOT=""

usage() {
  echo "Usage: $0 [--cached] [--base-ref <git-ref>] [--root <repo>]" >&2
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --cached) SOURCE_MODE="index" ;;
    --base-ref)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      BASE_REF="$2"
      shift
      ;;
    --root)
      [ "$#" -ge 2 ] || { usage; exit 2; }
      REPO_ROOT="$2"
      shift
      ;;
    -h|--help) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
  shift
done

if ! [[ "${MAX_LINES}" =~ ^[1-9][0-9]*$ ]]; then
  echo "ERROR: FILE_SIZE_MAX_LINES must be a positive integer" >&2
  exit 2
fi

if [ -z "${REPO_ROOT}" ]; then
  REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
    echo "ERROR: file-size gate must run inside a Git repository" >&2
    exit 2
  }
fi
cd "${REPO_ROOT}"

if [ "${SOURCE_MODE}" = "index" ] && [ -z "${BASE_REF}" ] && git rev-parse --verify HEAD >/dev/null 2>&1; then
  BASE_REF="HEAD"
fi

TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/gopanel-file-size.XXXXXX")"
trap 'rm -rf "${TEMP_DIR}"' EXIT
FILES_LIST="${TEMP_DIR}/files"
BASELINE_FILE="${TEMP_DIR}/baseline"
CURRENT_COUNTS="${TEMP_DIR}/current"
ERRORS_FILE="${TEMP_DIR}/errors"
CURRENT_HAS_BASELINE="false"
INDEX_ROOT=""
: >"${CURRENT_COUNTS}"
: >"${ERRORS_FILE}"

if [ "${SOURCE_MODE}" = "index" ]; then
  INDEX_ROOT="${TEMP_DIR}/index"
  mkdir -p "${INDEX_ROOT}"
  git ls-files --cached -z -- '*.go' '*.vue' '*.ts' >"${FILES_LIST}"
  git checkout-index --prefix="${INDEX_ROOT}/" --stdin -z <"${FILES_LIST}"
  if git cat-file -e ":${BASELINE_PATH}" 2>/dev/null; then
    git show ":${BASELINE_PATH}" >"${BASELINE_FILE}"
    CURRENT_HAS_BASELINE="true"
  else
    : >"${BASELINE_FILE}"
  fi
else
  git ls-files --cached --others --exclude-standard -z -- '*.go' '*.vue' '*.ts' >"${FILES_LIST}"
  if [ -f "${BASELINE_PATH}" ]; then
    cp "${BASELINE_PATH}" "${BASELINE_FILE}"
    CURRENT_HAS_BASELINE="true"
  else
    : >"${BASELINE_FILE}"
  fi
fi

is_checked_file() {
  case "$1" in
    *.go|*.vue|*.ts) ;;
    *) return 1 ;;
  esac
  case "$1" in
    admin/src/i18n/locales/*.ts|admin/src/auto-imports.d.ts|admin/wailsjs/*) return 1 ;;
  esac
  return 0
}

file_line_count() {
  local path="$1"
  if [ "${SOURCE_MODE}" = "index" ]; then
    awk 'END { print NR + 0 }' "${INDEX_ROOT}/${path}"
  else
    awk 'END { print NR + 0 }' "${path}"
  fi
}

baseline_limit() {
  awk -F '\t' -v path="$1" '$1 == path { print $2; exit }' "${BASELINE_FILE}"
}

validate_baseline() {
  awk -F '\t' -v max="${MAX_LINES}" '
    BEGIN { failed = 0; previous = "" }
    NF != 2 || $1 == "" || $2 !~ /^[0-9]+$/ {
      printf "ERROR: invalid baseline entry at line %d\n", NR > "/dev/stderr"; failed = 1; next
    }
    $2 <= max {
      printf "ERROR: baseline limit must exceed %d: %s\n", max, $1 > "/dev/stderr"; failed = 1
    }
    previous != "" && $1 <= previous {
      printf "ERROR: baseline must be unique and path-sorted: %s\n", $1 > "/dev/stderr"; failed = 1
    }
    { previous = $1 }
    END { exit failed }
  ' "${BASELINE_FILE}"
}

validate_baseline

while IFS= read -r -d '' path; do
  is_checked_file "${path}" || continue
  if [ "${SOURCE_MODE}" = "worktree" ] && [ ! -f "${path}" ]; then
    continue
  fi
  count="$(file_line_count "${path}")"
  printf '%s\t%s\n' "${path}" "${count}" >>"${CURRENT_COUNTS}"
  limit="$(baseline_limit "${path}")"
  if [ "${count}" -gt "${MAX_LINES}" ]; then
    if [ -z "${limit}" ]; then
      printf 'ERROR: %s has %s lines (maximum %s)\n' "${path}" "${count}" "${MAX_LINES}" >>"${ERRORS_FILE}"
    elif [ "${count}" -ne "${limit}" ]; then
      printf 'ERROR: %s has %s lines; frozen baseline is %s (reduce it and update the baseline)\n' "${path}" "${count}" "${limit}" >>"${ERRORS_FILE}"
    fi
  elif [ -n "${limit}" ]; then
    printf 'ERROR: %s is now %s lines; remove it from the legacy baseline\n' "${path}" "${count}" >>"${ERRORS_FILE}"
  fi
done <"${FILES_LIST}"

while IFS=$'\t' read -r path limit; do
  [ -n "${path}" ] || continue
  actual="$(awk -F '\t' -v path="${path}" '$1 == path { print $2; exit }' "${CURRENT_COUNTS}")"
  if [ -z "${actual}" ]; then
    printf 'ERROR: legacy baseline path is missing or excluded: %s\n' "${path}" >>"${ERRORS_FILE}"
  fi
done <"${BASELINE_FILE}"

if [ -n "${BASE_REF}" ] && ! git rev-parse --verify "${BASE_REF}^{commit}" >/dev/null 2>&1; then
  echo "ERROR: base ref cannot be resolved: ${BASE_REF}" >&2
  exit 2
fi

if [ -n "${BASE_REF}" ]; then
  BASE_REF_FILE="${TEMP_DIR}/base-ref-baseline"
  BASE_REF_HAS_BASELINE="false"
  if git cat-file -e "${BASE_REF}:${BASELINE_PATH}" 2>/dev/null; then
    git show "${BASE_REF}:${BASELINE_PATH}" >"${BASE_REF_FILE}"
    BASE_REF_HAS_BASELINE="true"
  else
    : >"${BASE_REF_FILE}"
  fi
  if [ "${BASE_REF_HAS_BASELINE}" = "true" ]; then
    if [ "${CURRENT_HAS_BASELINE}" != "true" ]; then
      printf 'ERROR: legacy baseline file cannot be removed: %s\n' "${BASELINE_PATH}" >>"${ERRORS_FILE}"
    fi
    while IFS=$'\t' read -r path limit; do
      [ -n "${path}" ] || continue
      previous="$(awk -F '\t' -v path="${path}" '$1 == path { print $2; exit }' "${BASE_REF_FILE}")"
      if [ -z "${previous}" ]; then
        printf 'ERROR: legacy baseline cannot add a new path: %s\n' "${path}" >>"${ERRORS_FILE}"
      elif [ "${limit}" -gt "${previous}" ]; then
        printf 'ERROR: legacy baseline cannot increase %s from %s to %s\n' "${path}" "${previous}" "${limit}" >>"${ERRORS_FILE}"
      fi
    done <"${BASELINE_FILE}"
  fi
fi

if [ -s "${ERRORS_FILE}" ]; then
  cat "${ERRORS_FILE}" >&2
  echo "File-size gate failed. Split files or reduce the exact legacy baseline." >&2
  exit 1
fi

checked_count="$(awk 'END { print NR + 0 }' "${CURRENT_COUNTS}")"
baseline_count="$(awk 'END { print NR + 0 }' "${BASELINE_FILE}")"
echo "File-size gate passed: ${checked_count} files checked, ${baseline_count} legacy files frozen, max ${MAX_LINES} lines."
