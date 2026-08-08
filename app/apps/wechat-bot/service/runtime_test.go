package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wechatModel "apipig/app/apps/wechat-bot/model"
	wechatReq "apipig/app/apps/wechat-bot/model/request"
	"apipig/core/api"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type testVault struct{}

func (testVault) Encrypt(value string) (string, error) { return "encrypted:" + value, nil }
func (testVault) Decrypt(value string) (string, error) {
	return strings.TrimPrefix(value, "encrypted:"), nil
}
func (testVault) Enabled() bool { return true }

func setupRuntimeTestDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&wechatModel.Bot{}, &wechatModel.Contact{}, &wechatModel.Message{}))
	global.DB = database
}

func createRuntimeTestBot(t *testing.T, id snowflake.ID) {
	t.Helper()
	require.NoError(t, global.DB.Create(&wechatModel.Bot{
		MODEL: api.MODEL{ID: id, CreatedAt: time.Now().UnixMilli()}, Name: "test",
		BotID: "wx-bot", BotToken: "encrypted:token", BaseURL: "https://example.test",
		Status: wechatModel.BotStatusOnline, Enabled: true,
	}).Error)
}

func TestRuntimeStoreInboundIsIdempotent(t *testing.T) {
	setupRuntimeTestDB(t)
	botID := snowflake.ID(90001)
	createRuntimeTestBot(t, botID)
	runtime := newRuntime(testVault{})
	message := ilink.WeixinMessage{
		MessageID: 42, FromUserID: "contact-1", ContextToken: "context-1",
		CreateTimeMs: time.Now().UnixMilli(),
		ItemList:     []ilink.MessageItem{{Type: ilink.ItemText, TextItem: &ilink.TextItem{Text: "hello"}}},
	}

	require.NoError(t, runtime.storeInbound(botID, message))
	require.NoError(t, runtime.storeInbound(botID, message))

	var bot wechatModel.Bot
	require.NoError(t, global.DB.First(&bot, botID).Error)
	require.Equal(t, int64(1), bot.MessageCount)

	var contact wechatModel.Contact
	require.NoError(t, global.DB.Where("bot_record_id = ? AND user_id = ?", botID, "contact-1").First(&contact).Error)
	require.Equal(t, int64(1), contact.MessageCount)
	require.Equal(t, "encrypted:context-1", contact.ContextToken)
	require.Equal(t, "hello", contact.LastMessage)

	var messageCount int64
	require.NoError(t, global.DB.Model(&wechatModel.Message{}).Count(&messageCount).Error)
	require.Equal(t, int64(1), messageCount)
}

func TestRuntimeSendUsesStoredContextAndPersistsOutboundMessage(t *testing.T) {
	setupRuntimeTestDB(t)
	botID := snowflake.ID(90002)
	createRuntimeTestBot(t, botID)
	require.NoError(t, global.DB.Create(&wechatModel.Contact{
		MODEL: api.MODEL{ID: 91002, CreatedAt: time.Now().UnixMilli()}, BotRecordID: botID,
		UserID: "contact-2", ContextToken: "encrypted:context-2", LastActiveAt: time.Now().UnixMilli(),
	}).Error)

	var received ilink.SendMessageReq
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/ilink/bot/sendmessage", request.URL.Path)
		require.NoError(t, json.NewDecoder(request.Body).Decode(&received))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0}`))
	}))
	defer server.Close()

	runtime := newRuntime(testVault{})
	runtime.conns[botID] = &botConnection{client: ilink.NewClient("bot-token", ilink.WithBaseURL(server.URL))}
	sent, err := runtime.Send(context.Background(), botID, "contact-2", "reply")
	require.NoError(t, err)
	require.Equal(t, "outbound", sent.Direction)
	require.Equal(t, "context-2", received.Msg.ContextToken)
	require.Equal(t, "contact-2", received.Msg.ToUserID)
	require.Equal(t, "reply", received.Msg.ItemList[0].TextItem.Text)

	var stored wechatModel.Message
	require.NoError(t, global.DB.First(&stored, sent.ID).Error)
	require.Equal(t, "reply", stored.Content)
}

func TestRuntimeSendRejectsExpiredContext(t *testing.T) {
	setupRuntimeTestDB(t)
	botID := snowflake.ID(90003)
	createRuntimeTestBot(t, botID)
	require.NoError(t, global.DB.Create(&wechatModel.Contact{
		MODEL: api.MODEL{ID: 91003, CreatedAt: time.Now().UnixMilli()}, BotRecordID: botID,
		UserID: "contact-3", ContextToken: "encrypted:context-3",
		LastActiveAt: time.Now().Add(-sendWindow).UnixMilli(),
	}).Error)
	runtime := newRuntime(testVault{})
	runtime.conns[botID] = &botConnection{client: ilink.NewClient("bot-token")}

	_, err := runtime.Send(context.Background(), botID, "contact-3", "reply")
	require.ErrorContains(t, err, "超过 24 小时")
}

func TestBotServiceSendMirrorsOnlyAfterWechatAcceptsMessage(t *testing.T) {
	setupRuntimeTestDB(t)
	botID := snowflake.ID(90004)
	createRuntimeTestBot(t, botID)
	require.NoError(t, global.DB.Create(&wechatModel.Contact{
		MODEL: api.MODEL{ID: 91004, CreatedAt: time.Now().UnixMilli()}, BotRecordID: botID,
		UserID: "contact-4", ContextToken: "encrypted:context-4", LastActiveAt: time.Now().UnixMilli(),
	}).Error)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0}`))
	}))
	defer server.Close()

	service := newBotService(testVault{})
	service.runtime.conns[botID] = &botConnection{client: ilink.NewClient("bot-token", ilink.WithBaseURL(server.URL))}
	var mirroredBotID snowflake.ID
	var mirroredUserID, mirroredContent string
	service.SetOutboundHandler(func(id snowflake.ID, userID, content string) error {
		mirroredBotID, mirroredUserID, mirroredContent = id, userID, content
		return errors.New("local mirror failed")
	})

	sent, err := service.Send(&wechatReq.SendMessageRequest{BotID: botID, UserID: "contact-4", Content: "hello"})
	require.NoError(t, err)
	require.Equal(t, "outbound", sent.Direction)
	require.Equal(t, botID, mirroredBotID)
	require.Equal(t, "contact-4", mirroredUserID)
	require.Equal(t, "hello", mirroredContent)
}
