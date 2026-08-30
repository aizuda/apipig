package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	gatewayStatusNormal   uint = 1
	gatewayStatusDisabled uint = 2
)

// GatewayService 负责 AI 网关的跨表汇总和流量转发编排。
//
// 各业务表的增删改查由独立服务负责；当前服务只处理调用方鉴权、路由、限流、
// 熔断、上游转发、成本回写和汇总统计等需要跨表协作的网关职责。
type GatewayService struct {
	runtimeOnce sync.Once
	limiter     RateLimiter
	breaker     CircuitBreaker
	vault       CredentialVault
	repository  GatewayRepository
	logSink     CallLogSink

	routeMu        sync.RWMutex
	routeRefreshMu sync.Mutex
	routeLoadedAt  time.Time
	routeTargets   []routeTarget
}

type GatewayDependencies struct {
	Limiter    RateLimiter
	Breaker    CircuitBreaker
	Vault      CredentialVault
	Repository GatewayRepository
	LogSink    CallLogSink
}

func NewGatewayService(dependencies GatewayDependencies) *GatewayService {
	return &GatewayService{
		limiter: dependencies.Limiter, breaker: dependencies.Breaker, vault: dependencies.Vault,
		repository: dependencies.Repository, logSink: dependencies.LogSink,
	}
}

func (s *GatewayService) credentialVault() CredentialVault {
	if s.vault == nil {
		s.vault = newAESCredentialVault(func() string { return global.CONFIG.AI.EncryptionKey })
	}
	return s.vault
}

func (s *GatewayService) gatewayRepository() GatewayRepository {
	if s.repository == nil {
		s.repository = newGormGatewayRepository(func() *gorm.DB { return global.DB })
	}
	return s.repository
}

type openAIProxyBody struct {
	Model string `json:"model"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		InputTokens         int `json:"input_tokens"`
		OutputTokens        int `json:"output_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
			ImageTokens  int `json:"image_tokens"`
		} `json:"prompt_tokens_details"`
		InputTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
			ImageTokens  int `json:"image_tokens"`
		} `json:"input_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
			ImageTokens     int `json:"image_tokens"`
		} `json:"completion_tokens_details"`
		OutputTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
			ImageTokens     int `json:"image_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
}

func (s *GatewayService) Summary(params *aiReq.GatewaySummaryParams) (summary aiResp.GatewaySummary, err error) {
	startAt, endAt, err := normalizeGatewaySummaryRange(params, time.Now())
	if err != nil {
		return summary, err
	}
	return s.gatewayRepository().Summary(startAt, endAt)
}

func normalizeGatewaySummaryRange(params *aiReq.GatewaySummaryParams, now time.Time) (int64, int64, error) {
	end := startOfLocalDay(now).Add(24*time.Hour - time.Millisecond)
	start := startOfLocalDay(now).AddDate(0, 0, -6)
	if params != nil && params.StartAt > 0 {
		start = startOfLocalDay(time.UnixMilli(params.StartAt))
	}
	if params != nil && params.EndAt > 0 {
		end = startOfLocalDay(time.UnixMilli(params.EndAt)).Add(24*time.Hour - time.Millisecond)
	}
	if end.Before(start) {
		return 0, 0, errors.New("结束时间不能早于开始时间")
	}
	if int(end.Sub(start).Hours()/24)+1 > 90 {
		return 0, 0, errors.New("概览统计时间范围不能超过 90 天")
	}
	return start.UnixMilli(), end.UnixMilli(), nil
}

