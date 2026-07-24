# Debug Session: cloudflare-dns01
- **Status**: [OPEN]
- **Issue**: Cloudflare DNS-01 证书签发时一直等待传播，最终失败，需确认是 TXT 记录未写入还是写入后不可见
- **Debug Server**: Pending
- **Log File**: .dbg/trae-debug-log-cloudflare-dns01.ndjson

## Reproduction Steps
1. 在 SSL 签发页选择 Cloudflare DNS 提供商
2. 对 `assets.fastd.cn` 发起 DNS-01 签发
3. 观察日志出现 `Waiting for DNS record propagation`
4. 最终在清理 challenge 后报错结束

## Hypotheses & Verification
| ID | Hypothesis | Likelihood | Effort | Evidence |
|----|------------|------------|--------|----------|
| A | Cloudflare API 调用失败，TXT 记录根本没创建成功 | High | Low | Pending |
| B | TXT 记录创建成功，但 Cloudflare Zone/Record 名称计算错误，写到了错误位置 | High | Medium | Pending |
| C | TXT 记录创建成功，但传播检查仍走本地 DNS `192.168.1.1` / `fe80::1`，导致看不到外部权威结果 | High | Low | Pending |
| D | Cloudflare 凭证权限不足，创建/删除记录部分失败但日志被吞掉 | Medium | Medium | Pending |
| E | 代码在创建后立刻清理 challenge，真实原因是 propagation timeout 过短或 resolver 配置不对 | Medium | Medium | Pending |

## Log Evidence
- 已有用户日志显示：acme 使用本地解析器 `[[fe80::1%en0]:53 192.168.1.1:53]` 检查传播，并在 1 分钟超时后清理 challenge
- `app/service/ssl_acme.go` 仅调用 `client.Challenge.SetDNS01Provider(provider)`，未设置 `dns01.AddRecursiveNameservers(...)` 或自定义传播超时
- `go-acme/lego/v4@v4.14.2` 默认传播超时为 60 秒，默认递归解析器来自本机 DNS 配置
- `pkg/cloudflare/acme_provider.go` 中 `AddTxtRecord` 若 Cloudflare API 请求失败会直接返回 error；当前日志已进入 `Trying to solve DNS-01` 和长时间 `Waiting for DNS record propagation`，说明创建记录阶段至少没有直接抛出 HTTP 级错误
- Cloudflare token 在前端保存为 `authorization.token`，后端 `pkg/cloud/provider.go` 正确读取该字段构造 provider
- 已新增诊断增强：
  - `pkg/cloudflare/client.go` 会输出 Cloudflare API path、status、errors/messages 明细
  - `pkg/cloudflare/acme_provider.go` 会输出 zone、record、fqdn、cleanup 上下文
  - `app/service/ssl_acme.go` 会输出原始 ACME 错误，并给出权限、Zone、传播、本地 DNS resolver 等针对性诊断建议

## Verification Conclusion
- A | Cloudflare API 调用失败，TXT 记录根本没创建成功 | ⏳ 暂未完全排除，但不是当前首要嫌疑
- B | TXT 记录创建成功，但 Cloudflare Zone/Record 名称计算错误，写到了错误位置 | ⏳ 可能性中等，当前代码按 lego 的 `FindZoneByFqdn` + `GetRecord` 生成，未见明显拼接错误
- C | TXT 记录创建成功，但传播检查仍走本地 DNS `192.168.1.1` / `fe80::1`，导致看不到外部权威结果 | ✅ 当前最符合现象
- D | Cloudflare 凭证权限不足，创建/删除记录部分失败但日志被吞掉 | ⏳ 有一定风险，因 `doCloudflareRequest` 仅检查 HTTP status，未校验 JSON `success`
- E | challenge 被正常创建，但 propagation timeout 过短或 resolver 配置不对 | ✅ 当前最符合现象
