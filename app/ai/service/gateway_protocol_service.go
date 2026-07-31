package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	"apipig/core/db"

	"github.com/gofiber/fiber/v2"
	"github.com/zendev-sh/goai/provider"
)

type gatewayCall struct {
	Token       model.AccessToken
	Target      routeTarget
	Model       string
	Path        string
	RequestID   string
	StartedAt   time.Time
	InputImages int
}

// ChatCompletions 提供 OpenAI POST /v1/chat/completions 兼容能力。
func (s *GatewayService) ChatCompletions(c *fiber.Ctx) error {
	if len(c.Body()) > maxGatewayRequestBodyBytes {
		return writeGatewayError(c, "openai", fiber.StatusRequestEntityTooLarge, errors.New("请求体超过 8 MiB 限制"))
	}
	req, params, err := parseOpenAIRequest(c.Body())
	if err != nil {
		return writeGatewayError(c, "openai", fiber.StatusBadRequest, err)
	}
	call, status, err := s.prepareGatewayCall(c, req.Model, "/v1/chat/completions")
	if err != nil {
		return writeGatewayError(c, "openai", status, err)
	}
	call.InputImages = countRequestImages(c.Body())
	providerModel, _ := resolveAccountModel(call.Target.Account.Models, call.Model)
	languageModel, err := buildLanguageModel(call.Target, providerModel)
	if err != nil {
		s.finishGatewayCall(call, buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusBadGateway, 0), provider.Usage{}, err)
		return writeGatewayError(c, "openai", fiber.StatusBadGateway, err)
	}
	applyGatewayProviderOptions(call.Target, &params)
	if req.Stream {
		return s.streamOpenAI(c, call, languageModel, params)
	}
	result, err := languageModel.DoGenerate(context.Background(), params)
	logRecord := buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusOK, time.Since(call.StartedAt).Milliseconds())
	if err != nil {
		logRecord.StatusCode = fiber.StatusBadGateway
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return writeGatewayError(c, "openai", fiber.StatusBadGateway, err)
	}
	s.finishGatewayCall(call, logRecord, result.Usage, nil)
	c.Set("X-AI-Gateway-Request-ID", call.RequestID)
	return c.JSON(buildOpenAIResponse(call.Model, result))
}

// AIChat 使用后台选定的 API 密钥执行一次无状态聊天测试。
func (s *GatewayService) AIChat(params *aiReq.AIChatParams) (map[string]any, error) {
	call, languageModel, generateParams, err := s.prepareAIChat(params)
	if err != nil {
		return nil, err
	}
	result, err := languageModel.DoGenerate(context.Background(), generateParams)
	logRecord := buildCallLog(params.Ctx, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusOK, time.Since(call.StartedAt).Milliseconds())
	if err != nil {
		logRecord.StatusCode = fiber.StatusBadGateway
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return nil, errors.New(redactSensitiveText(err.Error(), call.Target.Account.APIKey))
	}
	s.finishGatewayCall(call, logRecord, result.Usage, nil)
	responseBody := buildOpenAIResponse(call.Model, result)
	responseBody["request_id"] = call.RequestID
	return responseBody, nil
}

// AIChatStream 使用后台选定的 API 密钥执行一次 SSE 流式聊天测试。
func (s *GatewayService) AIChatStream(params *aiReq.AIChatParams) error {
	call, languageModel, generateParams, err := s.prepareAIChat(params)
	if err != nil {
		return err
	}
	return s.streamOpenAI(params.Ctx, call, languageModel, generateParams)
}