// ProxyOpenAI 转发 OpenAI 兼容请求。
//
// 请求链路：
// 1. 解析调用方 Token 并校验状态、模型授权、额度。
// 2. 根据模型选择可用渠道，并跳过处于熔断窗口内的渠道。
// 3. 按 Token 和渠道执行分钟级限流。
// 4. 构造上游请求并透传响应，同时记录审计日志和成本。
func (s *GatewayService) ProxyOpenAI(params *aiReq.GatewayProxyParams) error {
	s.ensureRuntimeComponents()
	if params == nil || params.Ctx == nil {
		return errors.New("代理请求上下文不能为空")
	}
	c := params.Ctx
	start := time.Now()
	body := append([]byte(nil), params.RawBody...)
	maxBodyBytes := params.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = maxGatewayRequestBodyBytes
	}
	if len(body) > maxBodyBytes {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "请求体超过端点大小限制")
	}
	if err := validateGatewayUpstream(params.Upstream); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	requestID := gatewayRequestID(c)

	token, err := s.authenticateGatewayToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	contentType := c.Get(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	modelName, err := parseRequestModelByContentType(body, contentType)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if !containsModel(token.Models, modelName) {
		return fiber.NewError(fiber.StatusForbidden, "访问 Token 未授权该模型")
	}
	if quotaExhausted(token) {
		return fiber.NewError(fiber.StatusPaymentRequired, "访问 Token 成本额度已用尽")
	}
	limitMessage, err := s.accessTokenRateLimitMessage(token)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "API 密钥消费限额校验失败")
	}
	if limitMessage != "" {
		return fiber.NewError(fiber.StatusTooManyRequests, limitMessage)
	}
	if !s.allowRate(fmt.Sprintf("token:%d", token.ID), token.RPM, token.TPM, 0) {
		return fiber.NewError(fiber.StatusTooManyRequests, "访问 Token 已触发限流")
	}

	target, err := s.pickRoute(modelName)
	if err != nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
	}
	if !s.allowRate(fmt.Sprintf("channel:%d", target.Channel.ID), target.Channel.RPM, target.Channel.TPM, 0) {
		return fiber.NewError(fiber.StatusTooManyRequests, "渠道账号已触发限流")
	}
	providerModel, _ := resolveAccountModel(target.Account.Models, modelName)
	if providerModel != modelName {
		body, contentType, err = replaceRequestModelByContentType(body, contentType, providerModel)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "请求体模型映射失败")
		}
	}
	upstream := params.Upstream
	qwenTranscriptionResponseFormat := ""
	if isQwenTranscriptionTarget(target, upstream) {
		upstream, body, contentType, qwenTranscriptionResponseFormat, err = adaptQwenTranscriptionRequest(body, contentType)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	statusCode, respBody, headers, err := s.forwardToUpstream(c, target, upstream, body, contentType)
	usageResponseBody := respBody
	if err == nil && statusCode < 400 && qwenTranscriptionResponseFormat != "" {
		respBody, headers, err = adaptQwenTranscriptionResponse(respBody, headers, qwenTranscriptionResponseFormat)
	}
	latency := time.Since(start).Milliseconds()

	logRecord := buildCallLog(c, requestID, token, target, modelName, params.Upstream, statusCode, latency)
	if err != nil {
		safeError := redactSensitiveText(err.Error(), target.Account.APIKey, proxyPassword(target.Proxy))
		logRecord.StatusCode = fiber.StatusBadGateway
		logRecord.Success = gatewayStatusDisabled
		logRecord.ErrorMessage = truncateUTF8(safeError, 1000)
		applyBillingToLog(&logRecord, BillingUsage{}, BillingResult{Multiplier: normalizedCostMultiplier(target.Channel.CostMultiplier)})
		_ = s.saveCallLog(logRecord)
		s.recordAccessTokenUsage(buildAccessTokenUsageRecord(token.ID, logRecord, BillingUsage{}))
		s.recordBreakerFailure(target.Channel.ID)
		return fiber.NewError(fiber.StatusBadGateway, safeError)
	}

	usage := parseUsage(usageResponseBody)
	usage.InputImages = countRequestImages(body)
	usage.OutputImages = countResponseImages(respBody)
	usage = normalizeBillingUsage(usage)
	totalTokens := sumStoredTokenCounts(usage.InputTokens, usage.OutputTokens, usage.CacheReadTokens, usage.CacheWriteTokens)
	s.recordRateTokens(fmt.Sprintf("token:%d", token.ID), totalTokens)
	s.recordRateTokens(fmt.Sprintf("channel:%d", target.Channel.ID), totalTokens)
	if statusCode >= 400 {
		logRecord.Success = gatewayStatusDisabled
		logRecord.ErrorMessage = truncateUTF8(
			redactSensitiveText(string(respBody), target.Account.APIKey, proxyPassword(target.Proxy)),
			1000,
		)
		s.recordBreakerFailure(target.Channel.ID)
	} else {
		logRecord.Success = gatewayStatusNormal
		s.recordBreakerSuccess(target.Channel.ID)
	}
	billing := BillingResult{Multiplier: normalizedCostMultiplier(target.Channel.CostMultiplier)}
	if logRecord.Success == gatewayStatusNormal {
		billing = calculateModelBilling(target, modelName, providerModel, usage)
	}
	applyBillingToLog(&logRecord, usage, billing)
	_ = s.saveCallLog(logRecord)
	s.recordAccessTokenUsage(buildAccessTokenUsageRecord(token.ID, logRecord, usage))

	for key, values := range headers {
		if isBlockedUpstreamResponseHeader(key) {
			continue
		}
		for index, value := range values {
			if index == 0 {
				c.Set(key, value)
				continue
			}
			c.Append(key, value)
		}
	}
	c.Set("X-AI-Gateway-Request-ID", requestID)
	c.Status(statusCode)
	return c.Send(respBody)
}

