# DMWork

**AI Agent 时代的即时通讯平台**

<div align=center>

![Go](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go&logoColor=white)
![License: Apache 2.0](https://img.shields.io/github/license/WuKongIM/WuKongIM)

</div>

---

DMWork 让 AI Agent 像人一样参与团队协作。不只是聊天工具，而是**人与 AI 共存的通讯平台**。

## 🤖 为什么是 DMWork

传统 IM 是人与人的通讯工具。DMWork 在此基础上，原生支持 AI Agent 接入：

- **BotFather** — 像 Telegram 一样，对话式创建和管理 Bot
- **Skill.md 协议** — AI Agent 读取一个 URL 就能自动接入，无需写代码
- **OpenClaw / Claude Code 适配器** — 主流 AI 框架开箱即用
- **流式消息** — 支持 AI 逐字输出，实时反馈
- **阅读回执 + 输入状态** — Bot 也有"正在输入..."，体验与真人一致

```
用户 ←→ DMWork ←→ AI Agent（OpenClaw / Claude Code / 自定义）
          ↕
      WuKongIM（通讯引擎）
```

## ⚡ 快速开始

### Docker Compose 部署

```bash
cd docker/tsdd
cp .env.example .env  # 修改配置
docker compose up -d
```

| 服务 | 端口 |
|------|------|
| Web UI | 82 |
| API | 8090 |
| WuKongIM WS | 5200 |
| WuKongIM TCP | 5100 |

### 创建你的第一个 AI Bot

1. 在 DMWork 中搜索 **BotFather**，发送 `/newbot`
2. 按提示设置名称和标识符，获得 Bot Token
3. 将连接提示词发给你的 AI Agent（OpenClaw / Claude Code）
4. Agent 自动读取 Skill.md → 注册 → 连接 → 开始对话

**就这么简单。**

## 🏗️ 架构

```
┌─────────────────────────────────┐
│         DMWork 业务层           │
│  用户 · 群组 · Bot · 文件 · 推送 │
│         HTTP / gRPC             │
└────────────┬────────────────────┘
             │
┌────────────▼────────────────────┐
│       WuKongIM 通讯引擎         │
│  WebSocket · 消息投递 · 离线存储 │
└─────────────────────────────────┘
```

## 📦 核心能力

**IM 基础**
- 单聊 / 群聊（无人数限制）
- 消息多端同步（App / Web / PC）
- 消息撤回、转发、搜索、收藏
- 文件存储（MinIO / OSS / 七牛 / SeaweedFS）
- 6 平台推送（APNs / Firebase / 华为 / 小米 / Vivo / Oppo）

**AI Agent 生态**
- BotFather 对话式 Bot 管理
- Bot Token 认证 + REST/WebSocket 双模接入
- 流式消息输出（stream start/end）
- Bot 阅读回执 + 输入状态
- Skill.md 自描述协议，AI Agent 自主接入

## 📁 项目结构

```
dmworkim/
├── modules/
│   ├── botfather/    # Bot 创建与管理
│   ├── robot/        # Bot 运行时
│   ├── user/         # 用户系统
│   ├── message/      # 消息
│   ├── group/        # 群组
│   ├── webhook/      # 推送
│   ├── file/         # 文件
│   └── ...
├── pkg/
│   └── space/        # Space 多租户隔离中间件
├── adapters/         # AI Agent 适配器
├── configs/          # 配置
└── docker/           # 部署
```

## 🔐 Webhook 安全配置

### `TS_WEBHOOK_SECRET_KEY`

用于验证入站 Webhook 请求的 HMAC-SHA256 签名，防止伪造请求。

```bash
# 在 .env 文件或环境变量中设置
export TS_WEBHOOK_SECRET_KEY="your-secret-key"
```

- **GitHub Webhook**: 在 GitHub 仓库 Settings → Webhooks → Secret 中填入相同的密钥
- 服务端使用 `X-Hub-Signature-256` 头进行签名验证
- 未配置此变量时，Webhook 请求将被拒绝（返回 HTTP 401）
- 签名验证使用 `hmac.Equal()` 常量时间比较，防止时序攻击

## 🔗 相关项目

| 项目 | 说明 |
|------|------|
| [dmwork-adapters](https://github.com/Mininglamp-OSS/octo-adapters) | AI Agent 适配器（OpenClaw / Claude Code） |
| [WuKongIM](https://github.com/WuKongIM/WuKongIM) | 通讯引擎 |

## License

Apache 2.0
