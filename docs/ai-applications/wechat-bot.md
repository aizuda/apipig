# 微信 Bot 最小通信接入层

## 目标

本模块在“AI 应用”下提供微信 Bot 账户管理和最小消息通信能力，底层协议完全由 `github.com/openilink/openilink-sdk-go` 承担。模块不复制协议签名、长轮询、消息结构或二维码登录算法。

当前闭环包括：

- 扫码绑定和重新绑定微信 Bot；
- 多账户连接、停用、重连、状态监控和进程重启恢复；
- 文本消息长轮询接收及本地持久化；
- 按联系人保存最新 `context_token`，在 24 小时窗口内发送文本回复；
- 账户、联系人和消息管理页面。

## 对 openilink-hub 的分析与取舍

`openilink-hub` 的微信账户链路主要由四层组成：

| Hub 能力 | 本模块设计 | 取舍原因 |
| --- | --- | --- |
| `provider/ilink` 协议适配 | 直接依赖 `openilink-sdk-go` | 避免在 ApiPig 内重复协议实现 |
| `bot.Manager` 多实例生命周期 | `service.Runtime` 管理每个 Bot 的 SDK Client | 保留进程内多账户连接和恢复能力 |
| Bot credentials / sync state | 独立 Bot 表保存加密 Token、Base URL 和游标 | 长轮询重启后可继续消费，不暴露凭据 |
| Message / context token store | 联系人表保存最新加密上下文，消息表保存展示字段 | 满足可回复和管理控制台的最小数据面 |
| App、Webhook、WebSocket、AI 多路分发 | 暂不引入 | 属于 Hub 的应用平台层，不是通信接入层 |
| CDN 媒体下载、上传和 SILK 转码 | 暂不引入 | 首版仅承诺文本通信，降低文件存储与安全面 |
| 24 小时提醒、链路追踪 | 暂不引入 | 可在稳定的消息事件边界上继续扩展 |

## 模块结构

```text
微信扫码 -> Bind Session -> 加密保存 Bot Token
                              |
进程启动/重连 -> Runtime -> SDK Monitor -> 入站消息 -> Message
                                         -> 最新上下文 -> Contact

管理端发送 -> 校验 Bot 在线和 24h 窗口 -> 解密 context_token
          -> SDK SendText -> Outbound Message
```

- `model.Bot`：账户身份、连接状态、加密凭据、同步游标和聚合计数；
- `model.Contact`：每个外部联系人的最新上下文令牌和活跃时间；
- `model.Message`：控制台需要的收发方向、文本、内容类型和会话元数据；
- `service.Runtime`：SDK Client 生命周期、长轮询、游标更新和消息收发；
- `service.BotService`：扫码会话及账户、联系人、消息管理用例；
- `api/router`：复用现有 JWT 和 RBAC 的管理接口。

## 关键协议约束

1. `context_token` 来自入站消息，并且与联系人相关。回复时不能使用其他联系人的令牌。
2. SDK 的“主动推送”仍依赖联系人此前上行消息所产生的 `context_token`；扫码确认本身不提供该上下文，因此接入成功只在管理端提示，不向微信端主动发消息。
3. 当前会话窗口按联系人最后一条入站消息计算，超过 24 小时后管理端拒绝发送。
4. 扫码确认可能返回新的消息服务 `base_url`，必须和 Bot Token 一起持久化。
5. `get_updates_buf` 每次变化都持久化，服务重启时作为 SDK `InitialBuf` 恢复。
6. SDK 可能重复投递消息；模块以 Bot、方向和外部消息 ID 做幂等约束。
7. 二维码状态可能要求切换轮询主机，扫码会话会保存并使用重定向地址。

## 安全策略

- Bot Token 和联系人 `context_token` 使用项目现有 AES-256-GCM `CredentialVault` 加密；
- API 和 JSON 响应永不返回 Bot Token、同步游标或上下文令牌；
- 删除账户时硬删除该账户的本地联系人和消息记录；
- 所有管理接口挂载在现有 JWT/RBAC 中间件之后；
- 发送内容限制为 4000 字节，扫码会话在内存中保存并自动过期。

凭据加密复用 `ai.encryption-key` 配置，也可由 `APIPIG_AI_ENCRYPTION_KEY` 覆盖。缺少密钥时扫码确认不会保存明文凭据。

## 管理接口

统一前缀：`/v1/ai-applications/wechat-bot/bot/`

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `page` | 账户分页和状态筛选 |
| POST | `bind/start` | 获取扫码会话和二维码 |
| GET | `bind/status/:sessionId` | 长轮询扫码状态并在确认后保存账户 |
| POST | `rename` | 修改显示名称 |
| POST | `reconnect` | 重新建立 SDK 长轮询连接 |
| POST | `enabled` | 停用或启用账户 |
| POST | `delete` | 删除账户及本地通信数据 |
| GET | `contacts?botId=...` | 查询最近联系人及回复窗口 |
| POST | `messages` | 查询联系人消息记录 |
| POST | `send` | 在有效上下文窗口内发送文本消息 |

## 后续扩展边界

媒体消息应继续调用 SDK 的 `DownloadMedia` 和 `SendMediaFile`，但需要先增加受控对象存储、大小限制、MIME 校验和清理策略。AI 自动回复、Webhook 或 WebSocket 分发应订阅持久化后的入站消息事件，不应进入 SDK Runtime 或改变游标提交逻辑。
