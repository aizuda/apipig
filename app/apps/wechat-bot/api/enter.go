package api

import "apipig/app/apps/wechat-bot/service"

type WechatBotApiGroup struct {
	BotApi *BotApi
}

func NewWechatBotApiGroup(services *service.WechatBotServiceGroup) *WechatBotApiGroup {
	return &WechatBotApiGroup{BotApi: &BotApi{service: services.BotService}}
}

var WechatBotApi = NewWechatBotApiGroup(service.WechatBotService)
