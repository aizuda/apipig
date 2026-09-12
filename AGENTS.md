# PROJECT KNOWLEDGE BASE

**Generated:** 2026-09-11

## OVERVIEW

项目：**apipig**（运行输出中自称「ApiPig - 企业级AI网关」）
仓库：`/code/project/apipig`（Go module `apipig`，主分支 `master`）

一个 Go 语言单体服务 + Vue 3 管理后台。核心能力：
- **AI 协议网关**：统一 OpenAI / Anthropic 协议接入（`/v1/chat/completions`、`/v1/embeddings`、`/v1/images/generations`、`/v1/rerank`、`/v1/audio/*`、`/v1/messages`、`/v1/models`），通过供应商/渠道/访问 Token/代理/限流/熔断/计费/调用日志实现多模型统一路由（上游协议层使用 `github.com/zendev-sh/goai v0.9.8`）。
- **AI 应用**：Git WebHook 代码评审（GitHub/GitLab/Gitee）、远程 Agent（`cmd/remote-agent` 独立二进制）、微信公众号机器人。
- **系统管理**：用户/角色/资源/菜单（JWT + RBAC）。

技术栈：
- 后端：Go 1.26.0 · Fiber v2（fasthttp）· GORM（MySQL/PostgreSQL/SQLite，默认 SQLite）· Viper（配置）· Zap（日志）· swag（Swagger 文档）· robfig/cron（定时任务）· GoReleaser（发布）
- 前端：Vue 3.5 + TypeScript + Vite Plus（Rolldown）· Tailwind CSS v4 · shadcn-vue/Reka UI · Pinia · vue-router · vue-i18n · pnpm monorepo（workspace 包前缀 `@tabtab/*`）

## STRUCTURE

```
apipig/
├── main.go                 # 入口：配置解析 -> 首次初始化 -> Viper/Zap -> GORM -> RunServer
├── config/                 # 配置结构体（yaml/json/mapstructure tag，对应 config.yaml）
├── global/                 # 全局变量（DB/LOG/CONFIG）与常量（ConfigEnv="CONFIG"、ConfigFile="config.yaml"）
├── core/                   # 基础设施：viper.go/zap.go/server.go/schedule.go/db/cache/mail/api(通用响应与工具)
├── initialize/             # 组装：setup.go(首次初始化)、router.go(路由注册)、gorm.go/init_db.go/migrate.go
├── middleware/             # jwt/rbac/cors/recover/logger/signature 等中间件
├── app/                    # 业务模块（每个模块含 api/model/router/service 四层 + request/response）
│   ├── ai/                 #   AI 网关（provider/channel/channel-account/access-token/proxy/call-log/gateway）
│   ├── sys/                #   系统（user/role/resource/resource-api/web）
│   └── apps/               #   应用：code-review / remote-agent / wechat-bot
├── toolkit/                # 通用工具（aes/md5/crc16/json/validator/snowflake 等）
├── cmd/remote-agent/       # 远程 Agent 独立可执行文件（含 remote-agent.example.yaml）
├── docs/                   # swagger 生成文件 + 架构/应用文档（ai-gateway-architecture.md 等）
├── web/                    # 前端构建产物（go:embed 打包）与首次初始化页面 init.html
├── frontend/               # Vue 3 管理后台（pnpm monorepo，源码在 frontend/src）
├── boot.sh                 # Linux 部署 start/stop/restart 脚本（目录 /opt/apipig）
└── .goreleaser.yaml        # 同时产出 apipig 与 remote-agent 两个二进制
```

分层约定（`app/<模块>/`）：
- `api/`：HTTP 处理器，只做参数解析与协议入口，Swagger 注解写在这里
- `service/`：业务逻辑；`enter.go` 是组合根（依赖注入的默认绑定点）
- `model/`：GORM 模型 + `request/`（入参 DTO）+ `response/`（出参 DTO）
- `router/`：路由注册

## COMMANDS

