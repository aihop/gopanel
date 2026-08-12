# Contributing to GoPanel

[简体中文](#参与-gopanel-开发)

Thanks for taking the time. This guide covers what you need to build, test and
land a change.

Security issues do **not** belong here — see [SECURITY.md](./SECURITY.md).

## Repository layout

| Path | What lives there |
| --- | --- |
| `app/` | Go backend: `api/` handlers, `service/`, `repo/`, `model/` |
| `admin/` | Vue 3 + TypeScript web console |
| `client/` | Flutter mobile client |
| `gpc/`, `gp-agent/` | Privileged helper and host agent (separate Go modules) |
| `scripts/` | Repo tooling, including the file-size gate |

## Prerequisites

- Go 1.25.1
- Node.js 20 (the console declares `>=18`, CI runs 20)
- Flutter 3.41.8 — only if you touch `client/`

## Local configuration

`conf.dev.yaml` is ignored by Git — it holds machine-specific paths and the
instance encryption key. Copy the template to get started:

```bash
cp conf.dev.example.yaml conf.dev.yaml
```

Leave `encrypt_key` empty: the panel generates a 32-character random key on
first start. That key also signs JWTs, so never share it between instances and
never commit it.

## Running the checks

CI runs exactly these. Run them locally before opening a PR:

```bash
# Go — three separate modules
go test ./... -count=1
(cd gpc && go test ./... -count=1)
(cd gp-agent && go test ./... -count=1)
go run ./tools/contractcheck        # AI task contract validation

# Web console
cd admin
npm ci
npm run test:unit
npm run type-check
npm run lint

# Mobile client (only if you changed it)
cd client && flutter pub get && flutter test

# Gates (see below)
bash scripts/check-file-size.sh
bash scripts/check-gofmt.sh          # only the Go files you changed
bash scripts/check-commit-message.sh # first line of HEAD
```

If the file-size gate reports a baseline ordering error, run it under
`LC_ALL=C` — the baseline is sorted with byte collation.

Install the hooks once and the first two run on every commit:

```bash
bash scripts/install-git-hooks.sh
```

## House rules

These are enforced in review, and some in CI.

**File size.** No source file over **500 lines**. The gate fails the build and
the exceptions in `.file-size-baseline` are frozen legacy files, not a place to
add new entries. Split by responsibility, not by line count.

**Reuse before adding.** Read how a neighbouring module solves the same problem
and follow it. Two ways of doing one thing in the same codebase is worse than
either way on its own.

**Translate every user-facing string.** No hardcoded Chinese or English in
templates — use `t()` keys and add both locales.

**Cover loading, error and empty states.** A frontend API call that only handles
the success path is incomplete. Never swallow an exception silently; surface it.

**Wrap backend responses.** Return through `e.Succ()` / `e.Fail()`, not raw
`c.JSON`. Don't use `panic` in place of an error return.

**Keep the diff to the task.** Don't reformat files you didn't otherwise change
— it buries the real change. If your editor reformats on save, revert the noise
before committing.

**Say why, not what, in comments.** The code already says what it does. Comments
should explain the constraint or the trade-off that made it look like this.

## Commits and pull requests

Commit messages use `type: summary` on the first line (`feat` / `fix` /
`refactor` / `style` / `docs` / `chore`), followed by `-` bullets for the
notable points. Write them in the language you think in; the history is mixed
and that is fine.

One feature or one fix per commit. Squash noise before opening the PR.

In the PR description, say what changed and how you verified it. "Tests pass" is
weaker than "added a regression test that fails without the fix" — and the
second is what gets a change merged quickly.

## Reporting bugs

Open an issue with the version, platform, what you expected, what happened, and
the smallest reproduction you can manage. Logs and screenshots help. If you are
not sure whether it is a bug or a configuration problem, open it anyway and say
so.

---

# 参与 GoPanel 开发

安全问题**不要**走这里，见 [SECURITY.md](./SECURITY.md)。

## 仓库结构

| 路径 | 内容 |
| --- | --- |
| `app/` | Go 后端：`api/` 处理器、`service/`、`repo/`、`model/` |
| `admin/` | Vue 3 + TypeScript 管理台 |
| `client/` | Flutter 移动端 |
| `gpc/`、`gp-agent/` | 特权助手与宿主机代理（独立 Go 模块） |
| `scripts/` | 仓库工具，含文件大小门禁 |

## 环境要求

- Go 1.25.1
- Node.js 20（管理台声明 `>=18`，CI 用 20）
- Flutter 3.41.8 —— 仅在改动 `client/` 时需要

## 本地配置

`conf.dev.yaml` 已被 Git 忽略——它含本机路径和实例加密密钥。从模板开始：

```bash
cp conf.dev.example.yaml conf.dev.yaml
```

`encrypt_key` 留空即可，首次启动会自动生成 32 位随机密钥。
这个 key 同时用于 JWT 签名，不要在多个实例间共用，也不要提交。

## 本地检查

CI 跑的就是下面这些，提 PR 前先在本地过一遍：

```bash
# Go —— 三个独立模块
go test ./... -count=1
(cd gpc && go test ./... -count=1)
(cd gp-agent && go test ./... -count=1)
go run ./tools/contractcheck        # AI 任务契约校验

# 管理台
cd admin
npm ci
npm run test:unit
npm run type-check
npm run lint

# 移动端（改了才需要）
cd client && flutter pub get && flutter test

# 各项门禁（见下）
bash scripts/check-file-size.sh
bash scripts/check-gofmt.sh          # 只查你改动的 Go 文件
bash scripts/check-commit-message.sh # 校验 HEAD 的首行
```

如果门禁报 baseline 排序错误，加 `LC_ALL=C` 再跑一次——baseline 是按字节序排的。

装一次钩子，前两项每次提交自动跑：

```bash
bash scripts/install-git-hooks.sh
```

## 项目约定

以下在评审中执行，部分由 CI 强制。

**文件粒度**：单个源文件不超过 **500 行**。门禁会直接让构建失败；
`.file-size-baseline` 里的例外是冻结的历史文件，**不是用来加新条目的**。
按职责拆分，而不是按行数硬切。

**先复用再新增**：先看邻近模块怎么解决同类问题，保持一致。
同一个代码库里两套写法，比其中任何一套单独存在都糟。

**所有面向用户的文案都要翻译**：模板里不硬编码中文或英文，用 `t()` key，中英两边都加。

**覆盖 loading / error / empty 三种状态**：前端只写成功路径的接口调用是不完整的。
不要静默吞掉异常，要给用户反馈。

**后端响应统一包装**：走 `e.Succ()` / `e.Fail()`，不要直接 `c.JSON`；
不要用 `panic` 代替错误返回。

**diff 只包含本次任务**：不要顺手重排没改动的文件——那会把真正的改动埋掉。
如果编辑器保存时自动格式化，提交前把噪音还原掉。

**注释写「为什么」而不是「做了什么」**：代码本身已经说明做了什么，
注释应当解释是什么约束或权衡让它长成这样。

## 提交与 PR

提交信息首行用 `type: 概括`（`feat` / `fix` / `refactor` / `style` / `docs` / `chore`），
其后用 `-` 列出关键改点。用你思考时用的语言写即可，历史本来就是混合的。

一个功能或一个修复一次提交，提 PR 前把噪音压掉。

PR 描述里写清楚改了什么、怎么验证的。「测试通过」的说服力，
远不如「加了一条回归测试，不打这个补丁它就失败」——后者能让改动更快被合并。

## 报告缺陷

开 issue 时请附上版本、平台、预期行为、实际行为，以及你能给出的最小复现。
日志和截图很有帮助。不确定是缺陷还是配置问题也可以开，说明一下即可。
