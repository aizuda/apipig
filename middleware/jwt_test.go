package middleware

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestJwt(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJJRCI6IjE4MTI3NjExMDkyMjkyMDc1NTIiLCJVc2VybmFtZSI6ImFkbWluIiwiTmlja05hbWUiOiIiLCJleHAiOjE3MjI5MzcwMTMsImlhdCI6MTcyMjkzNTIxM30.mfUQ3B06ADIz-ozZ85y7_jxsnfLrutG_xkI5AUbgyOU"
	jwt := &JWT{[]byte("jC4ruWCOt29eyJhbGciOiJIUzI"), 30}
	tc, err := jwt.ParseToken(token)
	if err == nil {
		fmt.Println(tc.Username)
	}
}

func TestAPITokenSessionAllowedRoutes(t *testing.T) {
	app := fiber.New()
	handler := func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"allowed": isAPITokenSessionAllowed(c)})
	}
	app.Post("/v1/ai/gateway/log/page", handler)
	app.Post("/v1/ai/gateway/token/statistics", handler)
	app.Post("/v1/ai/gateway/token/page", handler)
	app.Get("/v1/ai/gateway/log/page", handler)

	tests := []struct {
		method  string
		path    string
		allowed bool
	}{
		{fiber.MethodPost, "/v1/ai/gateway/log/page", true},
		{fiber.MethodPost, "/v1/ai/gateway/token/statistics", true},
		{fiber.MethodPost, "/v1/ai/gateway/token/page", false},
		{fiber.MethodGet, "/v1/ai/gateway/log/page", false},
	}
	for _, test := range tests {
		response, err := app.Test(httptest.NewRequest(test.method, test.path, nil))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, response.StatusCode)
		var body struct {
			Allowed bool `json:"allowed"`
		}
		assert.NoError(t, json.NewDecoder(response.Body).Decode(&body))
		assert.Equal(t, test.allowed, body.Allowed)
		assert.NoError(t, response.Body.Close())
	}
}
