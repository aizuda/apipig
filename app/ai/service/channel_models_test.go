package service

import (
	"encoding/json"
	"testing"

	"apipig/app/ai/model"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAccountModelMappingsNormalizeAndResolve(t *testing.T) {
	normalized, err := normalizeAccountModels("gpt-4o, gpt-4.1")
	require.NoError(t, err)
	assert.JSONEq(t, `{"gpt-4.1":"gpt-4.1","gpt-4o":"gpt-4o"}`, normalized)

	normalized, err = normalizeAccountModels(`{"chat-fast":"gpt-4o-mini","chat":"gpt-4o"}`)
	require.NoError(t, err)
	assert.JSONEq(t, `{"chat":"gpt-4o","chat-fast":"gpt-4o-mini"}`, normalized)

	providerModel, supported := resolveAccountModel(normalized, "chat-fast")
	require.True(t, supported)
	assert.Equal(t, "gpt-4o-mini", providerModel)
	_, supported = resolveAccountModel(normalized, "missing")
	assert.False(t, supported)
	assert.True(t, containsModel(normalized, "chat"))
}

func TestChannelModelMappingRoutesAliasToProviderModel(t *testing.T) {
	database := setupCredentialTestDB(t)
	providerID := snowflake.ID(5101)
	channelID := snowflake.ID(5102)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID}, Name: "OpenAI", Code: "openai", Protocol: "openai",
		BaseURL: "https://api.openai.com/v1", Models: "gpt-4o,gpt-4o-mini", Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL: coreAPI.MODEL{ID: channelID}, ProviderID: providerID, Name: "primary",
		ModelPricing: `[]`, Weight: 1, Status: gatewayStatusNormal,
	}).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{
		MODEL: coreAPI.MODEL{ID: snowflake.ID(5103)}, ChannelID: channelID, Name: "account",
		APIKey: "secret", Models: `{"chat":"gpt-4o","chat-fast":"gpt-4o-mini"}`, Status: gatewayStatusNormal,
	}).Error)

	target, err := (&GatewayService{}).pickRoute("chat-fast")
	require.NoError(t, err)
	assert.Equal(t, channelID, target.Channel.ID)
	providerModel, supported := resolveAccountModel(target.Account.Models, "chat-fast")
	require.True(t, supported)
	assert.Equal(t, "gpt-4o-mini", providerModel)
}

func TestValidateChannelPricingAgainstProvider(t *testing.T) {
	database := setupCredentialTestDB(t)
	providerID := snowflake.ID(5201)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID}, Name: "OpenAI", Code: "openai", Protocol: "openai",
		BaseURL: "https://api.openai.com/v1", Models: "gpt-4o", Status: gatewayStatusNormal,
	}).Error)
	store := newGormAIStore(func() *gorm.DB { return database })

	channel := &model.Channel{ProviderID: providerID, Name: "valid", ModelPricing: `[{"model":"gpt-4o"}]`, Weight: 1}
	require.NoError(t, validateChannel(channel, store))
	channel.ModelPricing = `[{"model":"missing"}]`
	assert.EqualError(t, validateChannel(channel, store), "计价模型 missing 不在供应商支持模型中")
}

func TestReplaceRequestModelUsesProviderMapping(t *testing.T) {
	body, err := replaceRequestModel([]byte(`{"model":"chat-fast","messages":[]}`), "gpt-4o-mini")
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "gpt-4o-mini", payload["model"])
}