func (s *GatewayService) prepareAIChat(params *aiReq.AIChatParams) (gatewayCall, provider.LanguageModel, provider.GenerateParams, error) {
	if params == nil || params.Ctx == nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, errors.New("聊天请求上下文不能为空")
	}
	if err := validateAIChatParams(params); err != nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, err
	}
	messages := make([]openAIMessage, 0, len(params.Messages))
	for _, message := range params.Messages {
		content, err := json.Marshal(message.Content)
		if err != nil {
			return gatewayCall{}, nil, provider.GenerateParams{}, err
		}
		messages = append(messages, openAIMessage{Role: message.Role, Content: content})
	}
	requestBody, err := json.Marshal(openAIChatRequest{
		Model: params.Model, Messages: messages, Temperature: params.Temperature, MaxTokens: params.MaxTokens,
	})
	if err != nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, err
	}
	if len(requestBody) > maxGatewayRequestBodyBytes {
		return gatewayCall{}, nil, provider.GenerateParams{}, errors.New("请求体超过 8 MiB 限制")
	}
	req, generateParams, err := parseOpenAIRequest(requestBody)
	if err != nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, err
	}
	token, err := s.gatewayRepository().FindAccessTokenByID(params.TokenID)
	if err != nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, errors.New("所选 API 密钥无效或已禁用")
	}
	if err := validateGatewayTokenAccess(token, params.Ctx.IP()); err != nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, err
	}
	call, _, err := s.prepareGatewayCallWithToken(params.Ctx, token, req.Model, "/v1/chat/completions")
	if err != nil {
		return gatewayCall{}, nil, provider.GenerateParams{}, err
	}
	providerModel, _ := resolveAccountModel(call.Target.Account.Models, call.Model)
	languageModel, err := buildLanguageModel(call.Target, providerModel)
	if err != nil {
		s.finishGatewayCall(call, buildCallLog(params.Ctx, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusBadGateway, 0), provider.Usage{}, err)
		return gatewayCall{}, nil, provider.GenerateParams{}, errors.New(redactSensitiveText(err.Error(), call.Target.Account.APIKey))
	}
	applyGatewayProviderOptions(call.Target, &generateParams)
	return call, languageModel, generateParams, nil
}

func validateAIChatParams(params *aiReq.AIChatParams) error {
	if params == nil {
		return errors.New("聊天请求参数不能为空")
	}
	if params.Temperature != nil && (math.IsNaN(*params.Temperature) || math.IsInf(*params.Temperature, 0) || *params.Temperature < 0 || *params.Temperature > 2) {
		return errors.New("temperature 必须在 0 到 2 之间")
	}
	if params.MaxTokens < 0 || params.MaxTokens > 32768 {
		return errors.New("maxTokens 必须在 0 到 32768 之间")
	}
	return nil
}

// Messages 提供 Anthropic POST /v1/messages 兼容能力。
func (s *GatewayService) Messages(c *fiber.Ctx) error {
	if len(c.Body()) > maxGatewayRequestBodyBytes {
		return writeGatewayError(c, "anthropic", fiber.StatusRequestEntityTooLarge, errors.New("请求体超过 8 MiB 限制"))
	}
	req, params, err := parseAnthropicRequest(c.Body())
	if err != nil {
		return writeGatewayError(c, "anthropic", fiber.StatusBadRequest, err)
	}
	call, status, err := s.prepareGatewayCall(c, req.Model, "/v1/messages")
	if err != nil {
		return writeGatewayError(c, "anthropic", status, err)
	}
	call.InputImages = countRequestImages(c.Body())
	providerModel, _ := resolveAccountModel(call.Target.Account.Models, call.Model)
	languageModel, err := buildLanguageModel(call.Target, providerModel)
	if err != nil {
		s.finishGatewayCall(call, buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusBadGateway, 0), provider.Usage{}, err)
		return writeGatewayError(c, "anthropic", fiber.StatusBadGateway, err)
	}
	if req.Stream {
		return s.streamAnthropic(c, call, languageModel, params)
	}
	result, err := languageModel.DoGenerate(context.Background(), params)
	logRecord := buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusOK, time.Since(call.StartedAt).Milliseconds())
	if err != nil {
		logRecord.StatusCode = fiber.StatusBadGateway
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return writeGatewayError(c, "anthropic", fiber.StatusBadGateway, err)
	}
	s.finishGatewayCall(call, logRecord, result.Usage, nil)
	c.Set("X-AI-Gateway-Request-ID", call.RequestID)
	return c.JSON(buildAnthropicResponse(call.Model, result))
}

