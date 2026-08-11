# Security Policy

[简体中文](#安全策略)

GoPanel is a server control plane. It manages websites, containers, databases and
runs AI-assisted development tasks on the host, so a vulnerability here can mean
full host compromise. We take reports seriously and would rather hear about a
false alarm than miss a real issue.

## Reporting a vulnerability

**Do not open a public issue for security problems.**

Use one of these private channels:

- GitHub [private vulnerability reporting](https://github.com/aihop/gopanel/security/advisories/new)
  (preferred — it keeps the report, the fix and the advisory in one place)
- Email `security@gopanel.cn`

Please include:

- affected version (`gopanel --version` or the release tag) and platform
- what an attacker can achieve, not only what looks wrong
- reproduction steps or a proof of concept
- whether the issue is already public anywhere

We aim to acknowledge within 3 working days and to ship a fix or a documented
mitigation within 30 days for high-severity issues. If you plan to publish,
please give us that window first; we will credit you in the advisory unless you
prefer to stay anonymous.

## Supported versions

Only the latest released version receives security fixes. There are no
long-term-support branches — upgrade before reporting an issue against an older
build.

## Security model

Knowing what GoPanel assumes helps decide whether a finding is a vulnerability
or intended behaviour.

**Privileges.** The panel needs root-equivalent access to do its job: managing
system services, containers, firewall rules and certificates. Privileged
operations are funnelled through the `gpc` helper and the `gp-agent` host agent
over local sockets rather than executed inline. A bug that lets an unprivileged
panel user reach those sockets **is** a vulnerability.

**Network exposure.** GoPanel is meant to sit behind an authenticated entrance
path and, ideally, not be exposed to the public internet. Reports that require
the panel to already be publicly reachable with default credentials are
configuration problems, not vulnerabilities — but a way to bypass the entrance
path or authentication is.

**AI executors and remote execution.** AI development sessions run real commands
in real working directories. This is the product, not a flaw. What matters is
the boundary: sessions run in isolated Git worktrees, high-risk actions go
through an approval flow, and every remote execution is audited. Escaping the
approval flow, running commands outside the session workspace, or acting as
another user **is** a vulnerability.

**Credential storage.** Git credentials and provider API keys are encrypted at
rest and decrypted only for the duration of the command that needs them. They
must never appear in logs, in the terminal stream, or in a Git remote URL.
Anything that leaks them is a vulnerability.

**Supply chain.** In-app upgrades verify the package SHA-256 before installing.
If you find a way to make the panel install an unverified or substituted
artifact, report it — that is exactly the class of issue we want to hear about.

## Out of scope

- Missing hardening headers or TLS configuration on a deployment you control
- Denial of service through resource exhaustion by an already-authenticated admin
- Findings that require physical access or an already-root shell on the host
- Automated scanner output without a demonstrated impact

---

# 安全策略

GoPanel 是服务器控制面板，管理网站、容器、数据库，并在宿主机上执行 AI 开发任务。
这里的漏洞往往意味着整台主机失守。我们宁可收到误报，也不希望漏掉真问题。

## 如何报告

**请不要用公开 issue 报告安全问题。**

请走私密渠道：

- GitHub [私密漏洞报告](https://github.com/aihop/gopanel/security/advisories/new)（推荐，报告、修复和公告在一处）
- 邮件 `security@gopanel.cn`

请尽量包含：受影响版本与平台、攻击者**能达成什么**（而不只是哪里看着不对）、
复现步骤或 PoC、以及该问题是否已在别处公开。

我们争取 3 个工作日内确认，高危问题 30 天内给出修复或明确的缓解方案。
如果你打算公开，请先留出这个窗口；除非你希望匿名，我们会在公告中致谢。

## 支持范围

只有最新发布版本会收到安全修复，没有长期支持分支。

## 安全模型

先说清楚 GoPanel 的假设，便于判断某个发现算不算漏洞。

**权限**：面板需要 root 等价权限才能管理系统服务、容器、防火墙和证书。
特权操作统一走 `gpc` 助手与 `gp-agent` 宿主机代理的本地套接字，而不是就地执行。
**如果低权限的面板用户能触达这些套接字，那是漏洞。**

**网络暴露**：GoPanel 应当置于带认证的入口路径之后，且不建议直接暴露在公网。
"面板已公网可达且用默认口令"这类前提属于配置问题；但**绕过入口路径或认证**是漏洞。

**AI 执行器与远程执行**：AI 会话会在真实目录里执行真实命令，这是产品能力而非缺陷。
关键在边界：会话跑在隔离的 Git worktree 中，高风险动作走审批流，远程执行全部审计。
**绕过审批、越出会话工作区执行命令、或以他人身份操作，是漏洞。**

**凭据存储**：Git 凭据和模型 API Key 加密存储，仅在需要它的命令执行期间解密。
它们不应出现在日志、终端流或 Git 远端 URL 中。**任何导致泄露的路径都是漏洞。**

**供应链**：程序内升级会在安装前校验安装包 SHA-256。
如果你能让面板安装未经校验或被替换的产物，请务必报告——这正是我们最关心的一类问题。

## 不在范围内

- 你自己部署环境上缺失的加固响应头或 TLS 配置
- 已认证管理员通过资源耗尽制造的拒绝服务
- 需要物理访问或已取得宿主机 root 权限才能触发的问题
- 没有展示实际影响的扫描器输出
