package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apipig/app/ai/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zendev-sh/goai/provider"
)

func TestNormalizeProviderBaseURL(t *testing.T) {
	assert.Equal(t, "https://api.example.com/v1", normalizeProviderBaseURL("openai", "https://api.example.com"))
	assert.Equal(t, "https://api.example.com/v1", normalizeProviderBaseURL("openai", "https://api.example.com/v1/"))
	assert.Equal(t, "https://api.example.com/api/v1", normalizeProviderBaseURL("openai", "https://api.example.com/api/v1"))
	assert.Equal(t, "https://api.example.com", normalizeProviderBaseURL("custom", "https://api.example.com/"))
}

func TestNormalizeAndValidateProviderIcon(t *testing.T) {
	providerModel := &model.Provider{
		Name: "DeepSeek", Code: "deepseek", Icon: " DeepSeek ", Protocol: "openai",
		BaseURL: "https://api.deepseek.com", TimeoutMs: 60_000, Status: gatewayStatusNormal,
	}
	normalizeProvider(providerModel)
	assert.Equal(t, "deepseek", providerModel.Icon)
	require.NoError(t, validateProvider(providerModel))

	providerModel.Icon = strings.Repeat("x", 51)
	require.EqualError(t, validateProvider(providerModel), "供应商图标标识不能超过 50 个字符")
}

func TestOpenAICompatibleGatewayUsesV1ChatCompletions(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"id":"chatcmpl-test","model":"gpt-test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	target := routeTarget{
		Provider: model.Provider{Protocol: "openai", BaseURL: server.URL, TimeoutMs: 5000},
		Account:  model.ChannelAccount{APIKey: "test-key"},
	}
	languageModel, err := buildLanguageModel(target, "gpt-test")
	require.NoError(t, err)
	params := provider.GenerateParams{Messages: []provider.Message{{
		Role:    provider.RoleUser,
		Content: []provider.Part{{Type: provider.PartText, Text: "hello"}},
	}}}
	applyGatewayProviderOptions(target, &params)
	result, err := languageModel.DoGenerate(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Text)
	assert.Equal(t, "/v1/chat/completions", requestedPath)
}
