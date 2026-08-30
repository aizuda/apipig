package service

import (
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/middleware"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNormalizeAIPageInfoBounds(t *testing.T) {
	info := normalizeAIPageInfo(coreReq.PageInfo{SearchCount: 99, Page: int(^uint(0) >> 1), PageSize: int(^uint(0) >> 1)})
	require.Equal(t, maxAIPage, info.Page)
	require.Equal(t, maxAIPageSize, info.PageSize)
	require.Zero(t, info.SearchCount)
}

func TestAIStoreUpdateAllowsBusinessZeroValuesAndProtectsAuditFields(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:ai-store-update?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.Provider{}))
	existing := model.Provider{
		MODEL: coreAPI.MODEL{ID: 201, CreatedId: 7, CreatedBy: "creator", CreatedAt: 123},
		Name:  "供应商", Code: "provider", Protocol: "openai", BaseURL: "https://example.com",
		Models: "old-model", TimeoutMs: 60_000, Status: 1, Remark: "待清空",
	}
	require.NoError(t, database.Create(&existing).Error)

	store := newGormAIStore(func() *gorm.DB { return database })
	updated := existing
	updated.CreatedId = 999
	updated.CreatedBy = "attacker"
	updated.CreatedAt = 999
	updated.Models = ""
	updated.Remark = ""
	success, err := store.Update(&updated)
	require.NoError(t, err)
	require.True(t, success)

	var stored model.Provider
	require.NoError(t, database.First(&stored, existing.ID).Error)
	require.Equal(t, "", stored.Models)
	require.Equal(t, "", stored.Remark)
	require.Equal(t, "creator", stored.CreatedBy)
	require.Equal(t, int64(123), stored.CreatedAt)
	require.Equal(t, existing.CreatedId, stored.CreatedId)
}

func TestAccessTokenSaveStoresOnlyHash(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:token-save-hash?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.Provider{}, &model.Channel{}, &model.ChannelAccount{}, &model.AccessToken{},
		&model.AccessTokenTag{}, &model.AccessTokenTagRelation{},
	))
	providerID, channelID := snowflake.ID(301), snowflake.ID(302)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID}, Name: "供应商", Code: "provider", BaseURL: "https://example.com",
		Models: "gpt-test", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL: coreAPI.MODEL{ID: channelID}, ProviderID: providerID, Name: "渠道", Weight: 1, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{
		MODEL: coreAPI.MODEL{ID: 303}, ChannelID: channelID, Name: "账户", APIKey: "legacy-key",
		Models: `{"gpt-test":"gpt-test"}`, Status: gatewayStatusNormal,
	}).Error)

	tokenService := &AccessTokenService{store: newGormAIStore(func() *gorm.DB { return database }), vault: newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" })}
	var saved aiResp.AccessTokenSaveResult
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		c.Locals("tokenClaims", &middleware.TokenClaims{ID: 1, Username: "tester"})
		var saveErr error
		saved, saveErr = tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: c, AccessToken: &model.AccessToken{
			ChannelID: channelID, Name: "新密钥", Models: "gpt-test", Status: gatewayStatusNormal,
		}})
		return saveErr
	})
	response, err := app.Test(httptest.NewRequest(http.MethodPost, "/", nil))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, fiber.StatusOK, response.StatusCode)
	require.NotEmpty(t, saved.Token)

	var stored model.AccessToken
	require.NoError(t, database.First(&stored).Error)
	require.True(t, isEncryptedCredential(stored.Token))
	require.NotEqual(t, saved.Token, stored.Token)
	managed, err := tokenService.Get(stored.ID)
	require.NoError(t, err)
	require.Empty(t, managed.Token)
}

func TestAccessTokenRawTokenUsesEncryptedStorage(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:token-raw?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.AccessToken{}))
	service := &AccessTokenService{
		store: newGormAIStore(func() *gorm.DB { return database }),
		vault: newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" }),
	}
	accessToken := model.AccessToken{MODEL: coreAPI.MODEL{ID: 361}, Name: "可导出密钥", Token: ""}
	require.NoError(t, database.Create(&accessToken).Error)
	// Simulate the encrypted value produced by Save without exposing it in the model JSON.
	encrypted, err := service.credentialVault().Encrypt("sk-raw")
	require.NoError(t, err)
	require.NoError(t, database.Model(&accessToken).Update("token", encrypted).Error)

	secret, err := service.GetRawToken(accessToken.ID)
	require.NoError(t, err)
	require.Equal(t, "sk-raw", secret.Token)
}

