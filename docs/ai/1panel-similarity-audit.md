# GoPanel 与 1Panel 当前源码相似文件审计基线

> 本文是工程审计基线，不是法律意见，也不直接判定代码权属。后续重写、许可证核查和发布审查均以本清单为入口。

## 对比快照

- GoPanel：当前工作区，基准提交 `9cbddbec597d2d88261e2ff8f65e349d4a7293f3`（2026-08-01）。
- 1Panel：默认分支提交 `9159ab842d956dc802cbd3340b1e43d1e6406fbc`（2026-07-31）。
- 统计范围：双方受 Git 管理的 `.go`、`.vue`、`.ts`、`.tsx`、`.js`、`.jsx`、`.sh` 源文件；GoPanel 额外包含当前未跟踪源码。
- 排除范围：构建产物、依赖目录、静态发布目录。

## 统计口径

1. 去除空行和独立注释行，统一空白、大小写以及 `GoPanel`、`1Panel`、`FIT2CLOUD` 等品牌标识。
2. 以连续 8 个有效代码行为窗口，与 1Panel 当前源码建立匹配。
3. 文件有效代码不少于 20 行、命中比例不少于 20% 时进入主清单。
4. 完全相同文件不受 20 行门槛限制，另行完整列出。
5. 相似率是审计线索，不代表整个文件逐字相同；通用样板可能造成少量误报，重写前仍需人工逐文件确认。

## 汇总

| 指标 | 结果 |
|---|---:|
| GoPanel 纳入统计的源码 | 1,448 个文件，约 212,902 行 |
| 连续 8 行保守口径 | 8,289 个有效代码行，约 4.33% |
| 连续 5 行宽松口径 | 12,550 个有效代码行，约 6.55% |
| 主清单 | 110 个文件 |
| 高风险（相似率 ≥60%） | 40 个文件 |
| 中风险（40%–59.9%） | 35 个文件 |
| 观察项（20%–39.9%） | 35 个文件 |
| 归一化后完全一致的 Go 函数 | 150 个，约 1,769 行 |

## 使用约定

- 重写顺序建议：`log`、`utils/terminal`、`utils/files`、数据库客户端、容器服务、应用商店与 DTO/Repo。
- 每完成一个文件，补充独立规格、回归测试和设计记录，再从清单移入“已处置记录”。
- 不以改变量名、调换语句或机械拆函数作为完成标准；目标是基于功能规格形成独立实现。
- 每次升级 1Panel 或大规模重构 GoPanel 后重新运行相似度审计，结果不能直接沿用。

## 高风险文件（40）

