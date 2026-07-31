package service

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
)

const (
	modelPricingUnitPerMillion = "perMillion"
	modelPricingUnitPerToken   = "perToken"
	legacyPerTokenMaxPrice     = 0.0001
)

type ModelPricingRule struct {
	Model                      string  `json:"model"`
	TokenPriceUnit             string  `json:"tokenPriceUnit,omitempty"`
	InputPricePerMTokens       float64 `json:"inputPricePerMTokens"`
	OutputPricePerMTokens      float64 `json:"outputPricePerMTokens"`
	CacheReadPricePerMTokens   float64 `json:"cacheReadPricePerMTokens"`
	CacheWritePricePerMTokens  float64 `json:"cacheWritePricePerMTokens"`
	InputImagePricePerImage    float64 `json:"inputImagePricePerImage"`
	OutputImagePricePerImage   float64 `json:"outputImagePricePerImage"`
	InputImagePricePerMTokens  float64 `json:"inputImagePricePerMTokens"`
	OutputImagePricePerMTokens float64 `json:"outputImagePricePerMTokens"`
}

func normalizeModelPricing(value string) (string, error) {
	rules := make([]ModelPricingRule, 0)
	if strings.TrimSpace(value) != "" {
		if err := json.Unmarshal([]byte(value), &rules); err != nil {
			return "", errors.New("模型计价配置不是有效 JSON")
		}
	}
	seen := make(map[string]struct{}, len(rules))
	for index := range rules {
		rules[index].Model = strings.TrimSpace(rules[index].Model)
		if rules[index].Model == "" {
			return "", errors.New("计价模型不能为空")
		}
		if _, exists := seen[rules[index].Model]; exists {
			return "", errors.New("计价模型不能重复")
		}
		seen[rules[index].Model] = struct{}{}
		prices := []float64{rules[index].InputPricePerMTokens, rules[index].OutputPricePerMTokens, rules[index].CacheReadPricePerMTokens, rules[index].CacheWritePricePerMTokens, rules[index].InputImagePricePerImage, rules[index].OutputImagePricePerImage, rules[index].InputImagePricePerMTokens, rules[index].OutputImagePricePerMTokens}
		for _, price := range prices {
			if math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
				return "", errors.New("模型计价不能为负数、NaN 或无穷大")
			}
		}
		unit, err := normalizeModelPricingUnit(rules[index])
		if err != nil {
			return "", err
		}
		rules[index].TokenPriceUnit = unit
	}
	normalized, err := json.Marshal(rules)
	return string(normalized), err
}

func parseModelPricing(value string) map[string]ModelPricingRule {
	var rules []ModelPricingRule
	if json.Unmarshal([]byte(value), &rules) != nil {
		return map[string]ModelPricingRule{}
	}
	result := make(map[string]ModelPricingRule, len(rules))
	for _, rule := range rules {
		unit, err := normalizeModelPricingUnit(rule)
		if err != nil {
			continue
		}
		rule.TokenPriceUnit = unit
		result[rule.Model] = rule
	}
	return result
}

func calculateModelBilling(target routeTarget, gatewayModel, providerModel string, usage BillingUsage) BillingResult {
	multiplier := normalizedCostMultiplier(target.Channel.CostMultiplier)
	rule, ok := target.Pricing[providerModel]
	if !ok {
		rule, ok = target.Pricing[gatewayModel]
	}
	if !ok {
		rule, ok = target.Pricing["*"]
	}
	if !ok {
		return BillingResult{Multiplier: multiplier}
	}
	unit, err := normalizeModelPricingUnit(rule)
	if err != nil {
		return BillingResult{Multiplier: multiplier}
	}
	rule.TokenPriceUnit = unit
	tokenPriceDivisor := 1.0
	if rule.TokenPriceUnit == modelPricingUnitPerMillion {
		tokenPriceDivisor = 1_000_000
	}
	standardUSD := float64(usage.InputTokens)*rule.InputPricePerMTokens/tokenPriceDivisor +
		float64(usage.OutputTokens)*rule.OutputPricePerMTokens/tokenPriceDivisor +
		float64(usage.CacheReadTokens)*rule.CacheReadPricePerMTokens/tokenPriceDivisor +
		float64(usage.CacheWriteTokens)*rule.CacheWritePricePerMTokens/tokenPriceDivisor +
		float64(usage.InputImages)*rule.InputImagePricePerImage +
		float64(usage.OutputImages)*rule.OutputImagePricePerImage +
		float64(usage.InputImageTokens)*rule.InputImagePricePerMTokens/tokenPriceDivisor +
		float64(usage.OutputImageTokens)*rule.OutputImagePricePerMTokens/tokenPriceDivisor
	standardMicroUSD := usdToMicroUSD(standardUSD)
	snapshot, _ := json.Marshal(rule)
	return BillingResult{
		StandardMicroUSD: standardMicroUSD, EffectiveMicroUSD: int64(math.Round(float64(standardMicroUSD) * multiplier)),
		Multiplier: multiplier, PricingModel: rule.Model, PricingSnapshot: string(snapshot),
	}
}

func normalizeModelPricingUnit(rule ModelPricingRule) (string, error) {
	switch strings.TrimSpace(rule.TokenPriceUnit) {
	case modelPricingUnitPerMillion:
		return modelPricingUnitPerMillion, nil
	case modelPricingUnitPerToken:
		return modelPricingUnitPerToken, nil
	case "":
		if looksLikeLegacyPerTokenPricing(rule) {
			return modelPricingUnitPerToken, nil
		}
		return modelPricingUnitPerMillion, nil
	default:
		return "", errors.New("Token 计价单位无效")
	}
}

func looksLikeLegacyPerTokenPricing(rule ModelPricingRule) bool {
	prices := []float64{
		rule.InputPricePerMTokens,
		rule.OutputPricePerMTokens,
		rule.CacheReadPricePerMTokens,
		rule.CacheWritePricePerMTokens,
		rule.InputImagePricePerMTokens,
		rule.OutputImagePricePerMTokens,
	}
	foundPositive := false
	for _, price := range prices {
		if price <= 0 {
			continue
		}
		foundPositive = true
		if price > legacyPerTokenMaxPrice {
			return false
		}
	}
	return foundPositive
}
