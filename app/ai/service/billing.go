package service

import (
	"errors"
	"math"

	"apipig/app/ai/model"
	"apipig/toolkit/snowflake"
)

const microUSDScale int64 = 1_000_000

const (
	maxUSDValue         = 999_999_999_999.0
	maxMicroUSDValue    = int64(maxUSDValue * float64(microUSDScale))
	maxStoredTokenCount = int(^uint32(0) >> 1)
)

// BillingUsage 是跨协议统一后的可计费用量。
type BillingUsage struct {
	InputTokens       int
	OutputTokens      int
	ReasoningTokens   int
	CacheReadTokens   int
	CacheWriteTokens  int
	InputImages       int
	OutputImages      int
	InputImageTokens  int
	OutputImageTokens int
}

// BillingResult 同时保存标准成本和渠道倍率后的有效成本。
type BillingResult struct {
	StandardMicroUSD  int64
	EffectiveMicroUSD int64
	Multiplier        float64
	PricingModel      string
	PricingSnapshot   string
}

type AccessTokenUsageRecord struct {
	TokenID               snowflake.ID
	Success               bool
	InputTokens           int
	OutputTokens          int
	ReasoningTokens       int
	CacheReadTokens       int
	CacheWriteTokens      int
	EffectiveCostMicroUSD int64
	OccurredAt            int64
}

func usdToMicroUSD(value float64) int64 {
	if value <= 0 || math.IsNaN(value) {
		return 0
	}
	if math.IsInf(value, 1) || value >= maxUSDValue {
		return maxMicroUSDValue
	}
	return int64(math.Round(value * float64(microUSDScale)))
}

func multiplyMicroUSD(value int64, multiplier float64) int64 {
	if value <= 0 || multiplier <= 0 || math.IsNaN(multiplier) {
		return 0
	}
	result := float64(value) * multiplier
	if math.IsInf(result, 1) || result >= float64(maxMicroUSDValue) {
		return maxMicroUSDValue
	}
	return int64(math.Round(result))
}

func microUSDToUSD(value int64) float64 {
	if value <= 0 {
		return 0
	}
	return float64(value) / float64(microUSDScale)
}

func validateUSD(value float64, field string) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return errors.New(field + "不能为负数、NaN 或无穷大")
	}
	if value > maxUSDValue {
		return errors.New(field + "超过系统金额上限")
	}
	return nil
}

func normalizedCostMultiplier(value float64) float64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 1
	}
	return value
}

func applyBillingToLog(logRecord *model.CallLog, usage BillingUsage, billing BillingResult) {
	usage = normalizeBillingUsage(usage)
	logRecord.PromptTokens = usage.InputTokens
	logRecord.CompletionTokens = usage.OutputTokens
	logRecord.ReasoningTokens = usage.ReasoningTokens
	logRecord.CacheReadTokens = usage.CacheReadTokens
	logRecord.CacheWriteTokens = usage.CacheWriteTokens
	logRecord.InputImages = usage.InputImages
	logRecord.OutputImages = usage.OutputImages
	logRecord.InputImageTokens = usage.InputImageTokens
	logRecord.OutputImageTokens = usage.OutputImageTokens
	logRecord.TotalTokens = sumStoredTokenCounts(usage.InputTokens, usage.OutputTokens, usage.CacheReadTokens, usage.CacheWriteTokens)
	logRecord.StandardCostMicroUSD = billing.StandardMicroUSD
	logRecord.CostMicroUSD = billing.EffectiveMicroUSD
	logRecord.StandardCost = microUSDToUSD(billing.StandardMicroUSD)
	logRecord.Cost = microUSDToUSD(billing.EffectiveMicroUSD)
	logRecord.CostMultiplier = billing.Multiplier
	logRecord.PricingModel = billing.PricingModel
	logRecord.PricingSnapshot = billing.PricingSnapshot
}

func normalizeBillingUsage(usage BillingUsage) BillingUsage {
	usage.InputTokens = normalizeStoredTokenCount(usage.InputTokens)
	usage.OutputTokens = normalizeStoredTokenCount(usage.OutputTokens)
	usage.ReasoningTokens = normalizeStoredTokenCount(usage.ReasoningTokens)
	usage.CacheReadTokens = normalizeStoredTokenCount(usage.CacheReadTokens)
	usage.CacheWriteTokens = normalizeStoredTokenCount(usage.CacheWriteTokens)
	usage.InputImages = normalizeStoredTokenCount(usage.InputImages)
	usage.OutputImages = normalizeStoredTokenCount(usage.OutputImages)
	usage.InputImageTokens = normalizeStoredTokenCount(usage.InputImageTokens)
	usage.OutputImageTokens = normalizeStoredTokenCount(usage.OutputImageTokens)
	return usage
}

func normalizeStoredTokenCount(value int) int {
	if value <= 0 {
		return 0
	}
	if value > maxStoredTokenCount {
		return maxStoredTokenCount
	}
	return value
}

func sumStoredTokenCounts(values ...int) int {
	total := 0
	for _, value := range values {
		value = normalizeStoredTokenCount(value)
		if value > maxStoredTokenCount-total {
			return maxStoredTokenCount
		}
		total += value
	}
	return total
}

func quotaExhausted(token model.AccessToken) bool {
	limit := token.QuotaMicroUSD
	used := token.UsedMicroUSD
	if limit == 0 && token.QuotaAmount > 0 {
		limit = usdToMicroUSD(token.QuotaAmount)
	}
	if used == 0 && token.UsedAmount > 0 {
		used = usdToMicroUSD(token.UsedAmount)
	}
	return limit > 0 && used >= limit
}

func syncAccessTokenAmounts(token *model.AccessToken) {
	if token.QuotaMicroUSD == 0 && token.QuotaAmount > 0 {
		token.QuotaMicroUSD = usdToMicroUSD(token.QuotaAmount)
	}
	if token.UsedMicroUSD == 0 && token.UsedAmount > 0 {
		token.UsedMicroUSD = usdToMicroUSD(token.UsedAmount)
	}
	token.QuotaAmount = microUSDToUSD(token.QuotaMicroUSD)
	token.UsedAmount = microUSDToUSD(token.UsedMicroUSD)
}
