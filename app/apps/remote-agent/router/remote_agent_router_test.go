package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	coreAPI "apipig/core/api"
	coreResp "apipig/core/api/response"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAgentRegisterAndHeartbeatRoutes(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&remoteModel.Agent{}, &remoteModel.Heartbeat{}, &remoteModel.Conversation{}, &remoteModel.Message{}, &remoteModel.MessageChunk{}, &remoteModel.Command{}))
	previousDB := global.DB
	previousConfig := global.CONFIG
	global.DB = database
	global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds = 90
	seedRouterAgent(t, database, "route-node", "route node", "bootstrap-secret")
	t.Cleanup(func() {
		global.DB = previousDB
		global.CONFIG = previousConfig
	})

	app := fiber.New()
	group := app.Group("/remote-agent/")
	RemoteAgentRoutes.InitAgentRouter(group)

	registerRequest := httptest.NewRequest(http.MethodPost, "/remote-agent/register", bytes.NewBufferString(`{"agentKey":"route-node","hostname":"route-host"}`))
	registerRequest.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	registerRequest.Header.Set("X-Agent-Registration-Token", "bootstrap-secret")
	registerResponse, err := app.Test(registerRequest)
	require.NoError(t, err)
	defer registerResponse.Body.Close()
	var registration struct {
		coreResp.Response
		Data struct {
			AgentID    string `json:"agentId"`
			AgentToken string `json:"agentToken"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(registerResponse.Body).Decode(&registration))
	assert.Equal(t, coreResp.Success, registration.Code)
	assert.NotEmpty(t, registration.Data.AgentID)
	assert.NotEmpty(t, registration.Data.AgentToken)

	heartbeatRequest := httptest.NewRequest(http.MethodPost, "/remote-agent/heartbeat", bytes.NewBufferString(`{"cpuUsage":12.5,"memoryUsed":1024,"busy":false}`))
	heartbeatRequest.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	heartbeatRequest.Header.Set(fiber.HeaderAuthorization, "Bearer "+registration.Data.AgentToken)
	heartbeatResponse, err := app.Test(heartbeatRequest)
	require.NoError(t, err)
	defer heartbeatResponse.Body.Close()
	var heartbeat coreResp.Response
	require.NoError(t, json.NewDecoder(heartbeatResponse.Body).Decode(&heartbeat))
	assert.Equal(t, coreResp.Success, heartbeat.Code)
}

func TestAgentDeleteAdminRoute(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&remoteModel.Agent{}, &remoteModel.Heartbeat{}, &remoteModel.Conversation{}, &remoteModel.Message{}, &remoteModel.MessageChunk{}, &remoteModel.Command{}))
	previousDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previousDB })
	agent := seedRouterAgent(t, database, "delete-route-node", "delete route node", "bootstrap-secret")

	app := fiber.New()
	group := app.Group("/admin/")
	RemoteAgentRoutes.InitAdminRouter(group)
	body, err := json.Marshal(map[string]string{"id": agent.ID.String()})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/admin/agent/delete", bytes.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	result, err := app.Test(request)
	require.NoError(t, err)
	defer result.Body.Close()
	var envelope coreResp.Response
	require.NoError(t, json.NewDecoder(result.Body).Decode(&envelope))
	assert.Equal(t, coreResp.Success, envelope.Code)

	var count int64
	require.NoError(t, database.Unscoped().Model(&remoteModel.Agent{}).Where("id = ?", agent.ID).Count(&count).Error)
	assert.Zero(t, count)
}

func TestConversationPinRenameAndDeleteAdminRoutes(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&remoteModel.Agent{}, &remoteModel.Heartbeat{}, &remoteModel.Conversation{}, &remoteModel.Message{}, &remoteModel.MessageChunk{}, &remoteModel.Command{}))
	previousDB := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previousDB })
	agent := seedRouterAgent(t, database, "conversation-delete-node", "conversation delete node", "bootstrap-secret")
	conversation := remoteModel.Conversation{
		MODEL:   coreAPI.MODEL{ID: snowflake.ID(30_001), CreatedBy: "test", CreatedAt: time.Now().UnixMilli()},
		AgentID: agent.ID, Title: "Delete me", Status: remoteModel.ConversationStatusArchived,
		LastMessageAt: time.Now().UnixMilli(),
	}
	require.NoError(t, database.Create(&conversation).Error)

	app := fiber.New()
	group := app.Group("/admin/")
	RemoteAgentRoutes.InitAdminRouter(group)
	pinBody, err := json.Marshal(map[string]any{"id": conversation.ID.String(), "pinned": true})
	require.NoError(t, err)
	pinRequest := httptest.NewRequest(http.MethodPost, "/admin/conversation/pin", bytes.NewReader(pinBody))
	pinRequest.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	pinResult, err := app.Test(pinRequest)
	require.NoError(t, err)
	defer pinResult.Body.Close()
	var pinEnvelope coreResp.Response
	require.NoError(t, json.NewDecoder(pinResult.Body).Decode(&pinEnvelope))
	assert.Equal(t, coreResp.Success, pinEnvelope.Code)
	var pinnedConversation remoteModel.Conversation
	require.NoError(t, database.First(&pinnedConversation, conversation.ID).Error)
	assert.True(t, pinnedConversation.Pinned)
	assert.NotZero(t, pinnedConversation.PinnedAt)
	renameBody, err := json.Marshal(map[string]any{"id": conversation.ID.String(), "title": "Renamed by route"})
	require.NoError(t, err)
	renameRequest := httptest.NewRequest(http.MethodPost, "/admin/conversation/rename", bytes.NewReader(renameBody))
	renameRequest.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	renameResult, err := app.Test(renameRequest)
	require.NoError(t, err)
	defer renameResult.Body.Close()
	var renameEnvelope coreResp.Response
	require.NoError(t, json.NewDecoder(renameResult.Body).Decode(&renameEnvelope))
	assert.Equal(t, coreResp.Success, renameEnvelope.Code)
	require.NoError(t, database.First(&pinnedConversation, conversation.ID).Error)
	assert.Equal(t, "Renamed by route", pinnedConversation.Title)

	body, err := json.Marshal(map[string]string{"id": conversation.ID.String()})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/admin/conversation/delete", bytes.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	result, err := app.Test(request)
	require.NoError(t, err)
	defer result.Body.Close()
	var envelope coreResp.Response
	require.NoError(t, json.NewDecoder(result.Body).Decode(&envelope))
	assert.Equal(t, coreResp.Success, envelope.Code)

	var count int64
	require.NoError(t, database.Unscoped().Model(&remoteModel.Conversation{}).Where("id = ?", conversation.ID).Count(&count).Error)
	assert.Zero(t, count)
}
