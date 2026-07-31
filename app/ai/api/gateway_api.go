package api

import (
	aiReq "apipig/app/ai/model/request"
	"apipig/app/ai/service"
	"apipig/core/api"
	"apipig/core/api/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// GatewayApi 负责 AI 网关的汇总查询和 OpenAI 兼容流量入口。
//
// 供应商、渠道、访问令牌、代理节点和调用日志等表接口分别由对应的 API 类型负责，
// 当前类型仅保留跨表汇总与网关转发编排，避免管理职责与流量职责继续混杂。
type GatewayApi struct {
	api.API
	service *service.GatewayService
}

func NewGatewayApi(gatewayService *service.GatewayService) *GatewayApi {
	return &GatewayApi{service: gatewayService}
}

// Summary 查询 AI 网关管理面汇总数据。
func (a *GatewayApi) Summary(c *fiber.Ctx) error {
	params := new(aiReq.GatewaySummaryParams)
	var parseErr error
	if value := c.Query("startAt"); value != "" {
		params.StartAt, parseErr = strconv.ParseInt(value, 10, 64)
	}
	if parseErr == nil {
		if value := c.Query("endAt"); value != "" {
			params.EndAt, parseErr = strconv.ParseInt(value, 10, 64)
		}
	}
	return response.Execute(c, a.service.Summary, params, parseErr)
}

// AIChat 使用管理端选定的 API 密钥执行一次临时聊天测试。
func (a *GatewayApi) AIChat(c *fiber.Ctx) error {
	params := new(aiReq.AIChatParams)
	err := a.BodyParserVerify(c, params, "AI Chat")
	params.Ctx = c
	return response.Execute(c, a.service.AIChat, params, err)
}

// AIChatStream 使用管理端选定的 API 密钥执行 SSE 流式聊天测试。
func (a *GatewayApi) AIChatStream(c *fiber.Ctx) error {
	params := new(aiReq.AIChatParams)
	if err := a.BodyParserVerify(c, params, "AI Chat"); err != nil {
		return response.Failed(c, err.Error())
	}
	params.Ctx = c
	if err := a.service.AIChatStream(params); err != nil {
		return response.Failed(c, err.Error())
	}
	return nil
}

// ProxyOpenAI 处理 OpenAI 兼容协议的对外流量转发。
func (a *GatewayApi) ProxyOpenAI(c *fiber.Ctx) error {
	upstream := "/" + c.Params("*")
	return a.service.ProxyOpenAI(&aiReq.GatewayProxyParams{
		Ctx:      c,
		RawBody:  c.Body(),
		Upstream: upstream,
	})
}

// ChatCompletions 处理 OpenAI Chat Completions 兼容协议。
func (a *GatewayApi) ChatCompletions(c *fiber.Ctx) error {
	return a.service.ChatCompletions(c)
}

// Messages 处理 Anthropic Messages 兼容协议。
func (a *GatewayApi) Messages(c *fiber.Ctx) error {
	return a.service.Messages(c)
}

// Models 返回当前访问令牌可用的模型列表。
func (a *GatewayApi) Models(c *fiber.Ctx) error {
	return a.service.Models(c)
}

// Healthz 返回网关存活状态。
func (a *GatewayApi) Healthz(c *fiber.Ctx) error {
	return a.service.Healthz(c)
}
