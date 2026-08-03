package service

import (
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelPageAssemblesCurrentPageAssociations(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:channel-page-associations?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.Provider{}, &model.Proxy{}, &model.Channel{}, &model.ChannelAccount{}))

	providers := []model.Provider{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(4101)}, Name: "OpenAI", Code: "openai", BaseURL: "https://api.openai.com", Models: "gpt-4o,gpt-4o-mini"},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(4102)}, Name: "Anthropic", Code: "anthropic", BaseURL: "https://api.anthropic.com", Models: "claude-sonnet"},
	}
	proxies := []model.Proxy{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(4201)}, Name: "美国代理", Scheme: "http", Host: "127.0.0.1", Port: 8080},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(4202)}, Name: "日本代理", Scheme: "http", Host: "127.0.0.2", Port: 8080},
	}
	channels := []model.Channel{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(4301)}, ProviderID: providers[0].ID, ProxyID: proxies[0].ID, Name: "OpenAI 主渠道", Priority: 20},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(4302)}, ProviderID: providers[1].ID, ProxyID: proxies[1].ID, Name: "Anthropic 备用渠道", Priority: 10},
	}
	require.NoError(t, database.Create(&providers).Error)
	require.NoError(t, database.Create(&proxies).Error)
	require.NoError(t, database.Create(&channels).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{MODEL: coreAPI.MODEL{ID: 4401}, ChannelID: channels[0].ID, Name: "account", APIKey: "key", Models: `{"chat":"gpt-4o","chat-fast":"gpt-4o-mini"}`}).Error)

	channelService := &ChannelService{store: newGormAIStore(func() *gorm.DB { return database })}
	page, err := channelService.Page(&aiReq.ChannelPageParams{PageInfo: coreReq.PageInfo{Page: 1, PageSize: 1}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, page.Total)
	records, ok := page.Records.([]aiResp.ChannelPageRecord)
	require.True(t, ok)
	require.Len(t, records, 1)
	assert.Equal(t, "OpenAI 主渠道", records[0].Name)
	assert.Equal(t, "OpenAI", records[0].ProviderName)
	assert.Equal(t, "美国代理", records[0].ProxyName)
	assert.Equal(t, []string{"chat", "chat-fast"}, records[0].AvailableModels)
	assert.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, records[0].ProviderModels)

	emptyPage, err := channelService.Page(&aiReq.ChannelPageParams{
		PageInfo: coreReq.PageInfo{Page: 1, PageSize: 10}, Keyword: "不存在的渠道",
	})
	require.NoError(t, err)
	assert.IsType(t, []aiResp.ChannelPageRecord{}, emptyPage.Records)
}
