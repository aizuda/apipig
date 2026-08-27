package service

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/compat"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/openai"
	"github.com/zendev-sh/goai/provider/xai"
)

// buildLanguageModel 根据供应商协议创建统一的 GoAI 模型实现。
func buildLanguageModel(target routeTarget, modelID string) (provider.LanguageModel, error) {
	protocol := strings.ToLower(strings.TrimSpace(target.Provider.Protocol))
	baseURL := normalizeProviderBaseURL(protocol, target.Provider.BaseURL)
	if baseURL == "" {
		return nil, errors.New("供应商 BaseURL 不能为空")
	}
	httpClient, err := buildGatewayHTTPClient(target)
	if err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(target.Account.APIKey)
	switch protocol {
	case "anthropic", "claude", "authropic":
		baseURL = strings.TrimSuffix(baseURL, "/v1")
		return anthropic.Chat(modelID,
			anthropic.WithAPIKey(apiKey),
			anthropic.WithBaseURL(baseURL),
			anthropic.WithHTTPClient(httpClient),
		), nil
	case "gemini", "google":
		baseURL = strings.TrimSuffix(baseURL, "/v1beta")
		return google.Chat(modelID,
			google.WithAPIKey(apiKey),
			google.WithBaseURL(baseURL),
			google.WithHTTPClient(httpClient),
		), nil
	case "grok", "xai":
		return xai.Chat(modelID,
			xai.WithAPIKey(apiKey),
			xai.WithBaseURL(baseURL),
			xai.WithHTTPClient(httpClient),
		), nil
	case "openai", "codex":
		return openai.Chat(modelID,
			openai.WithAPIKey(apiKey),
			openai.WithBaseURL(baseURL),
			openai.WithHTTPClient(httpClient),
		), nil
	case "qwen", "dashscope":
		// Qwen ASR's OpenAI-compatible endpoint is a standard
		// /chat/completions endpoint. Keep the configured base URL intact so
		// callers can use DashScope's /compatible-mode/v1 path.
		return compat.Chat(modelID,
			compat.WithProviderID(protocol),
			compat.WithAPIKey(apiKey),
			compat.WithBaseURL(baseURL),
			compat.WithHTTPClient(httpClient),
		), nil
	case "custom", "openai-compatible", "compat", "":
		return compat.Chat(modelID,
			compat.WithProviderID(defaultString(protocol, "compat")),
			compat.WithAPIKey(apiKey),
			compat.WithBaseURL(baseURL),
			compat.WithHTTPClient(httpClient),
		), nil
	default:
		return compat.Chat(modelID,
			compat.WithProviderID(protocol),
			compat.WithAPIKey(apiKey),
			compat.WithBaseURL(baseURL),
			compat.WithHTTPClient(httpClient),
		), nil
	}
}

func normalizeProviderBaseURL(protocol, value string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(value), "/")
	if baseURL == "" {
		return ""
	}
	if !isVersionedOpenAIProtocol(protocol) {
		return baseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Path != "" && parsed.Path != "/") {
		return baseURL
	}
	return baseURL + "/v1"
}

func isVersionedOpenAIProtocol(protocol string) bool {
	switch protocol {
	case "openai", "codex", "grok", "xai":
		return true
	default:
		return false
	}
}

func applyGatewayProviderOptions(target routeTarget, params *provider.GenerateParams) {
	if params == nil || !strings.EqualFold(strings.TrimSpace(target.Provider.Protocol), "openai") {
		return
	}
	if params.ProviderOptions == nil {
		params.ProviderOptions = make(map[string]any)
	}
	params.ProviderOptions["useResponsesAPI"] = false
}

func buildGatewayHTTPClient(target routeTarget) (*http.Client, error) {
	timeout := time.Duration(target.Provider.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	var transport http.RoundTripper = http.DefaultTransport
	if target.Proxy != nil {
		proxyTransport, err := buildProxyTransport(*target.Proxy)
		if err != nil {
			return nil, err
		}
		// 模型对象按请求创建，关闭连接复用可避免独立代理 Transport 遗留空闲连接。
		proxyTransport.DisableKeepAlives = true
		transport = proxyTransport
	}
	client.Transport = &responseLimitTransport{base: transport, maxBytes: maxGatewayResponseBodyBytes}
	return client, nil
}

type responseLimitTransport struct {
	base     http.RoundTripper
	maxBytes int64
}

func (t *responseLimitTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response.ContentLength > t.maxBytes {
		_ = response.Body.Close()
		return nil, errors.New("上游响应体超过 32 MiB 限制")
	}
	response.Body = &limitedResponseBody{ReadCloser: response.Body, remaining: t.maxBytes}
	return response, nil
}

type limitedResponseBody struct {
	io.ReadCloser
	remaining int64
}

func (r *limitedResponseBody) Read(buffer []byte) (int, error) {
	if r.remaining <= 0 {
		var probe [1]byte
		count, err := r.ReadCloser.Read(probe[:])
		if count > 0 {
			return 0, errors.New("上游响应体超过 32 MiB 限制")
		}
		return 0, err
	}
	if int64(len(buffer)) > r.remaining {
		buffer = buffer[:r.remaining]
	}
	count, err := r.ReadCloser.Read(buffer)
	r.remaining -= int64(count)
	return count, err
}
