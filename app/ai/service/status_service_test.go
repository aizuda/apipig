package service

import (
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagementStatusChanges(t *testing.T) {
	database := setupCredentialTestDB(t)
	providerID := snowflake.ID(201)
	channelID := snowflake.ID(202)
	tokenID := snowflake.ID(203)
	proxyID := snowflake.ID(204)

	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "provider", Code: "status-provider", Protocol: "openai", BaseURL: "https://example.com/v1", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL:      coreAPI.MODEL{ID: channelID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		ProviderID: providerID, Name: "channel", Weight: 1, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.AccessToken{
		MODEL: coreAPI.MODEL{ID: tokenID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "token", Token: "token-hash", RPM: 60, Status: gatewayStatusDisabled,
	}).Error)
	require.NoError(t, database.Create(&model.Proxy{
		MODEL: coreAPI.MODEL{ID: proxyID, CreatedId: 1, CreatedBy: "test", CreatedAt: 1},
		Name:  "proxy", Scheme: "http", Host: "127.0.0.1", Port: 8080, Status: gatewayStatusDisabled,
	}).Error)

	success, err := (&ProviderService{}).ChangeStatus(&aiReq.StatusChangeParams{ID: providerID, Status: gatewayStatusDisabled})
	require.NoError(t, err)
	assert.True(t, success)
	success, err = (&ChannelService{}).ChangeStatus(&aiReq.StatusChangeParams{ID: channelID, Status: gatewayStatusDisabled})
	require.NoError(t, err)
	assert.True(t, success)
	success, err = (&AccessTokenService{}).ChangeStatus(&aiReq.StatusChangeParams{ID: tokenID, Status: gatewayStatusNormal})
	require.NoError(t, err)
	assert.True(t, success)
	success, err = (&ProxyService{}).ChangeStatus(&aiReq.StatusChangeParams{ID: proxyID, Status: gatewayStatusNormal})
	require.NoError(t, err)
	assert.True(t, success)

	var provider model.Provider
	var channel model.Channel
	var accessToken model.AccessToken
	var proxy model.Proxy
	require.NoError(t, database.First(&provider, providerID).Error)
	require.NoError(t, database.First(&channel, channelID).Error)
	require.NoError(t, database.First(&accessToken, tokenID).Error)
	require.NoError(t, database.First(&proxy, proxyID).Error)
	assert.Equal(t, gatewayStatusDisabled, provider.Status)
	assert.Equal(t, gatewayStatusDisabled, channel.Status)
	assert.Equal(t, gatewayStatusNormal, accessToken.Status)
	assert.Equal(t, gatewayStatusNormal, proxy.Status)
}

func TestManagementStatusRejectsInvalidValue(t *testing.T) {
	_, err := (&ProviderService{}).ChangeStatus(&aiReq.StatusChangeParams{ID: 1, Status: 3})
	assert.EqualError(t, err, "状态仅支持启用或禁用")
}

func TestManagementStatusRejectsMissingResource(t *testing.T) {
	setupCredentialTestDB(t)
	_, err := (&ProviderService{}).ChangeStatus(&aiReq.StatusChangeParams{ID: 999, Status: gatewayStatusDisabled})
	assert.EqualError(t, err, "供应商不存在")
}
