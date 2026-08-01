#!/usr/bin/env bash
set -euo pipefail

required_commands=(curl jq)
for command_name in "${required_commands[@]}"; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "缺少依赖命令: ${command_name}" >&2
    exit 2
  fi
done
if ! command -v md5 >/dev/null 2>&1 && ! command -v md5sum >/dev/null 2>&1; then
  echo "缺少依赖命令: md5 或 md5sum" >&2
  exit 2
fi

required_variables=(GOPANEL_URL GOPANEL_API_TOKEN PIPELINE_ID PIPELINE_VERSION WEBSITE_ID)
for variable_name in "${required_variables[@]}"; do
  if [ -z "${!variable_name:-}" ]; then
    echo "缺少环境变量: ${variable_name}" >&2
    exit 2
  fi
done

POLL_INTERVAL="${POLL_INTERVAL:-3}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-1800}"
UPSTREAM_SCHEME="${UPSTREAM_SCHEME:-http}"

if ! [[ "${PIPELINE_ID}" =~ ^[1-9][0-9]*$ && "${WEBSITE_ID}" =~ ^[1-9][0-9]*$ ]]; then
  echo "PIPELINE_ID 和 WEBSITE_ID 必须是正整数" >&2
  exit 2
fi
if ! [[ "${POLL_INTERVAL}" =~ ^[1-9][0-9]*$ && "${TIMEOUT_SECONDS}" =~ ^[1-9][0-9]*$ ]]; then
  echo "POLL_INTERVAL 和 TIMEOUT_SECONDS 必须是正整数" >&2
  exit 2
fi
if [ "${UPSTREAM_SCHEME}" != "http" ] && [ "${UPSTREAM_SCHEME}" != "https" ]; then
  echo "UPSTREAM_SCHEME 只能是 http 或 https" >&2
  exit 2
fi

GOPANEL_URL="${GOPANEL_URL%/}"
case "${GOPANEL_URL}" in
  */api) API_BASE="${GOPANEL_URL}" ;;
  *) API_BASE="${GOPANEL_URL}/api" ;;
esac

api_request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local timestamp signature response

  timestamp="$(date +%s)"
  if command -v md5 >/dev/null 2>&1; then
    signature="$(printf 'gopanel_%s_%s' "${GOPANEL_API_TOKEN}" "${timestamp}" | md5)"
  else
    signature="$(printf 'gopanel_%s_%s' "${GOPANEL_API_TOKEN}" "${timestamp}" | md5sum | awk '{print $1}')"
  fi
  if [ -n "${body}" ]; then
    response="$(curl --fail-with-body --silent --show-error \
      --request "${method}" \
      --header "apiKey: ${signature}" \
      --header "timestamp: ${timestamp}" \
      --header "Content-Type: application/json" \
      --data "${body}" \
      "${API_BASE}${path}")"
  else
    response="$(curl --fail-with-body --silent --show-error \
      --request "${method}" \
      --header "apiKey: ${signature}" \
      --header "timestamp: ${timestamp}" \
      "${API_BASE}${path}")"
  fi

  if ! jq -e '.code == 0' >/dev/null 2>&1 <<<"${response}"; then
    echo "接口调用失败 ${method} ${path}: $(jq -r '.msg // .' <<<"${response}" 2>/dev/null || printf '%s' "${response}")" >&2
    return 1
  fi
  printf '%s' "${response}"
}

run_body="$(jq -cn \
  --argjson id "${PIPELINE_ID}" \
  --arg version "${PIPELINE_VERSION}" \
  '{id: $id, version: $version}')"
run_response="$(api_request POST '/pipeline/run' "${run_body}")"
record_id="$(jq -r '.data.recordId // empty' <<<"${run_response}")"
if ! [[ "${record_id}" =~ ^[1-9][0-9]*$ ]]; then
  echo "触发流水线成功，但响应中缺少有效的 recordId" >&2
  exit 1
fi
echo "已触发流水线: pipelineId=${PIPELINE_ID}, recordId=${record_id}"

started_at="$(date +%s)"
while true; do
  encoded_pipeline_id="$(jq -rn --arg value "${PIPELINE_ID}" '$value|@uri')"
  records_response="$(api_request GET "/pipeline/records?pipelineId=${encoded_pipeline_id}&page=1&limit=100")"
  record="$(jq -c --argjson record_id "${record_id}" '.data.items[]? | select(.id == $record_id)' <<<"${records_response}" | head -n 1)"
  if [ -z "${record}" ]; then
    echo "等待运行记录出现: recordId=${record_id}"
  else
    status="$(jq -r '.status // empty' <<<"${record}")"
    case "${status}" in
      success)
        container_id="$(jq -r '.runnerContainerId // empty' <<<"${record}")"
        host_port="$(jq -r '.runnerHostPort // 0' <<<"${record}")"
        runtime_host="$(jq -r '.runtimeHost // empty' <<<"${record}")"
        if [ -z "${container_id}" ] || ! [[ "${host_port}" =~ ^[1-9][0-9]*$ ]]; then
          echo "流水线已成功，但没有 Runner 容器信息；请确认流水线已启用 Runner 模式" >&2
          exit 1
        fi
        break
        ;;
      failed)
        error_message="$(jq -r '.errorMessage // "流水线执行失败"' <<<"${record}")"
        echo "流水线失败: ${error_message}" >&2
        exit 1
        ;;
      *) echo "流水线状态: ${status:-unknown}" ;;
    esac
  fi

  now="$(date +%s)"
  if [ "$((now - started_at))" -ge "${TIMEOUT_SECONDS}" ]; then
    echo "等待流水线超时: ${TIMEOUT_SECONDS}s" >&2
    exit 1
  fi
  sleep "${POLL_INTERVAL}"
done

bind_body="$(jq -cn \
  --arg container_id "${container_id}" \
  --arg runtime_host "${runtime_host}" \
  --argjson website_id "${WEBSITE_ID}" \
  --argjson host_port "${host_port}" \
  --arg scheme "${UPSTREAM_SCHEME}" \
  '{containerId: $container_id, runtimeHost: $runtime_host, websiteId: $website_id, hostPort: $host_port, scheme: $scheme}')"
api_request POST '/container/bind-website' "${bind_body}" >/dev/null

echo "绑定完成: containerId=${container_id}, hostPort=${host_port}, websiteId=${WEBSITE_ID}"
