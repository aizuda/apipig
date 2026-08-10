package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	wechatModel "apipig/app/apps/wechat-bot/model"
	wechatReq "apipig/app/apps/wechat-bot/model/request"
	"apipig/core/api"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	ilink "github.com/openilink/openilink-sdk-go"
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

func TestWebhookCredentialsAreEncryptedAndSecretIsReturnedOnlyWhenChanged(t *testing.T) {
	setupRuntimeTestDB(t)
	botID := snowflake.ID(92010)
	createRuntimeTestBot(t, botID)
	service := newBotService(testVault{})

	var first, unchanged, rotated string
	var firstURL, rotatedURL string
	app := fiber.New()
	app.Post("/custom/v1/admin", func(c *fiber.Ctx) error {
		result, err := service.WebhookCredentials(&wechatReq.WebhookCredentialsParams{
			Ctx: c, Request: wechatReq.WebhookCredentialsRequest{ID: botID},
		})
		require.NoError(t, err)
		first, firstURL = result.WebhookSecret, result.WebhookURL
		result, err = service.WebhookCredentials(&wechatReq.WebhookCredentialsParams{
			Ctx: c, Request: wechatReq.WebhookCredentialsRequest{ID: botID},
		})
		require.NoError(t, err)
		unchanged = result.WebhookSecret
		result, err = service.WebhookCredentials(&wechatReq.WebhookCredentialsParams{
			Ctx: c, Request: wechatReq.WebhookCredentialsRequest{ID: botID, RotateSecret: true},
		})
		require.NoError(t, err)
		rotated, rotatedURL = result.WebhookSecret, result.WebhookURL
		return c.SendStatus(fiber.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/custom/v1/admin", nil)
	request.Host = "api.example"
	response, err := app.Test(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())

	require.NotEmpty(t, first)
	require.Empty(t, unchanged)
	require.NotEmpty(t, rotated)
	require.NotEqual(t, first, rotated)
	require.Equal(t, firstURL, rotatedURL)
	require.Contains(t, firstURL, "http://api.example/custom/v1/apps/wechat-bot/webhook/")
	var stored wechatModel.Bot
	require.NoError(t, global.DB.First(&stored, botID).Error)
	require.NotEmpty(t, stored.WebhookKey)
	require.Equal(t, "encrypted:"+rotated, stored.WebhookSecret)
}

func TestWebhookPushAuthenticatesAndDefaultsToLatestActiveContact(t *testing.T) {
	setupRuntimeTestDB(t)
	botID := snowflake.ID(92020)
	createRuntimeTestBot(t, botID)
	require.NoError(t, global.DB.Model(&wechatModel.Bot{}).Where("id = ?", botID).Updates(map[string]any{
		"webhook_key": "public-key", "webhook_secret": "encrypted:webhook-secret",
	}).Error)
	now := time.Now()
	contacts := []wechatModel.Contact{
		{MODEL: api.MODEL{ID: 92101}, BotRecordID: botID, UserID: "older", ContextToken: "encrypted:older-token", LastActiveAt: now.Add(-time.Hour).UnixMilli()},
		{MODEL: api.MODEL{ID: 92102}, BotRecordID: botID, UserID: "latest", ContextToken: "encrypted:latest-token", LastActiveAt: now.Add(-time.Minute).UnixMilli()},
		{MODEL: api.MODEL{ID: 92103}, BotRecordID: botID, UserID: "expired", ContextToken: "encrypted:expired-token", LastActiveAt: now.Add(-25 * time.Hour).UnixMilli()},
	}
	require.NoError(t, global.DB.Create(&contacts).Error)

	var received []ilink.SendMessageReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var payload ilink.SendMessageReq
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		received = append(received, payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0}`))
	}))
	defer server.Close()

	service := newBotService(testVault{})
	service.runtime.conns[botID] = &botConnection{client: ilink.NewClient("bot-token", ilink.WithBaseURL(server.URL))}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := testWebhookSignature("webhook-secret", timestamp)
	_, err := service.WebhookPush(&wechatReq.WebhookPushParams{
		WebhookKey: "public-key", Timestamp: timestamp, Signature: "wrong",
		Request: wechatReq.WebhookPushRequest{Content: "ignored"},
	})
	require.ErrorIs(t, err, ErrWebhookUnauthorized)
	require.Empty(t, received)

	result, err := service.WebhookPush(&wechatReq.WebhookPushParams{
		WebhookKey: "public-key", Timestamp: timestamp, Signature: signature,
		Request: wechatReq.WebhookPushRequest{Content: "latest message"},
	})
	require.NoError(t, err)
	require.Equal(t, "latest", result.Message.UserID)
	require.Len(t, received, 1)
	require.Equal(t, "latest", received[0].Msg.ToUserID)
	require.Equal(t, "latest-token", received[0].Msg.ContextToken)

	result, err = service.WebhookPush(&wechatReq.WebhookPushParams{
		WebhookKey: "public-key", Timestamp: timestamp, Signature: signature,
		Request: wechatReq.WebhookPushRequest{UserID: "older", Content: "targeted message"},
	})
	require.NoError(t, err)
	require.Equal(t, "older", result.Message.UserID)
	require.Len(t, received, 2)
	require.Equal(t, "older", received[1].Msg.ToUserID)
	require.NoError(t, service.SendTakeover(context.Background(), botID, "", "default target"))
	require.Len(t, received, 3)
	require.Equal(t, "latest", received[2].Msg.ToUserID)
}

func TestWebhookSignatureRejectsStaleTimestamp(t *testing.T) {
	now := time.Now()
	validTimestamp := strconv.FormatInt(now.UnixMilli(), 10)
	validSignature := testWebhookSignature("secret", validTimestamp)
	require.True(t, verifyWebhookSignature("secret", validTimestamp, validSignature, now))
	staleTimestamp := strconv.FormatInt(now.Add(-webhookTimestampSkew-time.Millisecond).UnixMilli(), 10)
	staleSignature := testWebhookSignature("secret", staleTimestamp)
	require.False(t, verifyWebhookSignature("secret", staleTimestamp, staleSignature, now))
}

func testWebhookSignature(secret, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
