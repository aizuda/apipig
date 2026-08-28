package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"apipig/app/ai/model"
	coreAPI "apipig/core/api"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestAccessTokenResetRouteReturnsNewTokenOnce(t *testing.T) {
	database := setupStatusRouterTestDB(t)
	previousEncryptionKey := global.CONFIG.AI.EncryptionKey
	global.CONFIG.AI.EncryptionKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONFIG.AI.EncryptionKey = previousEncryptionKey })

	tokenID := snowflake.ID(401)
	require.NoError(t, database.Create(&model.AccessToken{
		MODEL: coreAPI.MODEL{ID: tokenID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "token", Token: "old-token", RPM: 60, Status: 1,
	}).Error)

	app := fiber.New()
	AiRouter.InitAccessTokenRouter(app.Group("/ai/"))
	body, err := json.Marshal(map[string]string{"id": strconv.FormatInt(int64(tokenID), 10)})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/ai/gateway/token/reset", bytes.NewReader(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)

	var payload struct {
		Code string `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	require.Equal(t, "1", payload.Code)
	require.Regexp(t, `^sk-[A-Za-z0-9_-]{43}$`, payload.Data.Token)

	var storedToken string
	require.NoError(t, database.Model(&model.AccessToken{}).
		Select("token").Where("id = ?", tokenID).Scan(&storedToken).Error)
	require.True(t, strings.HasPrefix(storedToken, "enc:v1:"))
	require.NotEqual(t, payload.Data.Token, storedToken)
}
