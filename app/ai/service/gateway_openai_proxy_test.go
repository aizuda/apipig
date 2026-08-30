package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"apipig/app/ai/model"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type openAIProxyTestRepository struct {
	fakeGatewayRepository
	token model.AccessToken
}

func (r *openAIProxyTestRepository) FindAccessToken(candidates []string) (model.AccessToken, error) {
	expected, _ := gatewayTokenLookupCandidates(r.token.Token)
	if !slices.Equal(candidates, expected) {
		return model.AccessToken{}, errors.New("unexpected token candidates")
	}
	return r.token, nil
}

func newOpenAIProxyTestService(upstreamURL string) (*GatewayService, *openAIProxyTestRepository) {
	const logicalModel = "public-model"
	repository := &openAIProxyTestRepository{
		token: model.AccessToken{
			MODEL: coreAPI.MODEL{ID: snowflake.ID(7103)}, Name: "test-token", Token: "sk-test",
			Models: logicalModel, Status: gatewayStatusNormal,
		},
		fakeGatewayRepository: fakeGatewayRepository{config: routeConfig{
			Providers: []model.Provider{{
				MODEL: coreAPI.MODEL{ID: snowflake.ID(7101)}, Name: "OpenAI", Code: "openai",
				Protocol: "openai", BaseURL: upstreamURL + "/v1", Models: "upstream-model",
				TimeoutMs: 5_000, Status: gatewayStatusNormal,
			}},
			Channels: []model.Channel{{
				MODEL: coreAPI.MODEL{ID: snowflake.ID(7102)}, ProviderID: snowflake.ID(7101),
				Name: "primary", Weight: 1, Status: gatewayStatusNormal,
			}},
			Accounts: []model.ChannelAccount{{
				MODEL: coreAPI.MODEL{ID: snowflake.ID(7104)}, ChannelID: snowflake.ID(7102),
				Name: "account", APIKey: "upstream-key",
				Models: `{"public-model":"upstream-model"}`, Status: gatewayStatusNormal,
			}},
		}},
	}
	service := NewGatewayService(GatewayDependencies{
		Repository: repository,
		Vault:      newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" }),
	})
	return service, repository
}