func TestAccessTokenResetRotatesEncryptedToken(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:token-reset?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.AccessToken{}))
	service := &AccessTokenService{
		store: newGormAIStore(func() *gorm.DB { return database }),
		vault: newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" }),
	}
	oldToken := "sk-old-token"
	encryptedOldToken, err := service.credentialVault().Encrypt(oldToken)
	require.NoError(t, err)
	accessToken := model.AccessToken{
		MODEL: coreAPI.MODEL{ID: 371}, Name: "待重置密钥", Token: encryptedOldToken,
		RPM: 60, Status: gatewayStatusNormal,
	}
	require.NoError(t, database.Create(&accessToken).Error)

	secret, err := service.ResetToken(&aiReq.AccessTokenResetParams{ID: accessToken.ID})
	require.NoError(t, err)
	require.Regexp(t, `^sk-[A-Za-z0-9_-]{43}$`, secret.Token)
	require.NotEqual(t, oldToken, secret.Token)

	var stored model.AccessToken
	require.NoError(t, database.First(&stored, accessToken.ID).Error)
	require.True(t, isEncryptedCredential(stored.Token))
	decrypted, err := service.credentialVault().Decrypt(stored.Token)
	require.NoError(t, err)
	require.Equal(t, secret.Token, decrypted)
	_, err = service.Authenticate(oldToken, "127.0.0.1")
	require.Error(t, err)
	authenticated, err := service.Authenticate(secret.Token, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, accessToken.ID, authenticated.ID)
}

func TestForwardToUpstreamDoesNotFollowRedirect(t *testing.T) {
	var redirectedRequests atomic.Int64
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirectedRequests.Add(1)
	}))
	defer redirectTarget.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, redirectTarget.URL, http.StatusFound)
	}))
	defer upstream.Close()

	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		status, _, _, err := (&GatewayService{}).forwardToUpstream(c, routeTarget{
			Provider: model.Provider{BaseURL: upstream.URL, TimeoutMs: 5_000},
			Account:  model.ChannelAccount{APIKey: "secret"},
		}, "/v1/test", nil, fiber.MIMEApplicationJSON)
		require.NoError(t, err)
		require.Equal(t, http.StatusFound, status)
		return c.SendStatus(fiber.StatusNoContent)
	})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, fiber.StatusNoContent, response.StatusCode)
	require.Zero(t, redirectedRequests.Load())
}

func TestGatewayBoundaryValidation(t *testing.T) {
	require.Error(t, validateGatewayUpstream("/v1/../admin"))
	require.Error(t, validateGatewayUpstream("/v1/%2e%2e/admin"))
	require.NoError(t, validateGatewayUpstream("/v1/chat/completions?trace=1"))
	require.True(t, isBlockedUpstreamResponseHeader("Connection"))
	require.True(t, isBlockedUpstreamResponseHeader("set-cookie"))
	require.False(t, isBlockedUpstreamResponseHeader("Content-Type"))
	require.Equal(t, strings.Repeat("中", 2), truncateUTF8(strings.Repeat("中", 3), 8))
	replaced, err := replaceRequestModel([]byte(`{"model":"old","seed":9007199254740993}`), "new")
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"new","seed":9007199254740993}`, string(replaced))
}

func TestStrictRulesAndBillingBounds(t *testing.T) {
	require.Error(t, validateAccessTokenIPRule(`{"enabled":true,"allowedIps":["127.0.0.1"]}`))
	require.Error(t, validateAccessTokenRateLimitRule(`{"enabled":true,"dailyAmount":1}`))
	require.Equal(t, maxMicroUSDValue, usdToMicroUSD(math.Inf(1)))
	require.Equal(t, maxMicroUSDValue, multiplyMicroUSD(maxMicroUSDValue, 1000))

	usage := normalizeBillingUsage(BillingUsage{InputTokens: -1, OutputTokens: int(^uint(0) >> 1)})
	require.Zero(t, usage.InputTokens)
	require.Equal(t, maxStoredTokenCount, usage.OutputTokens)
	require.Equal(t, maxStoredTokenCount, sumStoredTokenCounts(maxStoredTokenCount, 1))
}

func TestGatewaySummaryAppliesRangeToAllCallMetrics(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:summary-range?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.Provider{}, &model.Channel{}, &model.AccessToken{}, &model.Proxy{}, &model.CallLog{},
	))
	require.NoError(t, database.Create(&[]model.CallLog{
		{MODEL: coreAPI.MODEL{ID: 401, CreatedAt: 1_500}, RequestID: "inside", Model: "gpt-test", TotalTokens: 10, Cost: 2, Success: gatewayStatusNormal},
		{MODEL: coreAPI.MODEL{ID: 402, CreatedAt: 2_500}, RequestID: "outside", Model: "gpt-test", TotalTokens: 20, Cost: 3, Success: gatewayStatusDisabled},
	}).Error)

	repository := newGormGatewayRepository(func() *gorm.DB { return database })
	summary, err := repository.Summary(1_000, 2_000)
	require.NoError(t, err)
	require.Equal(t, int64(1), summary.CallCount)
	require.Equal(t, int64(1), summary.SuccessCount)
	require.Zero(t, summary.ErrorCount)
	require.Equal(t, int64(10), summary.TotalTokens)
	require.Equal(t, float64(2), summary.TotalCost)
}

func TestProtocolHTTPClientLimitsResponseBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("12345"))
	}))
	defer upstream.Close()
	client := &http.Client{Transport: &responseLimitTransport{base: http.DefaultTransport, maxBytes: 4}}
	response, err := client.Get(upstream.URL)
	if err != nil {
		require.Contains(t, err.Error(), "上游响应体超过")
		return
	}
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	require.ErrorContains(t, err, "上游响应体超过")
}
