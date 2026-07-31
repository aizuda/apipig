package request

import (
	"apipig/toolkit/snowflake"
	"github.com/gofiber/fiber/v2"
)

// AIChatMessage 表示管理端临时聊天会话中的一条消息。
type AIChatMessage struct {
	Role    string `json:"role" validate:"required"`
	Content string `json:"content" validate:"required"`
}

// AIChatParams 定义管理端使用指定 API 密钥发起无状态聊天测试的参数。
type AIChatParams struct {
	Ctx         *fiber.Ctx      `json:"-"`
	TokenID     snowflake.ID    `json:"tokenId" swaggertype:"string" validate:"required"`
	Model       string          `json:"model" validate:"required"`
	Messages    []AIChatMessage `json:"messages" validate:"required"`
	Temperature *float64        `json:"temperature"`
	MaxTokens   int             `json:"maxTokens"`
}

// GatewayProxyParams 封装一次 OpenAI 兼容代理请求所需的运行时上下文。
type GatewayProxyParams struct {
	Ctx      *fiber.Ctx `json:"-"` // Fiber 请求上下文
	RawBody  []byte     `json:"-"` // 未经修改的原始请求体
	Upstream string     `json:"-"` // 上游相对路径，例如 /chat/completions
}

type GatewaySummaryParams struct {
	StartAt int64 `json:"startAt"`
	EndAt   int64 `json:"endAt"`
}