// Models 返回访问令牌可见且存在启用渠道的模型。
func (s *GatewayService) Models(c *fiber.Ctx) error {
	token, err := s.authenticateGatewayToken(c)
	if err != nil {
		return writeGatewayError(c, "openai", fiber.StatusUnauthorized, err)
	}
	targets, err := s.loadRouteTargets()
	if err != nil {
		return writeGatewayError(c, "openai", fiber.StatusInternalServerError, err)
	}
	seen := make(map[string]struct{})
	items := make([]map[string]any, 0)
	for _, target := range targets {
		modelIDs, mappingErr := accountGatewayModels(target.Account.Models)
		if mappingErr != nil {
			continue
		}
		if len(modelIDs) == 0 {
			modelIDs = splitModels(target.Provider.Models)
		}
		if len(modelIDs) == 1 && modelIDs[0] == "*" {
			modelIDs = splitModels(token.Models)
		}
		for _, modelID := range modelIDs {
			if modelID == "*" || !containsModel(token.Models, modelID) {
				continue
			}
			if _, ok := seen[modelID]; ok {
				continue
			}
			seen[modelID] = struct{}{}
			items = append(items, map[string]any{
				"id": modelID, "object": "model", "created": target.Provider.CreatedAt / 1000, "owned_by": target.Provider.Code,
			})
		}
	}
	return c.JSON(map[string]any{"object": "list", "data": items})
}

// Healthz 返回应用和数据库健康状态。
func (s *GatewayService) Healthz(c *fiber.Ctx) error {
	if err := s.gatewayRepository().Ping(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]any{"status": "unhealthy", "database": "unavailable"})
	}
	return c.JSON(map[string]any{"status": "ok", "database": "ok"})
}

func (s *GatewayService) prepareGatewayCall(c *fiber.Ctx, modelID, path string) (gatewayCall, int, error) {
	s.ensureRuntimeComponents()
	token, err := s.authenticateGatewayToken(c)
	if err != nil {
		return gatewayCall{}, fiber.StatusUnauthorized, err
	}
	return s.prepareGatewayCallWithToken(c, token, modelID, path)
}

func (s *GatewayService) prepareGatewayCallWithToken(c *fiber.Ctx, token model.AccessToken, modelID, path string) (gatewayCall, int, error) {
	s.ensureRuntimeComponents()
	if !containsModel(token.Models, modelID) {
		return gatewayCall{}, fiber.StatusForbidden, errors.New("访问 Token 未授权该模型")
	}
	if quotaExhausted(token) {
		return gatewayCall{}, fiber.StatusPaymentRequired, errors.New("访问 Token 成本额度已用尽")
	}
	limitMessage, err := s.accessTokenRateLimitMessage(token)
	if err != nil {
		return gatewayCall{}, fiber.StatusServiceUnavailable, errors.New("API 密钥消费限额校验失败")
	}
	if limitMessage != "" {
		return gatewayCall{}, fiber.StatusTooManyRequests, errors.New(limitMessage)
	}
	if !s.allowRate(fmt.Sprintf("token:%d", token.ID), token.RPM, token.TPM, 0) {
		return gatewayCall{}, fiber.StatusTooManyRequests, errors.New("访问 Token 已触发限流")
	}
	target, err := s.pickRoute(modelID)
	if err != nil {
		return gatewayCall{}, fiber.StatusServiceUnavailable, err
	}
	if !s.allowRate(fmt.Sprintf("channel:%d", target.Channel.ID), target.Channel.RPM, target.Channel.TPM, 0) {
		return gatewayCall{}, fiber.StatusTooManyRequests, errors.New("渠道账号已触发限流")
	}
	requestID := strings.TrimSpace(c.Get("X-Request-ID"))
	if requestID == "" {
		requestID = fmt.Sprintf("gw-%d", db.GetId())
	}
	return gatewayCall{Token: token, Target: target, Model: modelID, Path: path, RequestID: requestID, StartedAt: time.Now()}, fiber.StatusOK, nil
}

