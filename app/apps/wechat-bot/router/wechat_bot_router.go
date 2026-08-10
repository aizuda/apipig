package router

import (
	"apipig/app/apps/wechat-bot/api"

	"github.com/gofiber/fiber/v2"
)

type WechatBotRouter struct{}

func (r *WechatBotRouter) InitWebhookRouter(router fiber.Router) {
	router.Post("/apps/wechat-bot/webhook/:webhookKey", api.WechatBotApi.BotApi.WebhookPush)
}

func (r *WechatBotRouter) InitAdminRouter(router fiber.Router) {
	bot := router.Group("bot/")
	bot.Post("page", api.WechatBotApi.BotApi.Page)
	bot.Post("list", api.WechatBotApi.BotApi.List)
	bot.Post("bind/start", api.WechatBotApi.BotApi.StartBind)
	bot.Get("bind/status/:sessionId", api.WechatBotApi.BotApi.PollBind)
	bot.Post("rename", api.WechatBotApi.BotApi.Rename)
	bot.Post("reconnect", api.WechatBotApi.BotApi.Reconnect)
	bot.Post("enabled", api.WechatBotApi.BotApi.SetEnabled)
	bot.Post("delete", api.WechatBotApi.BotApi.Delete)
	bot.Get("contacts", api.WechatBotApi.BotApi.Contacts)
	bot.Post("messages", api.WechatBotApi.BotApi.Messages)
	bot.Post("send", api.WechatBotApi.BotApi.Send)
	bot.Post("webhook/credentials", api.WechatBotApi.BotApi.WebhookCredentials)
}
