# Remote Agent Controller

## 工作方式

Remote Agent 使用一条统一记录保存接入配置、注册凭据、机器信息和在线状态。管理员必须先在 **Remote Agent > 添加 Agent** 中创建记录；客户端注册只会激活这条记录，不会自动创建新的 Agent。一个注册配置只能对应一台工作站。

Web Agent 控制台以会话组织交互。每条用户消息会派发一个对话轮次，Agent 在固定的会话目录中运行 Codex，并将输出分片实时回传。会话、用户消息、助手消息和执行状态都会持久化，可在控制台留档或恢复。

## Controller 配置

```yaml
remote-agent:
  heartbeat-timeout-seconds: 90
  offline-check-cron: "* * * * *"
  command-poll-timeout-seconds: 25
  dispatch-lease-seconds: 30
```

Controller 只保存注册 Token 和运行 Token 的 SHA-256 摘要。

## 构建与运行 Agent

1. 在工作站安装 Codex CLI。
2. 在 Remote Agent 页面添加 Agent，下载一次性展示的 `remote-agent.yaml`。
3. 构建 Agent：

   ```shell
   go build -o remote-agent ./cmd/remote-agent
   ```

4. 使用非 root 系统用户运行：

   ```shell
   ./remote-agent -c remote-agent.yaml
   ```

每个会话使用 `<workspace-root>/conversations/<conversation-id>` 作为持久工作目录。默认 Codex 调用等价于：

```shell
codex exec --skip-git-repo-check -
```

提示词通过 stdin 传入，不会出现在进程参数中。

Agent 进程在执行轮次期间意外退出时，下次注册会自动清理未完成的部分输出并重新派发该轮次。

重置注册 Token 会立即注销旧的运行 Token。重置后需要替换工作站上的完整配置文件并重启 Agent。

## API

Agent 接口：

- `POST /v1/remote-agent/register`
- `POST /v1/remote-agent/heartbeat`
- `GET /v1/remote-agent/command/next`
- `POST /v1/remote-agent/command/acknowledge`
- `POST /v1/remote-agent/message/chunks`
- `POST /v1/remote-agent/message/result`

管理接口：

- `POST /v1/ai-applications/remote-agent/agent/create`
- `POST /v1/ai-applications/remote-agent/agent/update`
- `POST /v1/ai-applications/remote-agent/agent/page`
- `GET /v1/ai-applications/remote-agent/agent/get?id=<agent-id>`
- `POST /v1/ai-applications/remote-agent/agent/rotate-token?id=<agent-id>`
- `POST /v1/ai-applications/remote-agent/agent/status`
- `POST /v1/ai-applications/remote-agent/agent/delete`
- `POST /v1/ai-applications/remote-agent/conversation/create`
- `POST /v1/ai-applications/remote-agent/conversation/page`
- `GET /v1/ai-applications/remote-agent/conversation/get?id=<conversation-id>`
- `POST /v1/ai-applications/remote-agent/conversation/archive`
- `POST /v1/ai-applications/remote-agent/conversation/message/send`
- `POST /v1/ai-applications/remote-agent/conversation/message/stream`