func (s *GatewayService) finishGatewayCall(call gatewayCall, logRecord model.CallLog, usage provider.Usage, callErr error) {
	logRecord.LatencyMs = time.Since(call.StartedAt).Milliseconds()
	billingUsage := BillingUsage{InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, ReasoningTokens: usage.ReasoningTokens, CacheReadTokens: usage.CacheReadTokens, CacheWriteTokens: usage.CacheWriteTokens, InputImages: call.InputImages}
	totalTokens := billingUsage.InputTokens + billingUsage.OutputTokens + billingUsage.CacheReadTokens + billingUsage.CacheWriteTokens
	s.recordRateTokens(fmt.Sprintf("token:%d", call.Token.ID), totalTokens)
	s.recordRateTokens(fmt.Sprintf("channel:%d", call.Target.Channel.ID), totalTokens)
	if callErr != nil {
		logRecord.Success = gatewayStatusDisabled
		message := redactSensitiveText(callErr.Error(), call.Target.Account.APIKey)
		logRecord.ErrorMessage = string(limitBytes([]byte(message), 1000))
		if logRecord.StatusCode < 400 {
			logRecord.StatusCode = fiber.StatusBadGateway
		}
		s.recordBreakerFailure(call.Target.Channel.ID)
	} else {
		logRecord.Success = gatewayStatusNormal
		if logRecord.StatusCode == 0 {
			logRecord.StatusCode = fiber.StatusOK
		}
		s.recordBreakerSuccess(call.Target.Channel.ID)
	}
	billing := BillingResult{Multiplier: normalizedCostMultiplier(call.Target.Channel.CostMultiplier)}
	if logRecord.Success == gatewayStatusNormal {
		providerModel, _ := resolveAccountModel(call.Target.Account.Models, call.Model)
		billing = calculateModelBilling(call.Target, call.Model, providerModel, billingUsage)
	}
	applyBillingToLog(&logRecord, billingUsage, billing)
	_ = s.saveCallLog(logRecord)
	s.recordAccessTokenUsage(buildAccessTokenUsageRecord(call.Token.ID, logRecord, billingUsage))
}

