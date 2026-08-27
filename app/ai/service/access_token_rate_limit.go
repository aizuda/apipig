package service

import (
	"encoding/json"
	"errors"
	"strings"

	"apipig/app/ai/model"
	"apipig/toolkit"
)

type accessTokenRateLimitRule struct {
	Enabled        bool    `json:"enabled"`
	FiveHourAmount float64 `json:"fiveHourAmount"`
	DayAmount      float64 `json:"dayAmount"`
	SevenDayAmount float64 `json:"sevenDayAmount"`
}

func normalizeAccessTokenRateLimitRule(token *model.AccessToken) error {
	rule, err := parseAccessTokenRateLimitRule(token.RateLimitRule)
	if err != nil {
		return err
	}
	normalized, err := json.Marshal(rule)
	if err != nil {
		return errors.New("API 密钥速率限制规则序列化失败")
	}
	token.RateLimitRule = string(normalized)
	return nil
}

func validateAccessTokenRateLimitRule(value string) error {
	rule, err := parseAccessTokenRateLimitRule(value)
	if err != nil {
		return err
	}
	limits := []struct {
		value float64
		name  string
	}{
		{rule.FiveHourAmount, "API 密钥 5 小时限额"},
		{rule.DayAmount, "API 密钥日限额"},
		{rule.SevenDayAmount, "API 密钥 7 天限额"},
	}
	for _, limit := range limits {
		if err := validateUSD(limit.value, limit.name); err != nil {
			return err
		}
	}
	return nil
}

func parseAccessTokenRateLimitRule(value string) (accessTokenRateLimitRule, error) {
	var rule accessTokenRateLimitRule
	value = strings.TrimSpace(value)
	if value == "" {
		return rule, nil
	}
	if err := decodeStrictJSON(value, &rule); err != nil {
		return accessTokenRateLimitRule{}, errors.New("API 密钥速率限制规则必须是有效的 JSON")
	}
	return rule, nil
}

func (s *GatewayService) accessTokenRateLimitMessage(token model.AccessToken) (string, error) {
	rule, err := parseAccessTokenRateLimitRule(token.RateLimitRule)
	if err != nil {
		return "", err
	}
	if !rule.Enabled || (rule.FiveHourAmount <= 0 && rule.DayAmount <= 0 && rule.SevenDayAmount <= 0) {
		return "", nil
	}
	costs, err := s.gatewayRepository().AccessTokenCostWindows(token.ID, toolkit.GetNowUnixMilli())
	if err != nil {
		return "", err
	}
	limits := []struct {
		limit   int64
		used    int64
		message string
	}{
		{usdToMicroUSD(rule.FiveHourAmount), costs.FiveHourMicroUSD, "API 密钥已达到 5 小时消费限额"},
		{usdToMicroUSD(rule.DayAmount), costs.DayMicroUSD, "API 密钥已达到日消费限额"},
		{usdToMicroUSD(rule.SevenDayAmount), costs.SevenDayMicroUSD, "API 密钥已达到 7 天消费限额"},
	}
	for _, limit := range limits {
		if limit.limit > 0 && limit.used >= limit.limit {
			return limit.message, nil
		}
	}
	return "", nil
}