| GoPanel 文件 | 相似率 | 匹配有效行 | 最接近的 1Panel 文件 |
|---|---:|---:|---|
| `app/dto/ftp.go` | 100.0% | 35 | `agent/app/dto/ftp.go` |
| `app/dto/device.go` | 100.0% | 33 | `agent/app/dto/device.go` |
| `app/dto/group.go` | 100.0% | 21 | `agent/app/dto/group.go` |
| `log/writer.go` | 99.1% | 223 | `agent/log/writer.go` |
| `utils/postgresql/client.go` | 94.2% | 49 | `agent/utils/postgresql/client.go` |
| `app/dto/command.go` | 87.9% | 29 | `agent/app/dto/command.go` |
| `log/manager.go` | 86.7% | 39 | `agent/log/manager.go` |
| `app/dto/ai.go` | 86.5% | 32 | `agent/app/dto/ai.go` |
| `utils/terminal/ws_session.go` | 81.9% | 158 | `core/utils/terminal/ws_session.go` |
| `app/service/container_stats_utils.go` | 81.0% | 68 | `agent/app/service/container.go` |
| `utils/files/fileinfo.go` | 80.7% | 330 | `agent/utils/files/fileinfo.go` |
| `app/dto/app.go` | 78.1% | 121 | `agent/app/dto/app.go` |
| `utils/terminal/ws_local_session.go` | 78.0% | 92 | `core/utils/terminal/ws_local_session.go` |
| `app/dto/host.go` | 76.6% | 49 | `agent/app/dto/host.go` |
| `utils/encrypt/encrypt.go` | 76.3% | 129 | `core/utils/encrypt/encrypt.go` |
| `app/dto/dashboard.go` | 76.2% | 80 | `agent/app/dto/dashboard.go` |
| `app/dto/database.go` | 75.9% | 214 | `agent/app/dto/database.go` |
| `utils/files/tar.go` | 75.0% | 24 | `agent/utils/files/tar.go` |
| `utils/ntp/ntp.go` | 73.2% | 52 | `agent/utils/ntp/ntp.go` |
| `utils/postgresql/client/info.go` | 72.7% | 48 | `agent/utils/postgresql/client/info.go` |
| `app/dto/request/file.go` | 71.8% | 89 | `agent/app/dto/request/file.go` |
| `app/model/app_install.go` | 71.7% | 33 | `agent/app/model/app_install.go` |
| `log/config.go` | 71.4% | 25 | `agent/log/config.go` |
| `utils/storage/client/webdav.go` | 71.3% | 77 | `agent/utils/cloud_storage/client/webdav.go` |
| `app/service/container_docker_config.go` | 71.2% | 215 | `agent/app/service/docker.go` |
| `app/dto/fail2ban.go` | 70.8% | 17 | `agent/app/dto/fail2ban.go` |
| `utils/storage/client/sftp.go` | 70.5% | 129 | `agent/utils/cloud_storage/client/sftp.go` |
| `utils/mysql/client/info.go` | 69.5% | 82 | `agent/utils/mysql/client/info.go` |
| `app/service/container_network.go` | 69.5% | 116 | `agent/app/service/container_network.go` |
| `app/repo/compose_template.go` | 69.4% | 84 | `agent/app/repo/compose_template.go` |
| `app/dto/request/app.go` | 69.4% | 68 | `agent/app/dto/request/app.go` |
| `utils/postgresql/client/local.go` | 69.4% | 154 | `agent/utils/postgresql/client/local.go` |
| `app/repo/app_install.go` | 66.2% | 131 | `agent/app/repo/app_install.go` |
| `utils/geo/geo.go` | 65.7% | 23 | `agent/utils/geo/geo.go` |
| `app/dto/response/app.go` | 65.3% | 94 | `agent/app/dto/response/app.go` |
| `utils/mysql/client/remote.go` | 64.5% | 196 | `agent/utils/mysql/client/remote.go` |
| `utils/mysql/client.go` | 63.8% | 44 | `agent/utils/mysql/client.go` |
| `app/dto/database_postgresql.go` | 63.5% | 47 | `agent/app/dto/database_postgresql.go` |
| `constant/website.go` | 62.5% | 25 | `agent/constant/website.go` |
| `app/repo/image_repo.go` | 60.9% | 56 | `agent/app/repo/image_repo.go` |

## 中风险文件（35）