func (s *GatewayService) streamOpenAI(c *fiber.Ctx, call gatewayCall, languageModel provider.LanguageModel, params provider.GenerateParams) error {
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := languageModel.DoStream(ctx, params)
	if err != nil {
		cancel()
		logRecord := buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusBadGateway, 0)
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return writeGatewayError(c, "openai", fiber.StatusBadGateway, err)
	}
	logRecord := buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusOK, 0)
	c.Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-AI-Gateway-Request-ID", call.RequestID)
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		defer cancel()
		usage := provider.Usage{}
		var streamErr error
		sawFinish := false
		toolIndexes := make(map[string]int)
		nextToolIndex := 0
		writeOpenAIChunk(writer, call, map[string]any{"role": "assistant", "content": ""}, nil, nil)
		for chunk := range stream.Stream {
			switch chunk.Type {
			case provider.ChunkText:
				writeOpenAIChunk(writer, call, map[string]any{"content": chunk.Text}, nil, nil)
			case provider.ChunkReasoning:
				writeOpenAIChunk(writer, call, map[string]any{"reasoning_content": chunk.Text}, nil, nil)
			case provider.ChunkToolCallStreamStart:
				index, ok := toolIndexes[chunk.ToolCallID]
				if !ok {
					index = nextToolIndex
					nextToolIndex++
					toolIndexes[chunk.ToolCallID] = index
				}
				toolCall := map[string]any{"index": index, "id": chunk.ToolCallID, "type": "function", "function": map[string]any{"name": chunk.ToolName, "arguments": chunk.ToolInput}}
				writeOpenAIChunk(writer, call, map[string]any{"tool_calls": []map[string]any{toolCall}}, nil, nil)
			case provider.ChunkToolCall:
				if _, ok := toolIndexes[chunk.ToolCallID]; !ok {
					index := nextToolIndex
					nextToolIndex++
					toolIndexes[chunk.ToolCallID] = index
					toolCall := map[string]any{"index": index, "id": chunk.ToolCallID, "type": "function", "function": map[string]any{"name": chunk.ToolName, "arguments": chunk.ToolInput}}
					writeOpenAIChunk(writer, call, map[string]any{"tool_calls": []map[string]any{toolCall}}, nil, nil)
				}
			case provider.ChunkToolCallDelta:
				index := toolIndexes[chunk.ToolCallID]
				toolCall := map[string]any{"index": index, "function": map[string]any{"arguments": chunk.ToolInput}}
				writeOpenAIChunk(writer, call, map[string]any{"tool_calls": []map[string]any{toolCall}}, nil, nil)
			case provider.ChunkFinish:
				sawFinish = true
				usage = chunk.Usage
				finish := openAIFinishReason(chunk.FinishReason)
				writeOpenAIChunk(writer, call, map[string]any{}, &finish, openAIUsage(usage))
			case provider.ChunkError:
				streamErr = chunk.Error
				writeSSEData(writer, map[string]any{"error": map[string]any{"message": errorText(chunk.Error), "type": "api_error"}})
			}
		}
		if !sawFinish && streamErr == nil {
			streamErr = errors.New("上游流在完成事件前结束")
		}
		_, _ = writer.WriteString("data: [DONE]\n\n")
		_ = writer.Flush()
		s.finishGatewayCall(call, logRecord, usage, streamErr)
	})
	return nil
}

