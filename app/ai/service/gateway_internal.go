package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"apipig/app/ai/model"
	"apipig/core/db"
	"apipig/toolkit/snowflake"

	"github.com/zendev-sh/goai/provider"
)

// InternalGenerateParams 定义后台任务通过现有 AI 网关执行模型调用的参数。
type InternalGenerateParams struct {
	Context       context.Context
	AccessTokenID snowflake.ID
	Model         string
	SystemPrompt  string
	UserPrompt    string
	MaxTokens     int
	Path          string
}

// GenerateInternal 让可信后台任务复用渠道路由、限流、熔断、计费和调用日志能力。
func (s *GatewayService) GenerateInternal(params InternalGenerateParams) (string, error) {
	ctx := params.Context
	if ctx == nil {
		ctx = context.Background()
	}
	modelName := strings.TrimSpace(params.Model)
	if params.AccessTokenID == 0 || modelName == "" || strings.TrimSpace(params.UserPrompt) == "" {
		return "", errors.New("内部 AI 调用参数不完整")
	}
	s.ensureRuntimeComponents()
	token, err := s.gatewayRepository().FindAccessTokenByID(params.AccessTokenID)
	if err != nil {
		return "", errors.New("所选 API 密钥无效或已禁用")
	}
	if err := validateGatewayTokenAccess(token, ""); err != nil && !strings.Contains(err.Error(), "IP") {
		return "", err
	}
	if !containsModel(token.Models, modelName) {
		return "", errors.New("访问 Token 未授权该模型")
	}
	if quotaExhausted(token) {
		return "", errors.New("访问 Token 成本额度已用尽")
	}
	if message, err := s.accessTokenRateLimitMessage(token); err != nil {
		return "", errors.New("API 密钥消费限额校验失败")
	} else if message != "" {
		return "", errors.New(message)
	}
	if !s.allowRate(fmt.Sprintf("token:%d", token.ID), token.RPM, token.TPM, 0) {
		return "", errors.New("访问 Token 已触发限流")
	}
	target, err := s.pickRoute(modelName)
	if err != nil {
		return "", err
	}
	if !s.allowRate(fmt.Sprintf("channel:%d", target.Channel.ID), target.Channel.RPM, target.Channel.TPM, 0) {
		return "", errors.New("渠道账号已触发限流")
	}
	providerModel, _ := resolveAccountModel(target.Account.Models, modelName)
	languageModel, err := buildLanguageModel(target, providerModel)
	call := gatewayCall{Token: token, Target: target, Model: modelName, Path: defaultString(params.Path, "/internal/ai-applications/code-review"), RequestID: fmt.Sprintf("review-%d", db.GetId()), StartedAt: time.Now()}
	logRecord := model.CallLog{RequestID: call.RequestID, AccessTokenID: token.ID, ProviderID: target.Provider.ID, ChannelID: target.Channel.ID, Model: modelName, Path: call.Path, Method: "INTERNAL", ClientIP: "127.0.0.1", StatusCode: 200}
	if err != nil {
		logRecord.StatusCode = 502
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return "", err
	}
	generateParams := provider.GenerateParams{
		System:          params.SystemPrompt,
		Messages:        []provider.Message{{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartText, Text: params.UserPrompt}}}},
		MaxOutputTokens: params.MaxTokens,
	}
	applyGatewayProviderOptions(target, &generateParams)
	result, err := languageModel.DoGenerate(ctx, generateParams)
	if err != nil {
		logRecord.StatusCode = 502
		s.finishGatewayCall(call, logRecord, provider.Usage{}, err)
		return "", errors.New(redactSensitiveText(err.Error(), target.Account.APIKey))
	}
	s.finishGatewayCall(call, logRecord, result.Usage, nil)
	return strings.TrimSpace(result.Text), nil
}
