# Git WebHook AI 代码评审

## 目标

本模块监听 GitHub、GitLab、Gitee 的 Push 与 Pull Request/Merge Request 事件，自动获取目标提交的最新代码，生成受控 Git Diff，调用现有 AI 网关完成代码评审，并在管理台保存结构化报告。

## 用户交互过程

1. 管理员先在 **AI 网关** 中配置供应商、渠道账号、模型和 API 密钥。
2. 进入 **AI 代码评审**，创建项目并填写 HTTPS 仓库地址、仓库访问 Token、AI API 密钥、模型和监听分支。
3. 保存后系统生成 WebHook URL 和 Secret；Secret 只在创建或轮换时显示一次。
4. 管理员在 Git 平台配置 WebHook，勾选 Push 与 Pull/Merge Request 事件。
5. Git 平台推送事件后，接口立即返回 `202 Accepted` 和任务 ID，不阻塞代码仓库操作。
6. 后台工作线程依次完成验签、事件去重、代码拉取、Diff 生成、AI 分析和报告持久化。
7. 管理员在“最近评审任务”中查看状态、风险级别、问题摘要和 Markdown 报告；失败任务可手动重试。

## 系统交互时序

```text
Git Provider
  -> POST /v1/ai-applications/code-review/webhook/{webhookKey}
  -> WebHook 验签与事件标准化
  -> 以 projectId + delivery/eventKey 去重
  -> 创建 queued 任务并返回 202
  -> Worker 将任务置为 running
  -> 临时隔离目录执行 git init/fetch/checkout
  -> 生成 base..head Diff、文件数和增删行统计
  -> 通过 GatewayService.GenerateInternal 调用现有 AI 渠道
  -> 复用模型授权、RPM/TPM、熔断、计费和调用日志
  -> 保存 succeeded/failed/ignored 状态及评审报告
```

## 项目管理过程

### 项目状态

- **启用**：接收并执行新 WebHook 任务。
- **禁用**：拒绝新事件；已进入队列的任务执行时会转为 `ignored`。
- **删除**：存在 `queued` 或 `running` 任务时禁止删除。

### 任务状态

- `queued`：已验签、已去重，等待工作线程。
- `running`：正在拉取代码或调用 AI。
- `succeeded`：报告生成并持久化完成。
- `failed`：Git、凭据、AI 或持久化步骤失败，可重试。
- `ignored`：事件类型未启用、分支不匹配、删除分支、无文本差异或项目被禁用。

服务启动时会把遗留的 `queued`、`running` 任务恢复为 `queued` 并重新入队。事件使用平台 delivery UUID；平台未提供 UUID 时使用事件类型、提交 SHA 和 PR/MR 编号构造幂等键。

## Git 平台配置

### GitHub

- Payload URL：管理台生成的 WebHook URL。
- Content type：`application/json`。
- Secret：管理台创建项目时生成的 Secret。
- Events：Pushes、Pull requests。
- 验签：`X-Hub-Signature-256` HMAC-SHA256。

### GitLab

- URL：管理台生成的 WebHook URL。
- Secret token：管理台生成的 Secret。
- Trigger：Push events、Merge request events。
- 验签：`X-Gitlab-Token` 常量时间比较。

### Gitee

- URL：管理台生成的 WebHook URL。
- WebHook 密码：管理台生成的 Secret。
- 事件：Push、Pull Request。
- 支持 `X-Hub-Signature-256`，并兼容 `X-Gitee-Token`。

## 管理 API

管理 API 使用现有 JWT/RBAC：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/v1/ai-applications/code-review/project/save` | 新增或更新项目 |
| GET | `/v1/ai-applications/code-review/project/get?id=...` | 查询项目 |
| POST | `/v1/ai-applications/code-review/project/page` | 项目分页 |
| POST | `/v1/ai-applications/code-review/project/delete` | 删除项目 |
| POST | `/v1/ai-applications/code-review/task/page` | 任务分页 |
| GET | `/v1/ai-applications/code-review/task/get?id=...` | 查询报告详情 |
| POST | `/v1/ai-applications/code-review/task/retry` | 重试失败或忽略任务 |

公共 WebHook 入口为 `POST /v1/ai-applications/code-review/webhook/{webhookKey}`，不使用后台 JWT，但必须通过项目 Secret 验签。

## 配置

```yaml
code-review:
  worker-count: 2
  queue-size: 128
  workspace-root: ""       # 空值使用系统临时目录
  git-timeout-sec: 120
  ai-timeout-sec: 180
  max-diff-bytes: 524288
  max-changed-files: 200
```

`max-diff-bytes` 超限时会截断 Diff，并在任务中记录 `diffTruncated=true`；`max-changed-files` 超限则任务失败，避免超大变更无边界消耗模型额度。

## 安全边界

- 仓库地址仅允许 HTTPS，禁止通过项目配置执行本地路径或 SSH 命令。
- WebHook Secret 和仓库 Token 使用现有 AES-256-GCM CredentialVault 加密，依赖 `APIPIG_AI_ENCRYPTION_KEY`。
- Git 使用独立临时目录，完成后自动删除；设置 `GIT_TERMINAL_PROMPT=0`，避免后台任务等待交互输入。
- Token 通过临时 HTTP Header 传给 Git，不写入仓库 remote URL。
- AI 只接收受大小限制的 Diff，不上传完整仓库。
- AI 内部调用复用访问 Token 的模型授权、额度、限流、渠道熔断、计费和调用日志。

## 后续迭代

1. 将报告回写为 GitHub PR Review、GitLab MR Note 或 Gitee PR 评论。
2. 按文件拆分并行评审，再由汇总模型合并报告。
3. 支持仓库级规则文件，例如 `.apipig-review.yml`。
4. 增加任务取消、指数退避重试、告警通知和队列监控。
5. 对高风险问题增加人工确认门禁或 CI 状态检查。
