package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"apipig/app/ai/model"
	coreAPI "apipig/core/api"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestStatusRoutesUpdateManagementResources(t *testing.T) {
	database := setupStatusRouterTestDB(t)
	providerID := snowflake.ID(301)
	channelID := snowflake.ID(302)
	tokenID := snowflake.ID(303)
	proxyID := snowflake.ID(304)

	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "provider", Code: "router-status-provider", Protocol: "openai", BaseURL: "https://example.com/v1", Status: 1,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL:      coreAPI.MODEL{ID: channelID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		ProviderID: providerID, Name: "channel", Weight: 1, Status: 1,
	}).Error)
	require.NoError(t, database.Create(&model.AccessToken{
		MODEL: coreAPI.MODEL{ID: tokenID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "token", Token: "token-hash", RPM: 60, Status: 1,
	}).Error)
	require.NoError(t, database.Create(&model.Proxy{
		MODEL: coreAPI.MODEL{ID: proxyID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "proxy", Scheme: "http", Host: "127.0.0.1", Port: 8080, Status: 1,
	}).Error)

	app := fiber.New()
	group := app.Group("/ai/")
	AiRouter.InitProviderRouter(group)
	AiRouter.InitChannelRouter(group)
	AiRouter.InitAccessTokenRouter(group)
	AiRouter.InitProxyRouter(group)

	cases := []struct {
		path string
		id   snowflake.ID
	}{
		{path: "/ai/gateway/provider/change-status", id: providerID},
		{path: "/ai/gateway/channel/change-status", id: channelID},
		{path: "/ai/gateway/token/change-status", id: tokenID},
		{path: "/ai/gateway/proxy/change-status", id: proxyID},
	}
	for _, testCase := range cases {
		body, err := json.Marshal(struct {
			ID     string `json:"id"`
			Status uint   `json:"status"`
		}{
			ID:     strconv.FormatInt(int64(testCase.id), 10),
			Status: 2,
		})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, testCase.path, bytes.NewReader(body))
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		response, err := app.Test(request)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.StatusCode)
		var payload struct {
			Code string `json:"code"`
			Data bool   `json:"data"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
		response.Body.Close()
		assert.Equal(t, "1", payload.Code)
		assert.True(t, payload.Data)
	}

	assertStatusValue(t, database, &model.Provider{}, providerID, 2)
	assertStatusValue(t, database, &model.Channel{}, channelID, 2)
	assertStatusValue(t, database, &model.AccessToken{}, tokenID, 2)
	assertStatusValue(t, database, &model.Proxy{}, proxyID, 2)
}

func setupStatusRouterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.Provider{}, &model.Channel{}, &model.AccessToken{}, &model.Proxy{}))
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
	return database
}

func assertStatusValue(t *testing.T, database *gorm.DB, resource any, id snowflake.ID, expected uint) {
	t.Helper()
	var status uint
	require.NoError(t, database.Model(resource).Select("status").Where("id = ?", id).Scan(&status).Error)
	assert.Equal(t, expected, status)
}
