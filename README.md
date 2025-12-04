# my-blog
一个简洁风格的博客项目，借助GPT开发
v0.1 完成整个项目的第一版， 非常简洁的前端页面加上轻量级的后端

## 介绍
1. 前端使用vue，支持通过wails打包为桌面应用
2. 后端使用go，支持1c 256m 部署
3. 后端功能：定时从自己的github中拉取文档，避免了自己备份和搭建数据库的压力，把github仓库作为自己的文档数据源
4. 前端功能：支持明亮和暗黑两种模式切换，支持搜索

## 部署
### Nginx
### 前端
### 后端

### 环境变量
```
export GITHUB_OWNER="你的用户名"
export GITHUB_TOKEN="你的github仓库token"
export GITHUB_BRANCH="master"
export GITHUB_REPO="你的仓库名称"
export POSTS_DIR="读取的顶级目录，根目录就留空"
export ALLOWED_ORIGINS="你的域名示例：http://localhost:3000"
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
