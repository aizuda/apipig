package router

import (
	"apipig/app/ai/api"
	"github.com/gofiber/fiber/v2"
)

// GatewayRouter 负责注册 AI 网关跨表汇总与 OpenAI 兼容流量入口。
type GatewayRouter struct{}

// InitGatewayAdminRouter 注册 AI 网关管理面的公共汇总接口。
//
// 供应商、渠道、访问令牌、代理节点和调用日志接口分别由对应路由类型注册，
// 这样每个路由文件只维护一张表的 URL 与处理器映射关系。
func (r *GatewayRouter) InitGatewayAdminRouter(router fiber.Router) {
	rg := router.Group("gateway")
	a := api.AiApi.GatewayApi
	rg.Get("summary", a.Summary)
	rg.Post("chat", a.AIChat)
	rg.Post("chat/stream", a.AIChatStream)
}

// InitGatewayProxyRouter 注册 OpenAI 兼容流量入口。
//
// 入口示例：
// POST /v1/ai-gateway/openai/chat/completions
// POST /v1/ai-gateway/openai/embeddings
func (r *GatewayRouter) InitGatewayProxyRouter(router fiber.Router) {
	rg := router.Group("openai")
	rg.All("/*", api.AiApi.GatewayApi.ProxyOpenAI)
}

// InitProtocolRouter 注册对外兼容 OpenAI 与 Anthropic 的标准接口。
func (r *GatewayRouter) InitProtocolRouter(router fiber.Router) {
	a := api.AiApi.GatewayApi
	router.Post("/chat/completions", a.ChatCompletions)
	router.Post("/embeddings", a.Embeddings)
	router.Post("/images/generations", a.ImageGenerations)
	router.Post("/rerank", a.Rerank)
	router.Post("/audio/speech", a.AudioSpeech)
	router.Post("/audio/transcriptions", a.AudioTranscriptions)
	router.Post("/messages", a.Messages)
	router.Get("/models", a.Models)
}

// InitHealthRouter 注册不需要鉴权的存活探针。
func (r *GatewayRouter) InitHealthRouter(router fiber.Router) {
	router.Get("/healthz", api.AiApi.GatewayApi.Healthz)
}
