package service

import (
	"testing"
	"time"

	wechatModel "apipig/app/apps/wechat-bot/model"
	wechatReq "apipig/app/apps/wechat-bot/model/request"
	"apipig/core/api"
	"apipig/global"

	"github.com/stretchr/testify/require"
)

func TestBotServiceListSearchesCustomNameAndFiltersAvailability(t *testing.T) {
	setupRuntimeTestDB(t)
	now := time.Now().UnixMilli()
	bots := []wechatModel.Bot{
		{MODEL: api.MODEL{ID: 92001, CreatedAt: now}, Name: "评审通知", BotID: "wx-review-online", BotToken: "token", BaseURL: "https://example.test", Status: wechatModel.BotStatusOnline, Enabled: true},
		{MODEL: api.MODEL{ID: 92002, CreatedAt: now}, Name: "评审离线", BotID: "wx-review-offline", BotToken: "token", BaseURL: "https://example.test", Status: wechatModel.BotStatusOffline, Enabled: true},
		{MODEL: api.MODEL{ID: 92003, CreatedAt: now}, Name: "评审停用", BotID: "wx-review-disabled", BotToken: "token", BaseURL: "https://example.test", Status: wechatModel.BotStatusOnline, Enabled: false},
		{MODEL: api.MODEL{ID: 92004, CreatedAt: now}, Name: "日常通知", BotID: "wx-daily-online", BotToken: "token", BaseURL: "https://example.test", Status: wechatModel.BotStatusOnline, Enabled: true},
	}
	require.NoError(t, global.DB.Create(&bots).Error)
	require.NoError(t, global.DB.Model(&wechatModel.Bot{}).Where("id = ?", 92003).Update("enabled", false).Error)

	enabled := true
	service := newBotService(testVault{})
	result, err := service.List(&wechatReq.BotListParams{
		Name: " 评审 ", Status: "online", Enabled: &enabled,
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "评审通知", result[0].Name)

	result, err = service.List(&wechatReq.BotListParams{Name: "wx-review-online"})
	require.NoError(t, err)
	require.Empty(t, result, "list search must only match the custom bot name")
}
