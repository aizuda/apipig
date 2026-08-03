package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSecureGatewayTokenAndHash(t *testing.T) {
	first, err := generateSecureGatewayToken()
	require.NoError(t, err)
	second, err := generateSecureGatewayToken()
	require.NoError(t, err)
	assert.NotEqual(t, first, second)
	assert.Regexp(t, `^sk-[A-Za-z0-9_-]{43}$`, first)
	assert.True(t, isHashedGatewayToken(hashGatewayToken(first)))
	assert.NotContains(t, hashGatewayToken(first), first)
}

func TestAuthenticateGatewayTokenKeepsPlaintext(t *testing.T) {
	database := setupCredentialTestDB(t)
	legacyToken := "sk-apipig-legacy"
	require.NoError(t, database.Create(&model.AccessToken{
		MODEL: coreAPI.MODEL{ID: 1, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "legacy", Token: legacyToken, RPM: 60, Status: gatewayStatusNormal,
	}).Error)

	service := &GatewayService{}
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		token, err := service.authenticateGatewayToken(c)
		if err != nil {
			return err
		}
		return c.SendString(token.Name)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+legacyToken)
	response, err := app.Test(request)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, response.StatusCode)
	assert.Equal(t, "legacy", string(body))

	var stored model.AccessToken
	require.NoError(t, database.First(&stored, 1).Error)
	assert.Equal(t, legacyToken, stored.Token)
}

