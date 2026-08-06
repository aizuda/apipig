package service

import (
	"context"

	aiService "apipig/app/ai/service"
)

type WechatBotServiceGroup struct {
	BotService *BotService
}

func NewWechatBotServiceGroup() *WechatBotServiceGroup {
	return &WechatBotServiceGroup{BotService: newBotService(aiService.AiService.Vault)}
}

var WechatBotService = NewWechatBotServiceGroup()

func StartWechatBots() {
	WechatBotService.BotService.runtime.StartAll()
}

func ShutdownWechatBots(ctx context.Context) error {
	return WechatBotService.BotService.runtime.Shutdown(ctx)
}