| 动作 | 命令 |
|------|------|
| 后端依赖 | `go mod tidy` / `go mod download`（`main.go` 的 `//go:generate` 已内置 `GOPROXY=https://goproxy.cn,direct`） |
| 后端测试 | `go test ./...` |
| 后端构建 | `go build`（发布用 `goreleaser release --clean`；无窗口 `go build -ldflags "-s -w -H=windowsgui"`） |
| 后端运行 | `./apipig`（首次运行无 config.yaml 时在 `9527` 端口启动初始化页并自动打开浏览器） |
| 前端依赖 | `pnpm install` + `pnpm approve-builds`（需 Node ≥20、pnpm ≥10.32） |
| 前端开发 | `pnpm dev`（`frontend/` 目录，dev server 端口 3001） |
| 前端构建 | `pnpm build`（产物输出到仓库根 `web/dist`，由 Go `go:embed` 打包进二进制） |
| 前端检查 | `pnpm check`（类型）/ `pnpm lint` / `pnpm fmt` / `pnpm test` |

## CODING STANDARDS

- **语言**：Go（后端）+ TypeScript/Vue 3 `<script setup lang="ts">`（前端）。注释、文档、提交说明均为**中文**。
- **缩进**：Go 4 空格，YAML 2 空格，UTF-8、LF、行尾去空格、文件末尾换行（见 `.editorconfig`）。
- **后端分层**：api → service → model（service 不直接持有 `global.DB`，AI 模块经 `ai_store`/`gateway_repository` 抽象访问）。处理器统一用 `response.Execute(c, serviceFunc, params, err)` 返回，入参校验走 `core/api` 的 `BodyParser`/`BodyParserVerify`。
- **命名**：Go 用导出驼峰；目录/文件名用蛇形 `snake_case`（如 `access_token_api.go`）；每个模块用 `enter.go` 做组合根。前端别名 `@` → `frontend/src`，样式 Tailwind v4，格式化 `semi:false, singleQuote:true`。
- **错误处理**：service 返回 `(result, error)`，api 层用 `response.Execute` 统一包装；AI 模块错误信息会清洗 Token/Key 等敏感内容。

## WHERE TO LOOK

- **后端源码**：`app/`、`core/`、`initialize/`、`middleware/`、`config/`
- **AI 网关架构说明**：`docs/ai-gateway-architecture.md`（模块边界、安全边界、计费、路由策略）
- **应用文档**：`docs/ai-applications/`（code-review-webhook.md、wechat-bot.md、code-review-push-channels.md）、`docs/remote-agent.md`
- **前端源码**：`frontend/src/`（`views/`、`api/`、`router/modules/`、`stores/`）
- **Swagger 文档**：`docs/swagger.yaml` / 运行时 `GET /swagger/index.html`
- **测试**：与源码同目录的 `*_test.go`（`toolkit/`、`middleware/`、`docs/`、`version/`）

## NOTES

- **首次运行初始化**：`config.yaml` 不存在或为空时，进程在 `9527` 端口启动初始化服务并自动打开 `web/init.html`，引导选择 SQLite/MySQL/PostgreSQL、端口、管理员账号等，原子写入配置后自动切到正式服务。配置路径优先级：`-c` 参数 > `CONFIG` 环境变量 > `config.yaml`。
- **配置与密钥**：Viper 支持 `APIPIG_` 前缀环境变量覆盖（如 `APIPIG_AI_ENCRYPTION_KEY`）。生产环境必须设置 ≥32 字符的 `APIPIG_AI_ENCRYPTION_KEY` 主密钥（AES-256-GCM 加密渠道 API Key 与代理密码，不得复用 JWT 签名密钥）。访问 Token 只返回一次明文，库中仅存哈希。
- **前端产物联动**：`pnpm build` 输出到根目录 `web/dist` 并被 `go:embed` 打包。改了前端后必须重新 `go build`（或发布打包）并重启，仅重启旧二进制不会加载新前端。
- **两个二进制**：GoReleaser 从同一 tag/commit 产出 `apipig` 与 `apipig-remote-agent`；`version/version.go` 的 `Default` 常量是 snapshot 版本来源，正式发布需 tag 与 `Default` 一致（构建前有 `go test ./version -args -expected-version=<tag>` 校验）。
- **发布**：`goreleaser release --snapshot --clean` 打 snapshot；正式 `goreleaser release --clean`。可选 `upx -9` 压缩可执行文件。
- **环境提示**：当前开发容器**未安装 Go**（`go: command not found`），仅 Node v22 + pnpm 可用；后端构建/测试需在装有 Go 1.26 的环境执行。
- **其它上下文文件**：`.claude/settings.local.json`（仅放行部分 WebSearch/WebFetch 权限），无 `CLAUDE.md`/`.cursorrules`。