func TestOpenAIJSONProtocolEndpointsPassthrough(t *testing.T) {
	var mu sync.Mutex
	paths := make([]string, 0, 4)
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		assert.Equal(t, "Bearer upstream-key", request.Header.Get("Authorization"))
		var payload map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, "upstream-model", payload["model"])
		mu.Lock()
		paths = append(paths, request.URL.Path)
		mu.Unlock()
		switch request.URL.Path {
		case "/v1/embeddings":
			writer.Header().Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			_, _ = fmt.Fprint(writer, `{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2]}],"model":"upstream-model","usage":{"prompt_tokens":3,"total_tokens":3}}`)
		case "/v1/images/generations":
			writer.Header().Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			_, _ = fmt.Fprint(writer, `{"created":1,"data":[{"b64_json":"aW1hZ2U="}],"usage":{"input_tokens":5,"output_tokens":7,"total_tokens":12}}`)
		case "/v1/rerank":
			writer.Header().Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			_, _ = fmt.Fprint(writer, `{"results":[{"index":0,"relevance_score":0.99}],"usage":{"total_tokens":4}}`)
		case "/v1/audio/speech":
			writer.Header().Set(fiber.HeaderContentType, "audio/mpeg")
			_, _ = writer.Write([]byte("fake-mp3"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer upstream.Close()

	service, repository := newOpenAIProxyTestService(upstream.URL)
	app := fiber.New()
	app.Post("/v1/embeddings", service.Embeddings)
	app.Post("/v1/images/generations", service.ImageGenerations)
	app.Post("/v1/rerank", service.Rerank)
	app.Post("/v1/audio/speech", service.AudioSpeech)

	tests := []struct {
		path        string
		body        string
		contentType string
		wantBody    string
	}{
		{path: "/v1/embeddings", body: `{"model":"public-model","input":["hello"]}`, contentType: fiber.MIMEApplicationJSON, wantBody: `"object":"embedding"`},
		{path: "/v1/images/generations", body: `{"model":"public-model","prompt":"draw"}`, contentType: fiber.MIMEApplicationJSON, wantBody: `"b64_json"`},
		{path: "/v1/rerank", body: `{"model":"public-model","query":"hello","documents":["a","b"]}`, contentType: fiber.MIMEApplicationJSON, wantBody: `"relevance_score"`},
		{path: "/v1/audio/speech", body: `{"model":"public-model","voice":"alloy","input":"hello"}`, contentType: "audio/mpeg", wantBody: "fake-mp3"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set(fiber.HeaderAuthorization, "Bearer sk-test")
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			request.Header.Set(fiber.HeaderAccept, test.contentType)
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			assert.Equal(t, http.StatusOK, response.StatusCode)
			responseBody, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Contains(t, string(responseBody), test.wantBody)
			assert.Contains(t, response.Header.Get(fiber.HeaderContentType), test.contentType)
			assert.NotEmpty(t, response.Header.Get("X-AI-Gateway-Request-ID"))
		})
	}

	assert.ElementsMatch(t, []string{
		"/v1/embeddings", "/v1/images/generations", "/v1/rerank", "/v1/audio/speech",
	}, paths)
	logs := repository.callLogs()
	require.Len(t, logs, 4)
	for _, logRecord := range logs {
		assert.Equal(t, "public-model", logRecord.Model)
		assert.Equal(t, gatewayStatusNormal, logRecord.Success)
	}
	assert.Equal(t, 3, logs[0].PromptTokens)
	assert.Equal(t, 5, logs[1].PromptTokens)
	assert.Equal(t, 7, logs[1].CompletionTokens)
	assert.Equal(t, 1, logs[1].OutputImages)
	assert.Equal(t, 4, logs[2].PromptTokens)
}

func TestOpenAIAudioTranscriptionsPreservesMultipartRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		require.Equal(t, "/v1/audio/transcriptions", request.URL.Path)
		require.Equal(t, "Bearer upstream-key", request.Header.Get("Authorization"))
		require.NoError(t, request.ParseMultipartForm(maxAudioTranscriptionRequestBodyBytes))
		assert.Equal(t, "upstream-model", request.FormValue("model"))
		assert.Equal(t, "zh", request.FormValue("language"))
		file, header, err := request.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "sample.wav", header.Filename)
		content, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, []byte("fake-wav"), content)
		writer.Header().Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		_, _ = fmt.Fprint(writer, `{"text":"hello"}`)
	}))
	defer upstream.Close()

	service, repository := newOpenAIProxyTestService(upstream.URL)
	app := fiber.New(fiber.Config{BodyLimit: maxAudioTranscriptionRequestBodyBytes})
	app.Post("/v1/audio/transcriptions", service.AudioTranscriptions)

	var requestBody bytes.Buffer
	form := multipart.NewWriter(&requestBody)
	require.NoError(t, form.WriteField("model", "public-model"))
	require.NoError(t, form.WriteField("language", "zh"))
	file, err := form.CreateFormFile("file", "sample.wav")
	require.NoError(t, err)
	_, err = file.Write([]byte("fake-wav"))
	require.NoError(t, err)
	require.NoError(t, form.Close())

	request := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", &requestBody)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer sk-test")
	request.Header.Set(fiber.HeaderContentType, form.FormDataContentType())
	response, err := app.Test(request)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"text":"hello"}`, string(responseBody))
	require.Len(t, repository.callLogs(), 1)
	assert.Equal(t, "public-model", repository.callLogs()[0].Model)
}

func TestAudioTranscriptionsRejectsNonMultipartBody(t *testing.T) {
	service := NewGatewayService(GatewayDependencies{})
	app := fiber.New()
	app.Post("/v1/audio/transcriptions", service.AudioTranscriptions)
	request := httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", strings.NewReader(`{"model":"whisper-1"}`))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request)
	require.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusUnsupportedMediaType, response.StatusCode)
}

func TestOpenAIChatCompletionsSupportsStreamingAndNonStreaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		assert.Equal(t, "upstream-model", payload.Model)
		writer.Header().Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
		if !payload.Stream {
			_, _ = fmt.Fprint(writer, `{"id":"chatcmpl-normal","model":"upstream-model","choices":[{"index":0,"message":{"role":"assistant","content":"normal"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`)
			return
		}
		writer.Header().Set(fiber.HeaderContentType, "text/event-stream")
		flusher, ok := writer.(http.Flusher)
		require.True(t, ok)
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"chatcmpl-stream\",\"model\":\"upstream-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"streamed\"},\"finish_reason\":null}]}\n\n")
		flusher.Flush()
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"chatcmpl-stream\",\"model\":\"upstream-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1,\"total_tokens\":3}}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer upstream.Close()

	service, _ := newOpenAIProxyTestService(upstream.URL)
	app := fiber.New()
	app.Post("/v1/chat/completions", service.ChatCompletions)

	for _, test := range []struct {
		name        string
		stream      bool
		want        string
		contentType string
	}{
		{name: "non-streaming", stream: false, want: "normal", contentType: fiber.MIMEApplicationJSON},
		{name: "streaming", stream: true, want: "streamed", contentType: "text/event-stream"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"model":"public-model","messages":[{"role":"user","content":"hello"}],"stream":%t}`, test.stream)
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
			request.Header.Set(fiber.HeaderAuthorization, "Bearer sk-test")
			request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			response, err := app.Test(request, 10_000)
			require.NoError(t, err)
			defer response.Body.Close()
			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Contains(t, response.Header.Get(fiber.HeaderContentType), test.contentType)
			responseBody, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Contains(t, string(responseBody), test.want)
			if test.stream {
				assert.Contains(t, string(responseBody), "data: [DONE]")
			}
		})
	}
}

func TestOpenAIModelsReturnsLogicalModels(t *testing.T) {
	service, _ := newOpenAIProxyTestService("https://api.example.com")
	app := fiber.New()
	app.Get("/v1/models", service.Models)
	for _, test := range []struct {
		name   string
		header string
		value  string
	}{
		{name: "authorization bearer", header: fiber.HeaderAuthorization, value: "Bearer sk-test"},
		{name: "lowercase x-api-key", header: "x-api-key", value: "sk-test"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			request.Header.Set(test.header, test.value)
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			assert.Equal(t, http.StatusOK, response.StatusCode)
			responseBody, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.JSONEq(t, `{"object":"list","data":[{"id":"public-model","object":"model","created":0,"owned_by":"openai"}]}`, string(responseBody))
		})
	}
}