func (s *GatewayService) authenticateGatewayToken(c *fiber.Ctx) (token model.AccessToken, err error) {
	raw := strings.TrimSpace(c.Get("Authorization"))
	if raw != "" {
		if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			return token, errors.New("Authorization 仅支持 Bearer Token")
		}
		raw = strings.TrimSpace(raw[7:])
	}
	if raw == "" {
		raw = strings.TrimSpace(c.Get("X-API-Key"))
	}
	if raw == "" {
		err = errors.New("缺少网关访问 Token")
		return
	}
	candidates, candidateErr := gatewayTokenLookupCandidates(raw)
	if candidateErr != nil {
		return token, errors.New("网关访问 Token 无效或已禁用")
	}
	token, err = s.gatewayRepository().FindAccessToken(candidates)
	if err != nil {
		err = errors.New("网关访问 Token 无效或已禁用")
		return
	}

	err = validateGatewayTokenAccess(token, c.IP())
	return
}

func validateGatewayTokenAccess(token model.AccessToken, clientIP string) error {
	now := toolkit.GetNowUnixMilli()
	if token.ExpireAt > 0 && token.ExpireAt < now {
		return errors.New("网关访问 Token 已过期")
	}
	if !accessTokenAllowsIP(token, clientIP) {
		return errors.New("当前 IP 不允许使用此 API 密钥")
	}
	return nil
}

func (s *GatewayService) forwardToUpstream(c *fiber.Ctx, target routeTarget, upstream string, body []byte, contentType string) (int, []byte, http.Header, error) {
	baseURL := strings.TrimRight(target.Provider.BaseURL, "/")
	targetURL := baseURL + "/" + strings.TrimLeft(upstream, "/")
	timeout := time.Duration(target.Provider.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, c.Method(), targetURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+target.Account.APIKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", c.Get("Accept", "application/json"))
	req.Header.Set("X-AI-Gateway-Channel", target.Channel.Name)

	client := &http.Client{
		Timeout: timeout,
		// 网关不自动跟随上游重定向，避免配置错误把服务端请求引向意外地址。
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	if target.Proxy != nil {
		transport, err := buildProxyTransport(*target.Proxy)
		if err != nil {
			return 0, nil, nil, err
		}
		client.Transport = transport
		defer transport.CloseIdleConnections()
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxGatewayResponseBodyBytes+1))
	if err != nil {
		return resp.StatusCode, nil, resp.Header, err
	}
	if len(respBody) > maxGatewayResponseBodyBytes {
		return resp.StatusCode, nil, resp.Header, errors.New("上游响应体超过 32 MiB 限制")
	}
	return resp.StatusCode, respBody, resp.Header, nil
}

func (s *GatewayService) saveCallLog(logRecord model.CallLog) error {
	if s.logSink == nil {
		return s.gatewayRepository().CreateCallLogs([]model.CallLog{logRecord})
	}
	return s.logSink.Enqueue(logRecord)
}

func replaceRequestModel(body []byte, modelName string) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	encodedModel, err := json.Marshal(modelName)
	if err != nil {
		return nil, err
	}
	payload["model"] = encodedModel
	return json.Marshal(payload)
}

func buildCallLog(c *fiber.Ctx, requestID string, token model.AccessToken, target routeTarget, modelName, upstream string, statusCode int, latencyMs int64) model.CallLog {
	return model.CallLog{
		RequestID:     requestID,
		AccessTokenID: token.ID,
		ProviderID:    target.Provider.ID,
		ChannelID:     target.Channel.ID,
		Model:         modelName,
		Path:          upstream,
		Method:        c.Method(),
		ClientIP:      c.IP(),
		StatusCode:    statusCode,
		LatencyMs:     latencyMs,
	}
}

