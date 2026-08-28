package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apipig/app/ai/model"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zendev-sh/goai/provider"
)

func TestNormalizeProviderBaseURL(t *testing.T) {
	assert.Equal(t, "https://api.example.com/v1", normalizeProviderBaseURL("openai", "https://api.example.com"))
	assert.Equal(t, "https://api.example.com/v1", normalizeProviderBaseURL("openai", "https://api.example.com/v1/"))
	assert.Equal(t, "https://api.example.com/api/v1", normalizeProviderBaseURL("openai", "https://api.example.com/api/v1"))
	assert.Equal(t, "https://api.example.com", normalizeProviderBaseURL("custom", "https://api.example.com/"))
	assert.Equal(t, "https://api.example.com/compatible-mode/v1", normalizeProviderBaseURL("qwen", "https://api.example.com/compatible-mode/v1/"))
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

func TestQwenProviderUsesConfiguredCompatibleModeBaseURL(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"id":"chatcmpl-qwen","model":"qwen3-asr-flash","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	languageModel, err := buildLanguageModel(routeTarget{
		Provider: model.Provider{
			Protocol:  "qwen",
			BaseURL:   server.URL + "/compatible-mode/v1",
			TimeoutMs: 5000,
		},
		Account: model.ChannelAccount{APIKey: "dashscope-key"},
	}, "qwen3-asr-flash")
	require.NoError(t, err)
	result, err := languageModel.DoGenerate(context.Background(), provider.GenerateParams{
		Messages: []provider.Message{{
			Role:    provider.RoleUser,
			Content: []provider.Part{{Type: provider.PartText, Text: "hello"}},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Text)
	assert.Equal(t, "/compatible-mode/v1/chat/completions", requestedPath)
}

type qwenGatewayRepository struct {
	fakeGatewayRepository
	token model.AccessToken
}

func (r *qwenGatewayRepository) FindAccessToken(_ []string) (model.AccessToken, error) {
	return r.token, nil
}

func TestQwenASRChatCompletionsPreservesAudioRequest(t *testing.T) {
	var requestedPath string
	var requestedKey string
	var requestedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath = request.URL.Path
		requestedKey = request.Header.Get("Authorization")
		assert.NoError(t, json.NewDecoder(request.Body).Decode(&requestedBody))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"id":"chatcmpl-asr","model":"qwen3-asr-flash","choices":[{"index":0,"message":{"role":"assistant","content":"hello world"},"finish_reason":"stop"}],"usage":{"prompt_tokens":42,"completion_tokens":2,"total_tokens":44}}`)
	}))
	defer server.Close()

	providerID := snowflake.ID(601)
	channelID := snowflake.ID(602)
	tokenID := snowflake.ID(603)
	modelID := "qwen3-asr-flash"
	repository := &qwenGatewayRepository{
		token: model.AccessToken{
			MODEL:  coreAPI.MODEL{ID: tokenID},
			Name:   "qwen-token",
			Token:  "sk-qwen-test",
			Models: modelID,
			Status: gatewayStatusNormal,
		},
		fakeGatewayRepository: fakeGatewayRepository{config: routeConfig{
			Providers: []model.Provider{{
				MODEL:     coreAPI.MODEL{ID: providerID},
				Name:      "Qwen",
				Code:      "qwen",
				Protocol:  "qwen",
				BaseURL:   server.URL + "/compatible-mode/v1",
				Models:    modelID,
				TimeoutMs: 5000,
				Status:    gatewayStatusNormal,
			}},
			Channels: []model.Channel{{
				MODEL:      coreAPI.MODEL{ID: channelID},
				ProviderID: providerID,
				Name:       "qwen-channel",
				Weight:     1,
				Status:     gatewayStatusNormal,
			}},
			Accounts: []model.ChannelAccount{{
				MODEL:     coreAPI.MODEL{ID: snowflake.ID(604)},
				ChannelID: channelID,
				Name:      "qwen-account",
				APIKey:    "dashscope-key",
				Models:    `{"qwen3-asr-flash":"qwen3-asr-flash"}`,
				Status:    gatewayStatusNormal,
			}},
		}},
	}
	service := NewGatewayService(GatewayDependencies{
		Repository: repository,
		Vault:      newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" }),
	})

	app := fiber.New()
	app.Post("/v1/chat/completions", service.ChatCompletions)
	body := `{"model":"qwen3-asr-flash","messages":[{"role":"user","content":[{"type":"input_audio","input_audio":{"data":"https://example.com/audio.mp3"}}]}],"stream":false,"asr_options":{"language":"zh","enable_itn":true}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer sk-qwen-test")
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	require.NoError(t, err)
	defer response.Body.Close()

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, "/compatible-mode/v1/chat/completions", requestedPath)
	assert.Equal(t, "Bearer dashscope-key", requestedKey)
	assert.Equal(t, modelID, requestedBody["model"])
	assert.Equal(t, false, requestedBody["stream"])
	assert.Equal(t, map[string]any{"language": "zh", "enable_itn": true}, requestedBody["asr_options"])
	messages := requestedBody["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	audio := content[0].(map[string]any)["input_audio"].(map[string]any)
	assert.Equal(t, "https://example.com/audio.mp3", audio["data"])
}
