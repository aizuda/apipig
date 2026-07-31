package service

import (
	"errors"
	"testing"
	"time"

	"apipig/app/ai/model"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeAndValidateAccessTokenRateLimitRule(t *testing.T) {
	token := model.AccessToken{RateLimitRule: `{
		"enabled": true,
		"fiveHourAmount": 1.25,
		"dayAmount": 5,
		"sevenDayAmount": 20
	}`}

	require.NoError(t, normalizeAccessTokenRateLimitRule(&token))
	require.NoError(t, validateAccessTokenRateLimitRule(token.RateLimitRule))
	assert.JSONEq(t, `{"enabled":true,"fiveHourAmount":1.25,"dayAmount":5,"sevenDayAmount":20}`, token.RateLimitRule)
}

func TestValidateAccessTokenRateLimitRuleRejectsInvalidValues(t *testing.T) {
	require.Error(t, validateAccessTokenRateLimitRule(`{"enabled":`))
	require.EqualError(t, validateAccessTokenRateLimitRule(
		`{"enabled":true,"fiveHourAmount":-1,"dayAmount":0,"sevenDayAmount":0}`,
	), "API 密钥 5 小时限额不能为负数、NaN 或无穷大")
	require.EqualError(t, validateAccessToken(&model.AccessToken{Name: "invalid expiry", RPM: 60, ExpireAt: -1}, nil), "API 密钥过期时间不能为负数")
}

func TestAccessTokenRateLimitMessage(t *testing.T) {
	token := model.AccessToken{
		MODEL:         coreAPI.MODEL{ID: snowflake.ID(4101)},
		RateLimitRule: `{"enabled":true,"fiveHourAmount":1,"dayAmount":5,"sevenDayAmount":10}`,
	}
	tests := []struct {
		name    string
		costs   accessTokenCostWindows
		err     error
		message string
	}{
		{name: "under limits", costs: accessTokenCostWindows{FiveHourMicroUSD: 999999}},
		{name: "five hour reached", costs: accessTokenCostWindows{FiveHourMicroUSD: 1000000}, message: "API 密钥已达到 5 小时消费限额"},
		{name: "day reached", costs: accessTokenCostWindows{DayMicroUSD: 5000000}, message: "API 密钥已达到日消费限额"},
		{name: "seven day reached", costs: accessTokenCostWindows{SevenDayMicroUSD: 10000000}, message: "API 密钥已达到 7 天消费限额"},
		{name: "repository error", err: errors.New("database unavailable")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeGatewayRepository{costWindows: test.costs, costWindowErr: test.err}
			gatewayService := NewGatewayService(GatewayDependencies{Repository: repository})
			message, err := gatewayService.accessTokenRateLimitMessage(token)
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.message, message)
		})
	}
}

func TestAccessTokenCostWindowsAggregatesRollingPeriods(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:access-token-cost-windows?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.CallLog{}))

	now := time.Now().UnixMilli()
	tokenID := snowflake.ID(4201)
	logs := []model.CallLog{
		{MODEL: coreAPI.MODEL{ID: 1, CreatedAt: now - int64(time.Hour/time.Millisecond)}, AccessTokenID: tokenID, CostMicroUSD: 100, Success: gatewayStatusNormal},
		{MODEL: coreAPI.MODEL{ID: 2, CreatedAt: now - int64(10*time.Hour/time.Millisecond)}, AccessTokenID: tokenID, CostMicroUSD: 200, Success: gatewayStatusNormal},
		{MODEL: coreAPI.MODEL{ID: 3, CreatedAt: now - int64(3*24*time.Hour/time.Millisecond)}, AccessTokenID: tokenID, CostMicroUSD: 300, Success: gatewayStatusNormal},
		{MODEL: coreAPI.MODEL{ID: 4, CreatedAt: now - int64(8*24*time.Hour/time.Millisecond)}, AccessTokenID: tokenID, CostMicroUSD: 400, Success: gatewayStatusNormal},
		{MODEL: coreAPI.MODEL{ID: 5, CreatedAt: now - int64(time.Hour/time.Millisecond)}, AccessTokenID: tokenID, CostMicroUSD: 500, Success: gatewayStatusDisabled},
		{MODEL: coreAPI.MODEL{ID: 6, CreatedAt: now - int64(time.Hour/time.Millisecond)}, AccessTokenID: snowflake.ID(9999), CostMicroUSD: 600, Success: gatewayStatusNormal},
	}
	require.NoError(t, database.Create(&logs).Error)

	repository := newGormGatewayRepository(func() *gorm.DB { return database })
	costs, err := repository.AccessTokenCostWindows(tokenID, now)
	require.NoError(t, err)
	assert.EqualValues(t, 100, costs.FiveHourMicroUSD)
	assert.EqualValues(t, 300, costs.DayMicroUSD)
	assert.EqualValues(t, 600, costs.SevenDayMicroUSD)
}