| GoPanel 文件 | 相似率 | 匹配有效行 | 最接近的 1Panel 文件 |
|---|---:|---:|---|
| `app/model/ssl.go` | 59.5% | 25 | `agent/app/model/website_ssl.go` |
| `app/repo/backup.go` | 58.7% | 27 | `agent/app/repo/backup.go` |
| `app/model/app.go` | 58.4% | 45 | `agent/app/model/app.go` |
| `app/dto/image_repo.go` | 56.8% | 21 | `agent/app/dto/image_repo.go` |
| `app/service/setting_cert.go` | 54.8% | 63 | `core/app/service/setting.go` |
| `utils/ssh/ssh.go` | 54.5% | 42 | `agent/utils/ssh/ssh.go` |
| `app/dto/docker.go` | 54.2% | 26 | `agent/app/dto/docker.go` |
| `app/service/monitor.go` | 54.1% | 105 | `agent/app/service/monitor.go` |
| `app/dto/clam.go` | 54.1% | 46 | `agent/app/dto/clam.go` |
| `utils/mysql/client/local.go` | 54.1% | 191 | `agent/utils/mysql/client/local.go` |
| `utils/common/common.go` | 53.0% | 223 | `agent/utils/common/common.go` |
| `utils/files/file_op.go` | 52.3% | 161 | `agent/utils/files/file_op.go` |
| `app/dto/response/file.go` | 51.0% | 26 | `agent/app/dto/response/file.go` |
| `app/service/apps_install_files.go` | 50.7% | 76 | `agent/app/service/app_utils.go` |
| `app/model/monitor.go` | 50.0% | 12 | `agent/app/model/monitor.go` |
| `utils/postgresql/client/remote.go` | 48.9% | 158 | `agent/utils/postgresql/client/remote.go` |
| `utils/terminal/local_cmd.go` | 48.3% | 42 | `core/utils/terminal/local_cmd.go` |
| `utils/websocket/client.go` | 47.5% | 19 | `agent/utils/websocket/client.go` |
| `buserr/errors.go` | 47.4% | 55 | `agent/buserr/errors.go` |
| `utils/toolbox/fail2ban.go` | 46.6% | 76 | `agent/utils/toolbox/fail2ban.go` |
| `app/dto/container.go` | 45.3% | 102 | `agent/app/dto/container.go` |
| `app/dto/firewall.go` | 45.2% | 28 | `agent/app/dto/firewall.go` |
| `app/repo/logs.go` | 44.9% | 40 | `core/app/repo/logs.go` |
| `app/service/file_tree.go` | 44.8% | 30 | `agent/app/service/file.go` |
| `utils/files/utils.go` | 44.2% | 73 | `agent/utils/files/utils.go` |
| `app/dto/response/process.go` | 44.2% | 19 | `agent/utils/websocket/process_data.go` |
| `app/service/dashboard.go` | 43.8% | 126 | `agent/app/service/dashboard.go` |
| `app/dto/setting.go` | 42.3% | 102 | `core/app/dto/setting.go` |
| `utils/http/request.go` | 41.9% | 18 | `core/utils/req_helper/requset.go` |
| `app/dto/common_req.go` | 41.7% | 20 | `agent/app/dto/common_req.go` |
| `utils/firewall/client/firewalld.go` | 41.4% | 101 | `agent/utils/firewall/client/firewalld.go` |
| `utils/firewall/client/ufw.go` | 41.0% | 133 | `agent/utils/firewall/client/ufw.go` |
| `app/service/setting.go` | 40.9% | 79 | `core/app/service/setting.go` |
| `utils/env/env.go` | 40.8% | 20 | `agent/utils/env/env.go` |
| `utils/cryptx/aes.go` | 40.8% | 64 | `core/utils/encrypt/encrypt.go` |

## 观察项（35）

