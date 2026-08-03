# Remote Agent Controller

## Controller configuration

```yaml
remote-agent:
  registration-token: replace-with-a-long-random-secret
  heartbeat-timeout-seconds: 90
  offline-check-cron: "* * * * *"
  command-poll-timeout-seconds: 25
  dispatch-lease-seconds: 30
```

Existing installations can set the bootstrap secret with the
`APIPIG_REMOTE_AGENT_REGISTRATION_TOKEN` environment variable. New installations
generate the secret during setup. Agent-specific tokens are returned on registration
and only their SHA-256 hashes are stored by the Controller.

## Build and run an Agent

1. Install `git` and the Codex CLI on the worker machine.
2. Create a configuration based on `../cmd/remote-agent/remote-agent.example.yaml`.
3. Build the Agent:

   ```shell
   go build -o remote-agent ./cmd/remote-agent
   ```

4. Run it as a non-root operating-system user:

   ```shell
   ./remote-agent -c remote-agent.yaml
   ```

The Agent clones every task into `<workspace-root>/<task-id>`. It invokes Git and
Codex directly without a shell. The default Codex invocation is equivalent to:

```shell
codex exec --skip-git-repo-check -
```

The prompt is sent through stdin instead of being exposed as a process argument.

To build all supported release archives from the repository root, run:

```shell
goreleaser release --snapshot --clean --config cmd/remote-agent/.goreleaser.yaml
```

Release archives are written to `dist/remote-agent/` for Linux, Windows, and
macOS on amd64 and arm64. Each archive includes `../cmd/remote-agent/remote-agent.example.yaml` and
this deployment reference.

## Controller APIs

Agent token APIs:

- `POST /v1/remote-agent/register`
- `POST /v1/remote-agent/heartbeat`
- `GET /v1/remote-agent/command/next`
- `POST /v1/remote-agent/command/acknowledge`
- `POST /v1/remote-agent/task/result`

JWT/RBAC management APIs:

- `POST /v1/ai-applications/remote-agent/task/create`
- `POST /v1/ai-applications/remote-agent/task/page`
- `GET /v1/ai-applications/remote-agent/task/get?id=<task-id>`
- `POST /v1/ai-applications/remote-agent/task/cancel`

Only HTTPS Git repository URLs without embedded credentials are accepted. Working
directories must be clean relative paths inside the cloned repository. The Controller
does not expose a general-purpose shell command endpoint.