func TestManagementPagesReturnExpectedCredentials(t *testing.T) {
	database := setupCredentialTestDB(t)
	providerID := snowflake.ID(10)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "provider", Code: "provider", Protocol: "openai", BaseURL: "https://example.com/v1", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL:      coreAPI.MODEL{ID: 11, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		ProviderID: providerID, Name: "channel", Weight: 1, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{
		MODEL:     coreAPI.MODEL{ID: 14, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		ChannelID: 11, Name: "account", APIKey: "secret-key", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.AccessToken{
		MODEL: coreAPI.MODEL{ID: 12, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "token", Token: "sk-39c-example-0061", RPM: 60, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Proxy{
		MODEL: coreAPI.MODEL{ID: 13, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "proxy", Scheme: "http", Host: "127.0.0.1", Port: 8080, Password: "secret-password", Status: gatewayStatusNormal,
	}).Error)

	accountPage, err := (&ChannelAccountService{}).Page(&aiReq.ChannelAccountPageParams{PageInfo: coreReq.PageInfo{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	accounts := accountPage.Records.([]aiResp.ChannelAccountPageRecord)
	assert.Equal(t, "secret-key", accounts[0].APIKey)

	tokenPage, err := (&AccessTokenService{}).Page(&aiReq.AccessTokenPageParams{PageInfo: coreReq.PageInfo{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	tokens := tokenPage.Records.([]aiResp.AccessTokenPageRecord)
	assert.Equal(t, "sk-39c-example-0061", tokens[0].Token)

	proxyPage, err := (&ProxyService{}).Page(&aiReq.ProxyPageParams{PageInfo: coreReq.PageInfo{Page: 1, PageSize: 10}})
	require.NoError(t, err)
	proxies := proxyPage.Records.([]model.Proxy)
	assert.Equal(t, maskedCredential, proxies[0].Password)
}

func TestGatewayRuntimeConcurrencyAndTPM(t *testing.T) {
	service := &GatewayService{}
	var waitGroup sync.WaitGroup
	for index := 0; index < 100; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			service.allowRate("token:1", 1000, 1000, 0)
		}()
	}
	waitGroup.Wait()
	service.recordRateTokens("token:1", 1000)
	assert.False(t, service.allowRate("token:1", 1000, 1000, 0))
}

func TestExpiredBreakerIsReset(t *testing.T) {
	breaker := newLocalCircuitBreaker()
	now := time.Now()
	breaker.now = func() time.Time { return now }
	service := &GatewayService{breaker: breaker}
	channelID := snowflake.ID(99)
	for index := 0; index < 5; index++ {
		service.recordBreakerFailure(channelID)
	}
	assert.True(t, service.breakerOpen(channelID))
	now = now.Add(time.Minute + time.Second)
	assert.False(t, service.breakerOpen(channelID))
	_, exists := breaker.states.Load(channelID)
	assert.False(t, exists)
}

func TestPickRouteOnlyUsesHighestPriorityGroup(t *testing.T) {
	database := setupCredentialTestDB(t)
	providerID := snowflake.ID(20)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "provider", Code: "priority-provider", Protocol: "openai", BaseURL: "https://example.com/v1", Models: "gpt-test", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL:      coreAPI.MODEL{ID: 21, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		ProviderID: providerID, Name: "primary", Priority: 100, Weight: 1, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL:      coreAPI.MODEL{ID: 22, CreatedId: 1, CreatedBy: "test", CreatedAt: 2},
		ProviderID: providerID, Name: "fallback", Priority: 10, Weight: 10000, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{
		MODEL:     coreAPI.MODEL{ID: 23, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		ChannelID: 21, Name: "primary-account", APIKey: "primary-key", Models: "gpt-test", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{
		MODEL:     coreAPI.MODEL{ID: 24, CreatedId: 1, CreatedBy: "test", CreatedAt: 2},
		ChannelID: 22, Name: "fallback-account", APIKey: "fallback-key", Models: "gpt-test", Status: gatewayStatusNormal,
	}).Error)

	service := &GatewayService{}
	for index := 0; index < 20; index++ {
		target, err := service.pickRoute("gpt-test")
		require.NoError(t, err)
		assert.Equal(t, "primary", target.Channel.Name)
	}
}

func TestPickRouteForChannelOnlyUsesTokenBoundChannel(t *testing.T) {
	providerID := snowflake.ID(30)
	boundChannelID := snowflake.ID(31)
	preferredChannelID := snowflake.ID(32)
	repository := &fakeGatewayRepository{config: routeConfig{
		Providers: []model.Provider{{
			MODEL: coreAPI.MODEL{ID: providerID}, Name: "provider", Code: "bound-provider", Protocol: "openai", BaseURL: "https://example.com/v1", Models: "gpt-test", Status: gatewayStatusNormal,
		}},
		Channels: []model.Channel{
			{MODEL: coreAPI.MODEL{ID: preferredChannelID}, ProviderID: providerID, Name: "preferred", Priority: 100, Weight: 10000, Status: gatewayStatusNormal},
			{MODEL: coreAPI.MODEL{ID: boundChannelID}, ProviderID: providerID, Name: "bound", Priority: 1, Weight: 1, Status: gatewayStatusNormal},
		},
		Accounts: []model.ChannelAccount{
			{MODEL: coreAPI.MODEL{ID: snowflake.ID(33)}, ChannelID: boundChannelID, Name: "bound-account", APIKey: "bound-key", Models: "gpt-test", Status: gatewayStatusNormal},
			{MODEL: coreAPI.MODEL{ID: snowflake.ID(34)}, ChannelID: preferredChannelID, Name: "preferred-account", APIKey: "preferred-key", Models: "gpt-test", Status: gatewayStatusNormal},
		},
	}}
	vault := newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" })
	service := NewGatewayService(GatewayDependencies{Repository: repository, Vault: vault})
	for index := 0; index < 20; index++ {
		target, err := service.pickRouteForChannel(boundChannelID, "gpt-test")
		require.NoError(t, err)
		assert.Equal(t, "bound", target.Channel.Name)
	}

	target, err := service.pickRoute("gpt-test")
	require.NoError(t, err)
	assert.Equal(t, "preferred", target.Channel.Name)

	_, err = service.pickRouteForChannel(snowflake.ID(99), "gpt-test")
	require.Error(t, err)
}

func TestGenerateInternalRoutesToTokenBoundChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"id":"chatcmpl-test","model":"gpt-test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	tokenID := snowflake.ID(40)
	providerID := snowflake.ID(41)
	boundChannelID := snowflake.ID(42)
	preferredChannelID := snowflake.ID(43)
	repository := &internalGatewayRepository{
		token: model.AccessToken{
			MODEL: coreAPI.MODEL{ID: tokenID}, ChannelID: boundChannelID, Name: "token", Token: "token",
			Models: "gpt-test", RPM: 60, Status: gatewayStatusNormal,
		},
		fakeGatewayRepository: fakeGatewayRepository{
			config: routeConfig{
				Providers: []model.Provider{{
					MODEL: coreAPI.MODEL{ID: providerID}, Name: "provider", Code: "bound-provider", Protocol: "openai",
					BaseURL: server.URL, Models: "gpt-test", TimeoutMs: 5000, Status: gatewayStatusNormal,
				}},
				Channels: []model.Channel{
					{MODEL: coreAPI.MODEL{ID: preferredChannelID}, ProviderID: providerID, Name: "preferred", Priority: 100, Weight: 10000, Status: gatewayStatusNormal},
					{MODEL: coreAPI.MODEL{ID: boundChannelID}, ProviderID: providerID, Name: "bound", Priority: 1, Weight: 1, Status: gatewayStatusNormal},
				},
				Accounts: []model.ChannelAccount{
					{MODEL: coreAPI.MODEL{ID: snowflake.ID(44)}, ChannelID: boundChannelID, Name: "bound-account", APIKey: "bound-key", Models: "gpt-test", Status: gatewayStatusNormal},
					{MODEL: coreAPI.MODEL{ID: snowflake.ID(45)}, ChannelID: preferredChannelID, Name: "preferred-account", APIKey: "preferred-key", Models: "gpt-test", Status: gatewayStatusNormal},
				},
			},
		},
	}
	vault := newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" })
	logSink := &fakeCallLogSink{}
	service := NewGatewayService(GatewayDependencies{Repository: repository, Vault: vault, LogSink: logSink})

	result, err := service.GenerateInternal(InternalGenerateParams{
		AccessTokenID: tokenID,
		Model:         "gpt-test",
		UserPrompt:    "hello",
		MaxTokens:     128,
		Path:          "/internal/apps/code-review",
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	require.Len(t, logSink.records, 1)
	assert.Equal(t, boundChannelID, logSink.records[0].ChannelID)
}

func TestGenerateInternalRejectsTokenWithoutBoundChannel(t *testing.T) {
	tokenID := snowflake.ID(50)
	repository := &internalGatewayRepository{
		token: model.AccessToken{
			MODEL: coreAPI.MODEL{ID: tokenID}, Name: "token", Token: "token",
			Models: "gpt-test", RPM: 60, Status: gatewayStatusNormal,
		},
	}
	service := NewGatewayService(GatewayDependencies{Repository: repository})

	_, err := service.GenerateInternal(InternalGenerateParams{
		AccessTokenID: tokenID,
		Model:         "gpt-test",
		UserPrompt:    "hello",
	})
	require.EqualError(t, err, "API key must be bound to a channel for internal AI calls")
}

type fakeCallLogSink struct {
	records []model.CallLog
}

func (s *fakeCallLogSink) Enqueue(record model.CallLog) error {
	s.records = append(s.records, record)
	return nil
}

func (s *fakeCallLogSink) Close(context.Context) error {
	return nil
}

type internalGatewayRepository struct {
	fakeGatewayRepository
	token model.AccessToken
}

func (r *internalGatewayRepository) FindAccessTokenByID(_ snowflake.ID) (model.AccessToken, error) {
	return r.token, nil
}

func setupCredentialTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.Provider{}, &model.Channel{}, &model.ChannelAccount{}, &model.AccessToken{},
		&model.AccessTokenTag{}, &model.AccessTokenTagRelation{}, &model.Proxy{},
	))
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
	return database
}
