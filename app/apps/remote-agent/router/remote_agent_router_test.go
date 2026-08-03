package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	remoteModel "apipig/app/apps/remote-agent/model"
	coreResp "apipig/core/api/response"
	"apipig/global"

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
	require.NoError(t, database.AutoMigrate(&remoteModel.Agent{}, &remoteModel.Heartbeat{}))
	previousDB := global.DB
	previousConfig := global.CONFIG
	global.DB = database
	global.CONFIG.RemoteAgent.RegistrationToken = "bootstrap-secret"
	global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds = 90
	t.Cleanup(func() {
		global.DB = previousDB
		global.CONFIG = previousConfig
	})

	app := fiber.New()
	group := app.Group("/remote-agent/")
	RemoteAgentRoutes.InitAgentRouter(group)

	registerRequest := httptest.NewRequest(http.MethodPost, "/remote-agent/register", bytes.NewBufferString(`{"agentKey":"route-node","name":"route node","hostname":"route-host"}`))
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
