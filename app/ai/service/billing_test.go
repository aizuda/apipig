package service

import (
	"testing"

	"apipig/app/ai/model"
	"apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCalculateBillingUsesChannelModelPricing(t *testing.T) {
	target := routeTarget{Channel: model.Channel{CostMultiplier: 1.5}, Pricing: map[string]ModelPricingRule{"gpt-test": {Model: "gpt-test", InputPricePerMTokens: 2, OutputPricePerMTokens: 8, InputImagePricePerImage: 0.01}}}
	result := calculateModelBilling(target, "chat", "gpt-test", BillingUsage{InputTokens: 1000, OutputTokens: 500, InputImages: 2})

	assert.Equal(t, int64(26000), result.StandardMicroUSD)
	assert.Equal(t, int64(39000), result.EffectiveMicroUSD)
	assert.Equal(t, 1.5, result.Multiplier)
	assert.Equal(t, "gpt-test", result.PricingModel)
}

func TestCalculateBillingSupportsPerTokenPricing(t *testing.T) {
	target := routeTarget{Channel: model.Channel{CostMultiplier: 1}, Pricing: map[string]ModelPricingRule{
		"deepseek-v4-flash": {
			Model:                    "deepseek-v4-flash",
			TokenPriceUnit:           modelPricingUnitPerToken,
			InputPricePerMTokens:     0.000001,
			OutputPricePerMTokens:    0.000002,
			CacheReadPricePerMTokens: 0.0000002,
		},
	}}

	result := calculateModelBilling(target, "ds-v4-flash", "deepseek-v4-flash", BillingUsage{InputTokens: 822, OutputTokens: 1028})

	assert.Equal(t, int64(2878), result.StandardMicroUSD)
	assert.Equal(t, int64(2878), result.EffectiveMicroUSD)
	assert.Contains(t, result.PricingSnapshot, `"tokenPriceUnit":"perToken"`)
}

func TestParseModelPricingInfersLegacyPerTokenUnit(t *testing.T) {
	pricing := parseModelPricing(`[{"model":"deepseek-v4-flash","inputPricePerMTokens":0.000001,"outputPricePerMTokens":0.000002}]`)

	rule := pricing["deepseek-v4-flash"]
	assert.Equal(t, modelPricingUnitPerToken, rule.TokenPriceUnit)
}

func TestNormalizeModelPricingPersistsInferredUnit(t *testing.T) {
	normalized, err := normalizeModelPricing(`[{"model":"deepseek-v4-flash","inputPricePerMTokens":0.000001}]`)

	require.NoError(t, err)
	assert.Contains(t, normalized, `"tokenPriceUnit":"perToken"`)
}

func TestBackfillLegacyPerTokenBilling(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:billing-backfill?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.AccessToken{}, &model.CallLog{}))
	tokenID := snowflake.ID(2001)
	require.NoError(t, database.Create(&model.AccessToken{MODEL: api.MODEL{ID: tokenID}, Name: "legacy", Token: "hash"}).Error)
	require.NoError(t, database.Create(&model.CallLog{
		MODEL:                api.MODEL{ID: snowflake.ID(2002), CreatedAt: 123},
		RequestID:            "legacy-zero-cost",
		AccessTokenID:        tokenID,
		Success:              gatewayStatusNormal,
		PromptTokens:         822,
		CompletionTokens:     1028,
		PricingSnapshot:      `{"model":"deepseek-v4-flash","inputPricePerMTokens":0.000001,"outputPricePerMTokens":0.000002}`,
		CostMultiplier:       1,
		CostMicroUSD:         0,
		StandardCostMicroUSD: 0,
	}).Error)

	require.NoError(t, BackfillLegacyPerTokenBilling(database))

	var logRecord model.CallLog
	require.NoError(t, database.Where("request_id = ?", "legacy-zero-cost").First(&logRecord).Error)
	assert.Equal(t, int64(2878), logRecord.StandardCostMicroUSD)
	assert.Equal(t, int64(2878), logRecord.CostMicroUSD)
	assert.Contains(t, logRecord.PricingSnapshot, `"tokenPriceUnit":"perToken"`)
	var token model.AccessToken
	require.NoError(t, database.First(&token, tokenID).Error)
	assert.Equal(t, int64(2878), token.UsedMicroUSD)
	assert.InDelta(t, 0.002878, token.UsedAmount, 0.0000001)
}

func TestRecordAccessTokenUsageUsesIntegerLedger(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:billing-ledger?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.AccessToken{}))
	tokenID := snowflake.ID(1001)
	require.NoError(t, database.Create(&model.AccessToken{MODEL: api.MODEL{ID: tokenID}, Name: "billing", Token: "hash"}).Error)
	repository := newGormGatewayRepository(func() *gorm.DB { return database })

	require.NoError(t, repository.RecordAccessTokenUsage(AccessTokenUsageRecord{
		TokenID: tokenID, Success: true, InputTokens: 100, OutputTokens: 20,
		ReasoningTokens: 5, CacheReadTokens: 30, CacheWriteTokens: 10,
		EffectiveCostMicroUSD: 12345, OccurredAt: 456,
	}))
	require.NoError(t, repository.RecordAccessTokenUsage(AccessTokenUsageRecord{TokenID: tokenID, Success: false, OccurredAt: 789}))

	var token model.AccessToken
	require.NoError(t, database.First(&token, tokenID).Error)
	assert.Equal(t, int64(1), token.SuccessCount)
	assert.Equal(t, int64(1), token.FailureCount)
	assert.Equal(t, int64(100), token.PromptTokensTotal)
	assert.Equal(t, int64(20), token.CompletionTokensTotal)
	assert.Equal(t, int64(5), token.ReasoningTokensTotal)
	assert.Equal(t, int64(30), token.CacheReadTokensTotal)
	assert.Equal(t, int64(10), token.CacheWriteTokensTotal)
	assert.Equal(t, int64(12345), token.UsedMicroUSD)
	assert.InDelta(t, 0.012345, token.UsedAmount, 0.0000001)
	assert.Equal(t, int64(789), token.LastUsedAt)
}
