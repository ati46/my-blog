## 项目简介
本项目是一个轻量级的 VPS 资产监控系统，采用 Agent (客户端) + Server (服务端) + Dashboard (前端) 的架构。

功能： 实时监控 CPU、内存、硬盘、流量、网速，管理 VPS 续费周期与到期日。

特点： 单二进制文件部署、支持 IPv4/IPv6、支持 Nginx 反代 (WSS)、支持多节点鉴权、支持深色模式。
<img width="1360" height="660" alt="image" src="https://github.com/user-attachments/assets/3510a422-b05d-42d5-8e11-c97134c1691c" />

### 目录结构
推荐的项目文件结构：
```
/g-probe
  ├── agent/
  │    └── main.go       # 客户端源码
  ├── server/
  │    ├── main.go       # 服务端源码
  │    ├── index.html    # 前端面板 (Vue3)
  │    └── db.json       # 配置文件 (自动生成/手动修改)
  ├── go.mod             # Go 依赖文件
  └── go.sum
```
## 核心代码实现
### 代码介绍
1 初始化项目
在项目根目录执行：
```
go mod init g-probe
# 下载依赖
go get github.com/gorilla/websocket
go get github.com/shirou/gopsutil/v3
go mod tidy
```
2. 服务端代码 (server/main.go)
服务端负责：鉴权、接收数据、计算流量差值、防止死锁、提供 HTTP/WebSocket 接口。
```
// 完整代码请参考对话历史中 "server/main.go" 的最终优化版本
// 关键特性：
// 1. 使用 sync.RWMutex 读写锁，防止高并发死锁
// 2. 只有在 db.json 中配置了 Secret 的节点才允许连接
// 3. 60秒心跳检测，自动清理僵尸连接
// 4. 支持 -p 参数自定义端口
```
3 客户端代码 (agent/main.go)
客户端负责：采集系统指标（原始数据），支持命令行参数启动。
```
// 完整代码请参考对话历史中 "agent/main.go" 的最终优化版本
// 关键特性：
// 1. 支持 flag 命令行参数 (-s, -d, -p, -ssl)
// 2. 采集 CPU、内存、磁盘、网络原始计数
// 3. 断线自动重连机制
```
4 前端代码 (server/index.html)
前端负责：数据可视化、深色模式、自动推算到期日。

### 编译打包 (交叉编译)
为了在不同架构的 VPS 上运行，需要在开发机（Mac/Windows）上进行交叉编译。

1. 编译客户端 (Agent)
Linux AMD64 (常见 VPS):
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o agent-amd64 agent/main.go
```
Linux ARM64 (甲骨文 ARM/树莓派):
```
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o agent-arm64 agent/main.go
```
2. 编译服务端 (Server)
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o dashboard-server server/main.go
```
### 服务端部署指南
1 上传文件
将以下文件上传至服务器 /opt/monitor 目录：
```
dashboard-server (二进制)
index.html (前端)
db.json (配置文件模板)
scp -P port file user@ip:path
```
2 配置 db.json
**Key 必须与 host_name 一致！**
```
{
  "HK-Node-01": {
    "secret": "my_secret_key_123",  <-- 必须设置
    "host_name": "HK-Node-01",      <-- 必须与 Key 一致
    "cost": "$5.00",
    "billing_cycle": "MONTH",       <-- MONTH 或 YEAR
    "reset_day": 1,
    "start_date": "2023-10-10",
    "location": "Hong Kong",
    "traffic_limit_gb": 500
  }
}
```
3 Systemd 守护进程
创建文件 /etc/systemd/system/monitor-server.service：
```
[Unit]
Description=G-Probe Server
After=network.target

[Service]
Type=simple
# 必须指定工作目录，否则找不到 index.html
WorkingDirectory=/opt/monitor
# 指定端口启动
ExecStart=/opt/monitor/dashboard-server -p 8081
Restart=always

[Install]
WantedBy=multi-user.target
```
启动服务：
```
systemctl enable --now monitor-server
```
4 Nginx 反向代理配置 (推荐)
支持 HTTPS 和 WSS。
```
server {
    listen 80;
    server_name monitor.yourdomain.com; # 你的域名

    location / {
        proxy_pass http://127.0.0.1:8081; # 转发给 Go
        
        # WebSocket 关键配置
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 获取真实 IP
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
### 客户端部署指南
1 上传文件
将 agent-amd64 (或 arm64) 上传至 /opt/agent，并赋予权限：
```
chmod +x /opt/agent/agent-amd64
注意：如果是 ARM 机器，请上传 arm64 版本。
```
2 Systemd 启动配置
创建文件 /etc/systemd/system/monitor-agent.service：
```
[Unit]
Description=G-Probe Agent
After=network.target

[Service]
Type=simple
# 注意：密钥包含特殊字符时必须用单引号包裹
ExecStart=/opt/agent/agent-amd64 -s 'my_secret_key_123' -d "monitor.yourdomain.com" -p "443" -ssl
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```
参数说明：
-s: 对应 db.json 中的 secret。
-d: 服务端域名。
-p: 端口 (Nginx 反代填 80/443，直连填 8081)。
-ssl: 如果使用了 HTTPS/WSS，必须加上此标记。

3 启动
```
systemctl enable --now monitor-agent
```
## 常见问题排查 (FAQ)
Q1: 启动 Agent 报错 status=203/EXEC
原因 1: 文件没有执行权限。 -> 执行 `chmod +x agent-xxx`。

原因 2: 架构不对（在 AMD 机器跑了 ARM 包）。 -> uname -m 检查架构，重新上传。

Q2: Agent 连接报错 bad handshake 或 secret丢失
原因: 密钥中包含 $ 等特殊字符，Shell 自动转义导致变空。

解决: 启动命令中的密钥必须用单引号 '...' 包裹。

Q3: 网页显示 API Pending 或 502/504
原因: 服务端死锁或旧进程未清理。

解决: pkill -9 dashboard-server 杀掉所有进程，然后重启服务。

Q4: 网页显示空白或 API 返回 null
原因: db.json 加载失败或 Agent 密钥未在服务端注册。

解决: 检查服务端日志 `journalctl -u monitor-server -f`，确认 db.json 路径正确且加载成功。
