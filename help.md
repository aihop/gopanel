## 新建用户流程

# --system: 创建系统用户

# --shell /usr/sbin/nologin: 禁止该用户登录 shell

# --group: 同时创建一个同名的用户组

sudo adduser --system --group --no-create-home --shell /usr/sbin/nologin gopanel

# 确保程序所在目录及其文件的所有权

sudo chown -R gopanel:gopanel /opt/gopanel

# 修改 Systemd 服务文件

你需要告诉 Systemd 使用新创建的用户来启动进程。修改 /etc/systemd/system/gopanel.service（以及 gp-agent.service）：

```
[Unit]
Description=GoPanel Service
After=network.target podman.socket

[Service]
Type=simple
# --- 关键修改：指定用户和组 ---
User=gopanel
Group=gopanel
# -------------------------
WorkingDirectory=/opt/gopanel
ExecStart=/opt/gopanel/gopanel
Restart=always

[Install]
WantedBy=multi-user.target
```

## 监听 1024 以下端口（如 80/443）

Linux 默认不允许普通用户监听 1024 以下的端口。如果你的 GoPanel 需要监听 80 端口，请执行以下命令赋予二进制文件特殊能力：

```
sudo setcap cap_net_bind_service=+ep /opt/gopanel/gopanel
```

#   访问 Podman Socket
默认情况下，/var/run/podman.sock 的所有者是 root。要让 gopanel 用户能管理容器，需要将该用户加入 podman 组（如果存在）或者修改 Socket 权限：

```
sudo usermod -a podman gopanel
```
或
```
sudo chmod 660 /var/run/podman.sock
```