func (s *GatewayService) streamAnthropic(c *fiber.Ctx, call gatewayCall, languageModel provider.LanguageModel, params provider.GenerateParams) error {
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := languageModel.DoStream(ctx, params)
	if err != nil {
		cancel()
		logRecord := buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusBadGateway, 0)
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return writeGatewayError(c, "anthropic", fiber.StatusBadGateway, err)
	}
	logRecord := buildCallLog(c, call.RequestID, call.Token, call.Target, call.Model, call.Path, fiber.StatusOK, 0)
	c.Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-AI-Gateway-Request-ID", call.RequestID)
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		defer cancel()
		usage := provider.Usage{}
		var streamErr error
		sawFinish := false
		textStarted := false
		openBlocks := make(map[int]struct{})
		toolIndexes := make(map[string]int)
		nextIndex := 0
		message := map[string]any{"id": responseID("", "msg"), "type": "message", "role": "assistant", "model": call.Model, "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": anthropicUsage(provider.Usage{})}
		writeAnthropicEvent(writer, "message_start", map[string]any{"type": "message_start", "message": message})
		for chunk := range stream.Stream {
			switch chunk.Type {
			case provider.ChunkText:
				if !textStarted {
					textStarted = true
					openBlocks[0] = struct{}{}
					nextIndex = 1
					writeAnthropicEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
				}
				writeAnthropicEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": chunk.Text}})
			case provider.ChunkToolCallStreamStart:
				index, ok := toolIndexes[chunk.ToolCallID]
				if !ok {
					index = nextIndex
					nextIndex++
					toolIndexes[chunk.ToolCallID] = index
					openBlocks[index] = struct{}{}
					writeAnthropicEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": index, "content_block": map[string]any{"type": "tool_use", "id": chunk.ToolCallID, "name": chunk.ToolName, "input": map[string]any{}}})
				}
				if chunk.ToolInput != "" {
					writeAnthropicEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "input_json_delta", "partial_json": chunk.ToolInput}})
				}
			case provider.ChunkToolCall:
				if _, ok := toolIndexes[chunk.ToolCallID]; !ok {
					index := nextIndex
					nextIndex++
					toolIndexes[chunk.ToolCallID] = index
					openBlocks[index] = struct{}{}
					writeAnthropicEvent(writer, "content_block_start", map[string]any{"type": "content_block_start", "index": index, "content_block": map[string]any{"type": "tool_use", "id": chunk.ToolCallID, "name": chunk.ToolName, "input": map[string]any{}}})
					if chunk.ToolInput != "" {
						writeAnthropicEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "input_json_delta", "partial_json": chunk.ToolInput}})
					}
				}
			case provider.ChunkToolCallDelta:
				index := toolIndexes[chunk.ToolCallID]
				writeAnthropicEvent(writer, "content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "input_json_delta", "partial_json": chunk.ToolInput}})
			case provider.ChunkFinish:
				sawFinish = true
				usage = chunk.Usage
				for index := range openBlocks {
					writeAnthropicEvent(writer, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
				}
				writeAnthropicEvent(writer, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": anthropicStopReason(chunk.FinishReason), "stop_sequence": nil}, "usage": anthropicUsage(usage)})
			case provider.ChunkError:
				streamErr = chunk.Error
				writeAnthropicEvent(writer, "error", map[string]any{"type": "error", "error": map[string]any{"type": "api_error", "message": errorText(chunk.Error)}})
			}
		}
		if !sawFinish && streamErr == nil {
			streamErr = errors.New("上游流在完成事件前结束")
		}
		writeAnthropicEvent(writer, "message_stop", map[string]any{"type": "message_stop"})
		_ = writer.Flush()
		s.finishGatewayCall(call, logRecord, usage, streamErr)
	})
	return nil
}

func writeOpenAIChunk(writer *bufio.Writer, call gatewayCall, delta map[string]any, finishReason *string, usage map[string]any) {
	choice := map[string]any{"index": 0, "delta": delta, "finish_reason": finishReason}
	payload := map[string]any{
		"id": call.RequestID, "object": "chat.completion.chunk", "created": time.Now().Unix(), "model": call.Model,
		"choices": []map[string]any{choice},
	}
	if usage != nil {
		payload["usage"] = usage
	}
	writeSSEData(writer, payload)
}

func writeSSEData(writer *bufio.Writer, payload any) {
	data, _ := json.Marshal(payload)
	_, _ = writer.WriteString("data: ")
	_, _ = writer.Write(data)
	_, _ = writer.WriteString("\n\n")
	_ = writer.Flush()
}

func writeAnthropicEvent(writer *bufio.Writer, event string, payload any) {
	data, _ := json.Marshal(payload)
	_, _ = writer.WriteString("event: " + event + "\n")
	_, _ = writer.WriteString("data: ")
	_, _ = writer.Write(data)
	_, _ = writer.WriteString("\n\n")
	_ = writer.Flush()
}

func writeGatewayError(c *fiber.Ctx, protocolName string, status int, err error) error {
	message := errorText(err)
	if protocolName == "anthropic" {
		errorType := "api_error"
		if status >= 400 && status < 500 {
			errorType = "invalid_request_error"
		}
		return c.Status(status).JSON(map[string]any{"type": "error", "error": map[string]any{"type": errorType, "message": message}})
	}
	errorType := "api_error"
	if status >= 400 && status < 500 {
		errorType = "invalid_request_error"
	}
	return c.Status(status).JSON(map[string]any{"error": map[string]any{"message": message, "type": errorType, "code": nil}})
}

func errorText(err error) string {
	if err == nil {
		return "unknown error"
	}
	return redactSensitiveText(err.Error())
}

func splitModels(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
