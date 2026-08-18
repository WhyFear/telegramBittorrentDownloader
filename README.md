# Telegram Bittorrent Downloader

## HTTP 下载接口

项目可以在保留 Telegram Bot 的同时提供 HTTP 下载接口。默认配置中的 `api.port` 为
`0`，因此接口默认关闭。启用示例：

```yaml
api:
  listen_ip: "127.0.0.1"
  port: 8081
  token: "replace-with-your-token"
```

请求地址为 `POST /api/v1/downloads`，请求体接受完整 Magnet 链接或 40 位 Hash，
`channel` 当前使用 `qbittorrent`：

```powershell
curl.exe -X POST "http://127.0.0.1:8081/api/v1/downloads" `
  -H "Authorization: Bearer replace-with-your-token" `
  -H "Content-Type: application/json" `
  -d '{"magnet":"magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567","channel":"qbittorrent"}'
```

Token 为空时接口不校验 `Authorization` Header。若在 Docker 中通过端口映射访问，
需要将 `listen_ip` 设置为 `0.0.0.0`，并使用 `-p 8081:8081` 暴露端口。

一个基于 Telegram Bot 的磁力链接搜索与下载工具。支持从 Nyaa 搜索资源，并一键推送到 qBittorrent 进行下载。

## 功能特性

- **磁力搜索**：通过 `/nyaa <关键词>` 在 Nyaa.si 上搜索种子。
- **分页展示**：搜索结果支持分页浏览。
- **一键下载**：点击搜索结果下方的按钮，直接将任务推送到 qBittorrent。
- **磁力解析**：支持通过 `/qb <磁力链接或Hash>` 手动添加下载任务。
- **深层链接**：支持通过 `t.me/your_bot?start=<hash>` 快速添加下载。
- **代理支持**：支持为 Bot 和搜索请求配置 HTTP 代理。
- **多平台支持**：基于 Go 开发，易于部署。

## 快速开始

### 1. 准备工作

- 获取一个 Telegram Bot Token（通过 [@BotFather](https://t.me/BotFather)）。
- 准备一个可访问的 qBittorrent 实例，并开启 Web UI 服务。

### 2. 配置文件

在项目根目录下创建 `config.yaml` 文件，内容格式如下：

```yaml
Proxy:
  URL: "http://127.0.0.1:7890" # 如果不需要代理请留空
bot:
  bot_name: "your_bot_name" # 你的机器人用户名（不含 @）
  token: "your_bot_token"   # 你的机器人 Token
searcher:
  - name: "nyaa"
    enable: true
downloader:
  - name: "qbittorrent"
    enable: true
    username: "admin"        # qBittorrent 用户名
    password: "password"     # qBittorrent 密码
    api_url: "http://localhost:8080" # qBittorrent Web UI 地址
    extra:
      category: "anime"      # 可选：下载分类
      save_path: "/downloads" # 可选：保存路径
```

### 3. 运行项目

#### 使用 Go 运行
```bash
go run main.go init.go
```

#### 使用 Docker (推荐)

1. **构建镜像**：
```bash
docker build -t tg-bt-downloader .
```

2. **运行容器**：
```bash
docker run -d \
  --name tg-bt-downloader \
  -v $(pwd)/config.yaml:/app/config.yaml \
  tg-bt-downloader
```

## 使用方法

- `/start` - 显示欢迎信息。
- `/nyaa <关键词>` - 搜索 Nyaa 资源。
- `/qb <磁力链接/Hash>` - 手动添加下载任务到 qBittorrent。

## 开发相关

### 项目结构
- `bot/` - Telegram Bot 逻辑处理。
- `config/` - 配置文件解析。
- `serivce/` - 核心服务层，包含搜索器 (searcher) 和下载器 (downloader)。
- `types/` - 公用结构体定义。
- `utils/` - 工具函数（日志、HTTP 客户端等）。

### 依赖项
- [telebot](https://github.com/tucnak/telebot) - Telegram Bot 框架。
- [go-qbittorrent](https://github.com/superturkey650/go-qbittorrent) - qBittorrent API 客户端。
- [yaml.v3](https://github.com/go-yaml/yaml) - YAML 解析。

## 许可证

[MIT License](LICENSE)
