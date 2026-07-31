package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayChatStreamRouteIsRegistered(t *testing.T) {
	app := fiber.New()
	group := app.Group("/ai/")
	AiRouter.InitGatewayAdminRouter(group)

	request := httptest.NewRequest(http.MethodPost, "/ai/gateway/chat/stream", bytes.NewBufferString(`{}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	result, err := app.Test(request)
	require.NoError(t, err)
	defer result.Body.Close()
	assert.Equal(t, http.StatusOK, result.StatusCode)

	var payload struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(t, json.NewDecoder(result.Body).Decode(&payload))
	assert.Equal(t, "0", payload.Code)
	assert.NotEmpty(t, payload.Msg)
}
