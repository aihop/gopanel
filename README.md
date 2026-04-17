# GoPanel

_[简体中文 (Simplified Chinese)](./README_zh.md)_

Official website: https://gopanel.cn/

GoPanel is a modern, lightweight server management panel built for developers and small teams. It is also a Docker-first application runtime and website deployment platform. Instead of stuffing every possible feature into one bloated backend, GoPanel focuses on the operational actions you actually do every day and turns them into a cleaner, faster, and easier-to-understand workflow.

GoPanel helps you unify:

- website hosting (static sites / reverse proxy / containerized web apps)
- containers and application lifecycle (start / stop / logs / resources)
- databases and middleware (one-click install and management)
- CI/CD pipelines (fetch code → build → publish → switch versions)
- certificates / domains / HTTPS workflows
- AI-assisted collaboration and troubleshooting (built into the panel)

Whether you want to:

- bring up application environments on a fresh server quickly
- manage databases, caches, and services with containers
- unify static sites, reverse proxies, and containerized web apps behind one control plane
- build, publish, and switch versions through a pipeline workflow
- handle AI assistance, app installation, logs, and daily operations from one panel

GoPanel is designed to make those actions feel direct, instead of forcing you to navigate through overloaded menus and fragmented tools.

## ⚡ Quick Start

```bash
bash <(curl -fsSL https://gopanel.run)
```

## 🖼 Preview

![GoPanel Preview](./preview.png)

The screenshot above reflects the current GoPanel product direction: consistent information density, clean functional zoning, a workspace-style layout, and an integrated operating experience built around hosts, websites, containers, pipelines, apps, and AI assistance.

## 🌟 Why GoPanel

Compared with traditional server panels, GoPanel puts much more emphasis on the following:

- **Lower cognitive overhead**: GoPanel keeps the focus on the modules that matter most in real operations work: websites, containers, apps, databases, pipelines, logs, and security settings.
- **Docker-first runtime model**: Databases, middleware, web apps, and service environments are naturally isolated, making migration, rollback, backup, and multi-environment management more predictable.
- **Built-in HTTP service and unified access entry**: The panel can serve itself without requiring an external web server just to get started. Combined with website management, reverse proxying, and entry routing, it can naturally take over traffic for static sites, web apps, and domains.
- **Process guarding for long-running stability**: GoPanel includes a long-running service mindset out of the box, with support for `systemd`, runtime guarding, and restart-friendly operation, so it behaves like a real production panel rather than a disposable script.
- **Automatic CDN and certificate lifecycle**: It goes beyond basic HTTPS by connecting domain resolution, certificate issuance, renewal, and CDN push workflows into a more complete production delivery path.
- **Built-in database management workspace**: It does not stop at launching database containers. You can also manage databases, table structures, connection info, and common maintenance tasks directly inside the panel.
- **One continuous path from build to release**: GoPanel does not only manage running services. It also connects code fetching, script execution, artifact generation, and website deployment into one coherent build-and-release chain.
- **Single-binary architecture that scales well operationally**: Built with Golang + Vue, with frontend assets bundled into the application, it keeps deployment simple, dependencies low, and upgrades easy to maintain.
- **Seamless self-updating**: Deep GitHub Releases integration lets the panel check, download, replace, and restart itself with minimal manual work while keeping configs and databases safe.
- **Cross-platform friendly**: Linux is the natural production environment, while macOS and Windows also work well for local development, debugging, and demos.

## ✨ What You Get

With GoPanel, you get a workspace designed around deployment efficiency and operational order:

- **Dashboard**: Quickly understand CPU, memory, disk, network, and popular app status from a single overview page.
- **AI Assistant**: Organize AI sessions by workspace group and keep commands, troubleshooting, collaboration, and context inside the panel.
- **Website system**: Manage static sites, reverse proxies, containerized apps, and pipeline-driven publishing in one place.
- **HTTP and access entry control**: Use the built-in HTTP service, unified entry configuration, and domain integration without having to wrap the panel in another external web server first.
- **Database and container management**: Inspect runtime status, ports, versions, logs, and resource usage, while keeping database operations inside the same control plane.
- **Certificate / CDN / domain workflow**: Manage domain access, certificate issuance, renewal, DNS flow, and CDN push as one continuous operational workflow.
- **Process guarding and runtime management**: Keep long-running services and the panel itself in a more stable, production-friendly state.
- **Pipeline workspace**: Fetch code, build artifacts, review execution history, inspect logs, and publish versions through a lightweight CI/CD path.
- **App marketplace**: Install common services quickly to bootstrap environments, databases, middleware, and frequently used business components.

## 🧭 Product Philosophy

GoPanel is built around three very explicit principles:

- **Less, but sharper**: It avoids bloated “do everything” interfaces and focuses on making high-frequency tasks feel smooth.
- **One operational entry**: It pulls actions that are usually spread across SSH, Docker, reverse proxies, scripts, and deployment logs into one consistent UI path.
- **Developer-friendly by design**: It works for quick setup, but also keeps enough engineering depth for long-term delivery, multi-environment operation, and iterative releases.

## 🚀 One-Click Install & Update

GoPanel uses the exact same simple command for both fresh installations and future upgrades. The script automatically detects your environment: guiding you through setup if it's new, or silently pulling the latest version and updating seamlessly if already installed.

```bash
bash <(curl -fsSL https://gopanel.run)
```

**Compatibility**:

- **Linux**: Installs to `/opt/gopanel` by default and configures `systemd` for auto-start.
- **macOS / Windows**: Installs to `~/.gopanel` by default, perfect as a local development tool.

## 🛠 Development & Build

- **Environment**: Go 1.25.1 / Node.js 20+
- **Dependencies**: `go mod tidy`
- **Run**: `go run main.go`

### Run GoPanel locally on your machine

```bash
# 1. Compile for all platforms (outputs to dist/)
git clone https://github.com/aihop/gopanel.git

cd gopanel

go run main.go
```
