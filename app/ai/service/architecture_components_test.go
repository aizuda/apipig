package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"apipig/app/ai/model"
	aiResp "apipig/app/ai/model/response"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESCredentialVault(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	vault := newAESCredentialVault(func() string { return key })
	encrypted, err := vault.Encrypt("secret-value")
	require.NoError(t, err)
	assert.True(t, isEncryptedCredential(encrypted))
	assert.NotContains(t, encrypted, "secret-value")
	plaintext, err := vault.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "secret-value", plaintext)

	wrongVault := newAESCredentialVault(func() string { return "fedcba9876543210fedcba9876543210" })
	_, err = wrongVault.Decrypt(encrypted)
	assert.ErrorContains(t, err, "解密失败")

	disabledVault := newAESCredentialVault(func() string { return "" })
	_, err = disabledVault.Encrypt("new-secret")
	assert.ErrorContains(t, err, "必须配置")
	plaintext, err = disabledVault.Decrypt("legacy-plaintext")
	require.NoError(t, err)
	assert.Equal(t, "legacy-plaintext", plaintext)
	_, err = disabledVault.Decrypt(encrypted)
	assert.ErrorContains(t, err, "未配置")
}

func TestRouteLoadingMigratesPlaintextCredentials(t *testing.T) {
	repository := &fakeGatewayRepository{config: routeConfig{
		Providers: []model.Provider{{
			MODEL: coreAPI.MODEL{ID: 1}, Name: "provider", Code: "provider", Protocol: "openai", Models: "gpt-test", Status: gatewayStatusNormal,
		}},
		Channels: []model.Channel{{
			MODEL: coreAPI.MODEL{ID: 2}, ProviderID: 1, Name: "channel", Priority: 10, Weight: 1, Status: gatewayStatusNormal, ProxyID: 3,
		}},
		Accounts: []model.ChannelAccount{{
			MODEL: coreAPI.MODEL{ID: 4}, ChannelID: 2, Name: "account", APIKey: "plain-api-key", Models: "gpt-test", Status: gatewayStatusNormal,
		}},
		Proxies: []model.Proxy{{
			MODEL: coreAPI.MODEL{ID: 3}, Name: "proxy", Scheme: "http", Host: "127.0.0.1", Port: 8080, Password: "plain-password", Status: gatewayStatusNormal,
		}},
	}}
	vault := newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" })
	service := NewGatewayService(GatewayDependencies{Repository: repository, Vault: vault})
	targets, err := service.loadRouteTargets()
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "plain-api-key", targets[0].Account.APIKey)
	require.NotNil(t, targets[0].Proxy)
	assert.Equal(t, "plain-password", targets[0].Proxy.Password)
	assert.True(t, isEncryptedCredential(repository.accountAPIKey))
	assert.True(t, isEncryptedCredential(repository.proxyPassword))
}

func TestAsyncCallLogSinkFlushesOnClose(t *testing.T) {
	repository := &fakeGatewayRepository{}
	sink := newAsyncCallLogSink(repository, func() AsyncLogOptions {
		return AsyncLogOptions{QueueSize: 2, BatchSize: 2, FlushInterval: time.Hour}
	}, nil)
	for index := 0; index < 10; index++ {
		require.NoError(t, sink.Enqueue(model.CallLog{RequestID: string(rune('a' + index))}))
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, sink.Close(ctx))
	assert.Len(t, repository.callLogs(), 10)
	assert.Error(t, sink.Enqueue(model.CallLog{RequestID: "closed"}))
}

func TestAsyncCallLogSinkFallsBackWhenBatchFails(t *testing.T) {
	repository := &fakeGatewayRepository{failBatch: true}
	sink := newAsyncCallLogSink(repository, func() AsyncLogOptions {
		return AsyncLogOptions{QueueSize: 4, BatchSize: 2, FlushInterval: time.Hour}
	}, nil)
	require.NoError(t, sink.Enqueue(model.CallLog{RequestID: "one"}))
	require.NoError(t, sink.Enqueue(model.CallLog{RequestID: "two"}))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, sink.Close(ctx))
	assert.Len(t, repository.callLogs(), 2)
}

type fakeGatewayRepository struct {
	mu            sync.Mutex
	config        routeConfig
	accountAPIKey string
	proxyPassword string
	logs          []model.CallLog
	failBatch     bool
	costWindows   accessTokenCostWindows
	costWindowErr error
}

func (r *fakeGatewayRepository) Summary(_, _ int64) (aiResp.GatewaySummary, error) {
	return aiResp.GatewaySummary{}, nil
}

func (r *fakeGatewayRepository) FindAccessToken(_ []string) (model.AccessToken, error) {
	return model.AccessToken{}, errors.New("not implemented")
}

func (r *fakeGatewayRepository) FindAccessTokenByID(_ snowflake.ID) (model.AccessToken, error) {
	return model.AccessToken{}, errors.New("not implemented")
}

func (r *fakeGatewayRepository) AccessTokenCostWindows(_ snowflake.ID, _ int64) (accessTokenCostWindows, error) {
	return r.costWindows, r.costWindowErr
}

func (r *fakeGatewayRepository) UpdateAccessTokenHash(_ snowflake.ID, _ string) error { return nil }

func (r *fakeGatewayRepository) RecordAccessTokenUsage(_ AccessTokenUsageRecord) error { return nil }

func (r *fakeGatewayRepository) LoadRouteConfig() (routeConfig, error) { return r.config, nil }

func (r *fakeGatewayRepository) UpdateChannelAccountAPIKey(_ snowflake.ID, encrypted string) error {
	r.mu.Lock()
	r.accountAPIKey = encrypted
	r.mu.Unlock()
	return nil
}

func (r *fakeGatewayRepository) UpdateProxyPassword(_ snowflake.ID, encrypted string) error {
	r.mu.Lock()
	r.proxyPassword = encrypted
	r.mu.Unlock()
	return nil
}

func (r *fakeGatewayRepository) CreateCallLogs(records []model.CallLog) error {
	if r.failBatch && len(records) > 1 {
		return errors.New("batch failed")
	}
	r.mu.Lock()
	r.logs = append(r.logs, records...)
	r.mu.Unlock()
	return nil
}

func (r *fakeGatewayRepository) Ping() error { return nil }

func (r *fakeGatewayRepository) callLogs() []model.CallLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]model.CallLog(nil), r.logs...)
}
