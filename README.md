# my-blog
一个简洁风格的博客项目，借助GPT开发
v0.1 完成整个项目的第一版， 非常简洁的前端页面加上轻量级的后端

## 介绍
1. 前端使用vue，支持通过wails打包为桌面应用
2. 后端使用go，支持1c 256m 部署
3. 后端功能：定时从自己的github中拉取文档，避免了自己备份和搭建数据库的压力，把github仓库作为自己的文档数据源
4. 前端功能：支持明亮和暗黑两种模式切换，支持搜索

## 部署
### 前端
#### Vue
1. 配置环境变量(服务端的地址，默认是""，在本机访问无问题，但是手机访问就会请求不通)
> export VITE_API_BASE_URL="" // 为空，或者是你的域名
2. web 打包（dist目录下即为需要的静态文件）
> cd blog-github-desktop/frontend
> npm install
> npm run build
#### Wails
1. **配置环境变量**(静态页面可选是因为一般都是在本机默认即可，但是wails是在本地的app，所以一定要配置环境变量，或者在项目中修改为自己的服务地址)
> export VITE_API_BASE_URL="https://myserver:80" //最好是https，否则会被macOS ATS 拦截
3. wails 打包
> cd blog-github-desktop
> wails build


### 后端
1. go 打包
> cd blog-github-api
> GOOS=linux GOARCH=amd64 go build -o blog-server .
blog-server 即为需要的服务文件
2. 测试运行
> chmod +x blog-server
> ./blog-server
3. 正常启动会输出日志
```
listening on :8080
initial sync done, posts: xx
```
### 环境变量
```
export GITHUB_OWNER="你的用户名"
export GITHUB_TOKEN="你的github仓库token"
export GITHUB_BRANCH="main"
export GITHUB_REPO="你的仓库名称"
export POSTS_DIR="读取的顶级目录，根目录就留空"
export ALLOWED_ORIGINS="你的域名示例：https://localhost:3000"
```
### 发布项目
1. 通过scp命令，将dist.zip 和 blog-server 文件上传到服务器
> scp -P port username@ip:/path
2. 把dist放到nginx目录下
3. 把blog-server放到你自己的目录下
> chmod +x blog-server
4. 创建快捷方式
> vim /etc/systemd/system/blog.service
```
# 加入以下内容
[Unit]
Description=My Blog Server
After=network.target

[Service]
Type=simple
ExecStart=/root/blog-server
WorkingDirectory=/root
Restart=always
RestartSec=5

# 你的环境变量（记得改）
Environment=GITHUB_TOKEN=你的token
Environment=GITHUB_OWNER=xxxx
Environment=GITHUB_REPO=my-blog-content
Environment=REFRESH_EVERY=2h
Environment=PAGE_SIZE=10
Environment=BIND_ADDR=:8080

[Install]
WantedBy=multi-user.target
```
5. 启动
> systemctl daemon-reload //刷新配置
> systemctl enable blog //启用
> systemctl start blog //启动
> systemctl status blog //查看状态
```
● blog.service - My Blog Server
     Loaded: loaded (/etc/systemd/system/blog.service; enabled; vendor preset: enabled)
     Active: active (running) since Thu 2025-12-04 12:50:31 GMT; 24s ago
   Main PID: 20057 (blog-server)
      Tasks: 4 (limit: 482)
     Memory: 9.4M
        CPU: 31ms
     CGroup: /system.slice/blog.service
             └─20057 /blog/blog-server

Dec 04 12:50:31 wowati.alice.ws systemd[1]: Started My Blog Server.
Dec 04 12:50:32 wowati.alice.ws blog-server[20057]: 2025/12/04 12:50:32 initial sync done, posts: 6
Dec 04 12:50:32 wowati.alice.ws blog-server[20057]: 2025/12/04 12:50:32 listening on :8080 (HTTP, no TLS)
```

### github 仓库token申请
1. 打开token创建页面 `https://github.com/settings/tokens`
2. 访问目录
> Developer settings → Personal access tokens → Tokens (classic)
3. 点击 「Generate new token (classic)」
> 名字随意
4. 勾选你需要的最小权限
> 建议只勾选 repo
5. 生成 Token
> ghp_xxxxxxxxxxxxxxxxxxxxx