func parseUsage(body []byte) BillingUsage {
	var payload openAIProxyBody
	_ = json.Unmarshal(body, &payload)
	cacheReadTokens := payload.Usage.PromptTokensDetails.CachedTokens
	if cacheReadTokens == 0 {
		cacheReadTokens = payload.Usage.InputTokensDetails.CachedTokens
	}
	inputImageTokens := payload.Usage.PromptTokensDetails.ImageTokens
	if inputImageTokens == 0 {
		inputImageTokens = payload.Usage.InputTokensDetails.ImageTokens
	}
	outputImageTokens := payload.Usage.CompletionTokensDetails.ImageTokens
	if outputImageTokens == 0 {
		outputImageTokens = payload.Usage.OutputTokensDetails.ImageTokens
	}
	inputTokenTotal := payload.Usage.PromptTokens
	if inputTokenTotal == 0 {
		inputTokenTotal = payload.Usage.InputTokens
	}
	outputTokenTotal := payload.Usage.CompletionTokens
	if outputTokenTotal == 0 {
		outputTokenTotal = payload.Usage.OutputTokens
	}
	if inputTokenTotal == 0 && outputTokenTotal == 0 {
		inputTokenTotal = payload.Usage.TotalTokens
	}
	inputTokens := inputTokenTotal - cacheReadTokens - inputImageTokens
	if inputTokens < 0 {
		inputTokens = 0
	}
	outputTokens := outputTokenTotal - outputImageTokens
	if outputTokens < 0 {
		outputTokens = 0
	}
	reasoningTokens := payload.Usage.CompletionTokensDetails.ReasoningTokens
	if reasoningTokens == 0 {
		reasoningTokens = payload.Usage.OutputTokensDetails.ReasoningTokens
	}
	return BillingUsage{
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
		ReasoningTokens:   reasoningTokens,
		CacheReadTokens:   cacheReadTokens,
		InputImageTokens:  inputImageTokens,
		OutputImageTokens: outputImageTokens,
	}
}

func buildAccessTokenUsageRecord(tokenID snowflake.ID, logRecord model.CallLog, usage BillingUsage) AccessTokenUsageRecord {
	usage = normalizeBillingUsage(usage)
	return AccessTokenUsageRecord{
		TokenID:               tokenID,
		Success:               logRecord.Success == gatewayStatusNormal,
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		ReasoningTokens:       usage.ReasoningTokens,
		CacheReadTokens:       usage.CacheReadTokens,
		CacheWriteTokens:      usage.CacheWriteTokens,
		EffectiveCostMicroUSD: logRecord.CostMicroUSD,
		OccurredAt:            toolkit.GetNowUnixMilli(),
	}
}

func (s *GatewayService) recordAccessTokenUsage(record AccessTokenUsageRecord) {
	if err := s.gatewayRepository().RecordAccessTokenUsage(record); err != nil && global.LOG != nil {
		global.LOG.Error("AI 计费用量写入失败", zap.String("accessTokenId", record.TokenID.String()), zap.Error(err))
	}
}

func truncateUTF8(value string, maxBytes int) string {
	value = strings.ToValidUTF8(value, "�")
	if maxBytes <= 0 {
		return ""
	}
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func buildProxyTransport(proxy model.Proxy) (*http.Transport, error) {
	proxyURL := &url.URL{
		Scheme: proxy.Scheme,
		Host:   net.JoinHostPort(proxy.Host, fmt.Sprintf("%d", proxy.Port)),
	}
	if proxy.Username != "" {
		proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	return transport, nil
}

func proxyPassword(proxy *model.Proxy) string {
	if proxy == nil {
		return ""
	}
	return proxy.Password
}

func gatewayRequestID(c *fiber.Ctx) string {
	requestID := strings.TrimSpace(c.Get("X-Request-ID"))
	if requestID != "" && len(requestID) <= 80 && !strings.ContainsAny(requestID, "\r\n") {
		return requestID
	}
	return fmt.Sprintf("gw-%d", db.GetId())
}

func validateGatewayUpstream(upstream string) error {
	if upstream == "" || len(upstream) > 2048 || strings.ContainsAny(upstream, "\r\n#") {
		return errors.New("上游路径无效或过长")
	}
	parsed, err := url.ParseRequestURI(upstream)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") {
		return errors.New("上游路径格式无效")
	}
	decodedPath, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return errors.New("上游路径编码无效")
	}
	for _, segment := range strings.Split(decodedPath, "/") {
		if segment == ".." {
			return errors.New("上游路径不允许包含上级目录")
		}
	}
	return nil
}

func isBlockedUpstreamResponseHeader(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Connection", "Content-Length", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
		"Set-Cookie", "Te", "Trailer", "Transfer-Encoding", "Upgrade":
		return true
	default:
		return false
	}
}