| GoPanel 文件 | 相似率 | 匹配有效行 | 最接近的 1Panel 文件 |
|---|---:|---:|---|
| `app/service/container_compose.go` | 36.0% | 58 | `agent/app/service/container_compose.go` |
| `app/repo/website_domain.go` | 35.3% | 18 | `agent/app/repo/website_domain.go` |
| `constant/common.go` | 35.0% | 41 | `core/constant/common.go` |
| `utils/files/zip.go` | 34.3% | 24 | `agent/utils/files/rar.go` |
| `app/service/image_repo.go` | 34.2% | 96 | `agent/app/service/image_repo.go` |
| `app/service/container_inspect_utils.go` | 32.9% | 128 | `agent/app/service/container.go` |
| `utils/websocket/process_data.go` | 32.1% | 108 | `agent/utils/websocket/process_data.go` |
| `utils/files/file_op_archive.go` | 32.0% | 86 | `agent/utils/files/file_op.go` |
| `app/middleware/operation.go` | 31.4% | 43 | `core/middleware/operation.go` |
| `app/service/container_volume.go` | 31.4% | 85 | `agent/app/service/container_volume.go` |
| `utils/docker/docker.go` | 30.8% | 69 | `agent/utils/docker/docker.go` |
| `app/service/firewall_utils.go` | 29.7% | 19 | `agent/app/service/firewall.go` |
| `app/service/container_config_utils.go` | 29.6% | 66 | `agent/app/service/container.go` |
| `app/service/image_auth.go` | 29.2% | 26 | `agent/app/service/image.go` |
| `admin/src/views/Dashboard/Index.ts` | 28.7% | 25 | `frontend/src/views/home/index.vue` |
| `utils/docker/compose.go` | 28.6% | 26 | `agent/utils/docker/compose.go` |
| `utils/xpack/xpack.go` | 28.6% | 12 | `agent/utils/req_helper/request.go` |
| `app/service/app_install_validate_utils.go` | 28.6% | 8 | `agent/app/service/app_utils.go` |
| `app/service/file_content.go` | 28.3% | 17 | `agent/app/service/file.go` |
| `utils/files/archiver.go` | 26.5% | 9 | `agent/utils/files/archiver.go` |
| `admin/src/components/LayoutContent.vue` | 26.4% | 32 | `frontend/src/components/layout-content/index.vue` |
| `utils/files/file_op_download.go` | 26.1% | 55 | `agent/utils/files/file_op.go` |
| `utils/captcha/captcha.go` | 25.6% | 10 | `core/utils/captcha/captcha.go` |
| `app/service/app_utils.go` | 25.0% | 16 | `agent/app/service/app_utils.go` |
| `app/repo/app.go` | 24.4% | 32 | `agent/app/repo/app.go` |
| `app/dto/ssh.go` | 24.0% | 12 | `agent/app/dto/ssh.go` |
| `init/log/log.go` | 23.1% | 9 | `agent/init/log/log.go` |
| `app/repo/common.go` | 22.9% | 32 | `agent/app/repo/common.go` |
| `app/service/container_runtime_info.go` | 22.8% | 44 | `agent/app/service/container.go` |
| `app/dto/request/runtime.go` | 21.3% | 10 | `agent/app/dto/request/runtime.go` |
| `app/repo/website.go` | 21.3% | 27 | `agent/app/repo/website.go` |
| `app/service/apps_install.go` | 21.2% | 39 | `agent/app/service/app.go` |
| `utils/firewall/client/info.go` | 21.1% | 8 | `agent/utils/firewall/client/info.go` |
| `app/service/app_install_runtime_utils.go` | 20.7% | 75 | `agent/app/service/app_utils.go` |
| `app/dto/logs.go` | 20.5% | 15 | `core/app/dto/logs.go` |

## 完全相同的小文件补充清单

以下文件与当前 1Panel 文件逐字相同，其中部分已出现在主清单，其余因有效代码少于 20 行而单列：

| GoPanel 文件 | 行数 | 相同的 1Panel 文件 |
|---|---:|---|
| `app/dto/common_res.go` | 16 | `agent/app/dto/common_res.go` |
| `app/dto/ftp.go` | 43 | `agent/app/dto/ftp.go` |
| `app/dto/group.go` | 25 | `core/app/dto/group.go` |
| `app/model/app_detail.go` | 15 | `agent/app/model/app_detail.go` |
| `app/model/app_ignore_upgrade.go` | 8 | `agent/app/model/app_ignore_upgrade.go` |
| `app/model/app_install_resource.go` | 10 | `agent/app/model/app_install_resource.go` |
| `app/model/app_launcher.go` | 17 | `agent/app/model/app_launcher.go` |
| `app/model/app_tag.go` | 7 | `agent/app/model/app_tag.go` |
| `app/model/base.go` | 9 | `agent/app/model/base.go`、`core/app/model/base.go` |
| `buserr/multi_err.go` | 23 | `agent/buserr/multi_err.go`、`core/buserr/multi_err.go` |

## 已处置记录

当前为空。后续每项至少记录：GoPanel 文件、处置方式、独立设计文档、回归测试、完成提交、复审结果。
