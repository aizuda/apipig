# Code Review Push Channels

Project review results can be delivered to multiple channels. The project list exposes a
`Push channels` action; channel records are managed independently from the repository and AI
configuration.

## Storage design

`ap_review_push_channel` contains one row per project/channel:

| Column | Purpose |
| --- | --- |
| `project_id` | Owning review project |
| `type` | `wecom_robot`, `dingtalk_robot`, or `wechat_bot` |
| `name` | User-facing label |
| `enabled` | Per-channel switch |
| `config` | Encrypted JSON payload |

The JSON shape is intentionally extensible. Current payloads are:

- WeCom robot: `{ "webhookUrl": "..." }`
- DingTalk robot: `{ "webhookUrl": "...", "secret": "..." }` (secret is optional)
- WeChat Bot: `{ "botId": "...", "userId": "..." }`; `userId` may be empty to use the
  most recently active contact within the 24-hour send window.

Webhook URLs and signing secrets are encrypted with the existing `CredentialVault`. Project list
responses return masked secret fields and a `secretConfigured` flag. The authenticated channel
editor endpoint returns decrypted values so the password inputs can be populated and revealed by
the operator; leaving a masked field empty still preserves the stored value when channels are saved.

## Admin API

- `POST /v1/ai-applications/code-review/project/push-channels` replaces the channel set for a project.
- `GET /v1/ai-applications/code-review/project/push-channels?id={projectId}` returns channel values for the authenticated editor.
- `POST /v1/ai-applications/code-review/project/push-channels/test` sends a test message.

When a review task reaches `succeeded`, enabled channels are dispatched asynchronously. A channel
failure is logged and does not change the review task result. DingTalk signing follows the official
timestamp + HMAC-SHA256 convention; WeCom uses the robot webhook payload, and WeChat Bot sends to
the selected contact, or the most recently active contact when `userId` is empty, through the
existing Bot runtime.
