# Remote Agent Controller

## 工作方式

Remote Agent 使用一条统一记录保存接入配置、注册凭据、机器信息和在线状态。管理员必须先在 **Remote Agent > 添加 Agent** 中创建记录；客户端注册只会激活这条记录，不会自动创建新的 Agent。一个注册配置只能对应一台工作站。

Web Agent 控制台以会话组织交互。每个会话绑定一个已有项目目录和 Codex/Claude CLI。每条用户消息会派发一个对话轮次，Agent 在项目目录中运行所选 CLI，并将输出分片实时回传。会话、用户消息、助手消息和执行状态都会持久化，可在控制台留档或恢复。

## Controller 配置

```yaml
remote-agent:
  heartbeat-timeout-seconds: 30
  offline-check-cron: "* * * * *"
  command-poll-timeout-seconds: 25
  dispatch-lease-seconds: 30
```

Controller 只保存注册 Token 和运行 Token 的 SHA-256 摘要。

## 构建与运行 Agent

1. 在工作站安装并登录 Codex CLI、Claude CLI 中需要使用的工具。
2. 在 Remote Agent 页面添加 Agent，下载一次性展示的 `remote-agent.yaml`。
3. 构建 Agent：

   ```shell
   go build -o remote-agent ./cmd/remote-agent
   ```

4. 使用非 root 系统用户运行：

   ```shell
   ./remote-agent -c remote-agent.yaml
   ```

`workspace-root` 是客户端允许远程开发的本地目录边界。目录不存在时，Agent Client 会在启动期间自动创建，包括 `./remote-agent-workspaces` 这类相对路径；如果路径已存在但不是目录，客户端会拒绝启动。例如项目位于 `D:\gowork\apipig` 和 `D:\gowork\other-project` 时，可配置：

```yaml
workspace-root: D:/gowork
```

新建会话时项目目录分别填写 `apipig` 或 `other-project`。项目目录必须是相对于 `workspace-root` 的路径；Agent 会拒绝绝对路径、越界路径、指向根目录外的符号链接、缺失目录和当前运行用户不可写的目录。项目目录留空或填写 `.` 时直接使用 `workspace-root`。

例如需要在 `D:\WebProjects` 克隆代码时，应将客户端配置为 `workspace-root: D:/WebProjects`，然后把会话项目目录填写为 `.`；如果仍配置为 `./remote-agent-workspaces`，`D:\WebProjects` 会被视为工作区外路径并被 Codex 沙箱拒绝。

默认 Codex 调用等价于：

```shell
codex --ask-for-approval never exec --json --sandbox workspace-write -c sandbox_workspace_write.network_access=true --skip-git-repo-check -
```

默认 Claude 调用等价于：

```shell
claude -p --permission-mode acceptEdits
```

提示词通过 stdin 传入，不会出现在进程参数中。Agent Client 会为 Codex `exec` 固定启用 `--ask-for-approval never`、`--sandbox workspace-write` 和 `sandbox_workspace_write.network_access=true`，允许在绑定项目目录中无人值守地写文件和访问网络；不会开放项目目录之外的写权限。客户端会移除已经废弃的 `--full-auto`，并覆盖 `--sandbox read-only`、`-c sandbox_mode=read-only` 或关闭网络的冲突配置。每次执行的 CLI、工作目录、沙箱模式和 Agent 版本会写入 Agent 日志。Claude 的 `acceptEdits` 自动接受文件编辑，但仍保留其他权限检查。如确实需要无人值守执行所有 Claude 工具，可在本地配置中显式改用 `bypassPermissions`，该模式风险更高，不作为默认值。

Agent 必须以拥有项目读写权限的普通系统用户运行。在 Linux/macOS 上应让该用户拥有项目目录和 `.git` 的读写权限；在 Windows 上需确保启动 Agent 的用户（包括服务账户）对项目目录具有“修改”权限。不要通过 root/管理员权限绕过目录授权。

Agent 进程在执行轮次期间意外退出时，下次注册会自动清理未完成的部分输出并重新派发该轮次。

Agent 正常退出时会主动上报离线并撤销本次运行令牌，控制台列表和详情页每 3 秒静默同步一次状态。进程崩溃、断电或网络中断无法主动上报时，由 30 秒心跳超时判定离线。

重置注册 Token 会立即注销旧的运行 Token。重置后需要替换工作站上的完整配置文件并重启 Agent。

## API

Agent 接口：

- `POST /v1/remote-agent/register`
- `POST /v1/remote-agent/heartbeat`
- `POST /v1/remote-agent/disconnect`
- `GET /v1/remote-agent/command/next`
- `POST /v1/remote-agent/command/acknowledge`
- `POST /v1/remote-agent/message/chunks`
- `POST /v1/remote-agent/message/result`

管理接口：

- `POST /v1/ai-applications/remote-agent/agent/create`
- `POST /v1/ai-applications/remote-agent/agent/update`
- `POST /v1/ai-applications/remote-agent/agent/page`
- `GET /v1/ai-applications/remote-agent/agent/get?id=<agent-id>`
- `GET /v1/ai-applications/remote-agent/agent/status?id=<agent-id>`
- `GET /v1/ai-applications/remote-agent/agent/events` (SSE)
- `POST /v1/ai-applications/remote-agent/agent/rotate-token?id=<agent-id>`
- `POST /v1/ai-applications/remote-agent/agent/status`
- `POST /v1/ai-applications/remote-agent/agent/delete`
- `POST /v1/ai-applications/remote-agent/conversation/create`
- `POST /v1/ai-applications/remote-agent/conversation/page`
- `GET /v1/ai-applications/remote-agent/conversation/get?id=<conversation-id>`
- `POST /v1/ai-applications/remote-agent/conversation/message/send`
- `POST /v1/ai-applications/remote-agent/conversation/message/stream`
