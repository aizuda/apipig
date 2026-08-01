package service

import (
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/middleware"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAccessTokenSaveAlwaysGeneratesAndPreservesToken(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:access-token-save?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.Provider{},
		&model.Channel{},
		&model.AccessToken{},
		&model.AccessTokenTag{},
		&model.AccessTokenTagRelation{},
	))
	providerID := snowflake.ID(3001)
	channelID := snowflake.ID(3002)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: coreAPI.MODEL{ID: providerID}, Name: "OpenAI", Code: "openai", BaseURL: "https://api.openai.com", Models: "gpt-4o",
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL: coreAPI.MODEL{ID: channelID}, ProviderID: providerID, Name: "OpenAI 主渠道",
	}).Error)
	require.NoError(t, database.Create(&model.ChannelAccount{MODEL: coreAPI.MODEL{ID: 3004}, ChannelID: channelID, Name: "account", APIKey: "key", Models: `{"chat":"gpt-4o"}`, Status: gatewayStatusNormal}).Error)
	tagID := snowflake.ID(3003)
	require.NoError(t, database.Create(&model.AccessTokenTag{
		MODEL: coreAPI.MODEL{ID: tagID}, Name: "生产", Sort: 1,
	}).Error)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)
	ctx.Locals("tokenClaims", &middleware.TokenClaims{ID: snowflake.ID(1), Username: "tester"})

	tokenService := &AccessTokenService{store: newGormAIStore(func() *gorm.DB { return database })}
	_, err = tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: ctx, AccessToken: &model.AccessToken{
		Name: "未关联渠道", Models: "gpt-4o", RPM: 60, Status: gatewayStatusNormal,
	}})
	require.EqualError(t, err, "API 密钥关联渠道不能为空")

	_, err = tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: ctx, AccessToken: &model.AccessToken{
		ChannelID: snowflake.ID(9999), Name: "无效渠道", RPM: 60, Status: gatewayStatusNormal,
	}})
	require.EqualError(t, err, "API 密钥关联的渠道不存在")

	_, err = tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: ctx, AccessToken: &model.AccessToken{
		ChannelID: channelID, Name: "未选择模型", RPM: 60, Status: gatewayStatusNormal,
	}})
	require.EqualError(t, err, "API 密钥关联渠道模型不能为空")

	_, err = tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: ctx, AccessToken: &model.AccessToken{
		ChannelID: channelID, Name: "越权模型", Models: "missing", RPM: 60, Status: gatewayStatusNormal,
	}})
	require.EqualError(t, err, "API 密钥模型 missing 不在关联号池可用模型中")

	created, err := tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: ctx, AccessToken: &model.AccessToken{
		ChannelID: channelID, Name: "生产环境", Token: "user-defined-token", Models: "chat", RPM: 60, Status: gatewayStatusNormal,
		TagIDs: []snowflake.ID{tagID},
	}})
	require.NoError(t, err)
	require.True(t, created.Success)
	require.NotEmpty(t, created.Token)
	assert.NotEqual(t, "user-defined-token", created.Token)

	var stored model.AccessToken
	require.NoError(t, database.Where("name = ?", "生产环境").First(&stored).Error)
	assert.Equal(t, created.Token, stored.Token)
	assert.Equal(t, channelID, stored.ChannelID)
	var relationCount int64
	require.NoError(t, database.Model(&model.AccessTokenTagRelation{}).
		Where("access_token_id = ? AND tag_id = ?", stored.ID, tagID).
		Count(&relationCount).Error)
	assert.EqualValues(t, 1, relationCount)

	page, err := tokenService.Page(&aiReq.AccessTokenPageParams{
		PageInfo: coreReq.PageInfo{Page: 1, PageSize: 10}, Keyword: "OpenAI 主渠道",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, page.Total)

	updated, err := tokenService.Save(&aiReq.AccessTokenSaveParams{Ctx: ctx, AccessToken: &model.AccessToken{
		MODEL: stored.MODEL, ChannelID: channelID, Name: "生产环境更新", Token: "replacement-token", Models: "chat", RPM: 120, Status: gatewayStatusNormal,
		TagIDs: []snowflake.ID{},
	}})
	require.NoError(t, err)
	require.True(t, updated.Success)
	assert.Empty(t, updated.Token)

	var persisted model.AccessToken
	require.NoError(t, database.First(&persisted, stored.ID).Error)
	assert.Equal(t, created.Token, persisted.Token)
	assert.Equal(t, channelID, persisted.ChannelID)
	assert.Equal(t, "生产环境更新", persisted.Name)
	require.NoError(t, database.Model(&model.AccessTokenTagRelation{}).
		Where("access_token_id = ?", stored.ID).
		Count(&relationCount).Error)
	assert.Zero(t, relationCount)
}

func TestAccessTokenPageAssemblesCurrentPageAssociations(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:access-token-page-associations?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.Provider{},
		&model.Channel{},
		&model.AccessToken{},
		&model.AccessTokenTag{},
		&model.AccessTokenTagRelation{},
	))

	providers := []model.Provider{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3101)}, Name: "OpenAI", Code: "openai", BaseURL: "https://api.openai.com"},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3102)}, Name: "Anthropic", Code: "anthropic", BaseURL: "https://api.anthropic.com"},
	}
	channels := []model.Channel{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3201)}, ProviderID: providers[0].ID, Name: "OpenAI main"},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3202)}, ProviderID: providers[1].ID, Name: "Anthropic backup"},
	}
	tokens := []model.AccessToken{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3301), CreatedAt: 2}, ChannelID: channels[0].ID, Name: "first", Token: "sk-first", RPM: 60},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3302), CreatedAt: 1}, ChannelID: channels[1].ID, Name: "second", Token: "sk-second", RPM: 60},
	}
	require.NoError(t, database.Create(&providers).Error)
	require.NoError(t, database.Create(&channels).Error)
	require.NoError(t, database.Create(&tokens).Error)
	tags := []model.AccessTokenTag{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3401), CreatedAt: 1}, Name: "生产", Sort: 1},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3402), CreatedAt: 2}, Name: "核心", Sort: 2},
	}
	require.NoError(t, database.Create(&tags).Error)
	require.NoError(t, database.Create(&[]model.AccessTokenTagRelation{
		{AccessTokenID: tokens[0].ID, TagID: tags[1].ID},
		{AccessTokenID: tokens[0].ID, TagID: tags[0].ID},
	}).Error)

	tokenService := &AccessTokenService{store: newGormAIStore(func() *gorm.DB { return database })}
	page, err := tokenService.Page(&aiReq.AccessTokenPageParams{PageInfo: coreReq.PageInfo{Page: 1, PageSize: 1}})
	require.NoError(t, err)
	assert.EqualValues(t, 2, page.Total)
	records, ok := page.Records.([]aiResp.AccessTokenPageRecord)
	require.True(t, ok)
	require.Len(t, records, 1)
	assert.Equal(t, "first", records[0].Name)
	assert.Equal(t, "OpenAI main", records[0].ChannelName)
	assert.Equal(t, "OpenAI", records[0].ProviderName)
	assert.Equal(t, []snowflake.ID{tags[0].ID, tags[1].ID}, records[0].TagIDs)
	require.Len(t, records[0].Tags, 2)
	assert.Equal(t, "生产", records[0].Tags[0].Name)
	assert.Equal(t, "核心", records[0].Tags[1].Name)

	emptyPage, err := tokenService.Page(&aiReq.AccessTokenPageParams{
		PageInfo: coreReq.PageInfo{Page: 1, PageSize: 10}, Keyword: "missing token",
	})
	require.NoError(t, err)
	assert.IsType(t, []aiResp.AccessTokenPageRecord{}, emptyPage.Records)
}

func TestAccessTokenStatisticsAggregatesRangeAndIncludesUnusedTokens(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:access-token-statistics?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.AccessToken{}, &model.CallLog{}, &model.AccessTokenTagRelation{}))

	tokens := []model.AccessToken{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3451), CreatedAt: 1}, Name: "production", Token: "sk-production", RPM: 60, Status: gatewayStatusNormal},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3452), CreatedAt: 2}, Name: "unused", Token: "sk-unused", RPM: 60, Status: gatewayStatusDisabled},
	}
	require.NoError(t, database.Create(&tokens).Error)
	tagID := snowflake.ID(3453)
	require.NoError(t, database.Create(&model.AccessTokenTagRelation{AccessTokenID: tokens[0].ID, TagID: tagID}).Error)
	require.NoError(t, database.Create(&[]model.CallLog{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3461), CreatedAt: 1_000}, RequestID: "inside-success", AccessTokenID: tokens[0].ID, Success: gatewayStatusNormal, PromptTokens: 100, CompletionTokens: 40, ReasoningTokens: 10, CacheReadTokens: 20, CacheWriteTokens: 5, TotalTokens: 165},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3462), CreatedAt: 2_000}, RequestID: "inside-failure", AccessTokenID: tokens[0].ID, Success: gatewayStatusDisabled},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3463), CreatedAt: 3_000}, RequestID: "outside", AccessTokenID: tokens[0].ID, Success: gatewayStatusNormal, PromptTokens: 999, TotalTokens: 999},
	}).Error)

	tokenService := &AccessTokenService{store: newGormAIStore(func() *gorm.DB { return database })}
	statistics, err := tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{StartAt: 500, EndAt: 2_500})
	require.NoError(t, err)
	assert.EqualValues(t, 2, statistics.Total)
	assert.Equal(t, 1, statistics.Page)
	assert.Equal(t, 10, statistics.PageSize)
	assert.EqualValues(t, 2, statistics.TokenCount)
	assert.EqualValues(t, 1, statistics.ActiveTokenCount)
	assert.EqualValues(t, 2, statistics.CallCount)
	assert.EqualValues(t, 1, statistics.SuccessCount)
	assert.EqualValues(t, 1, statistics.FailureCount)
	assert.EqualValues(t, 165, statistics.TotalTokens)
	require.Len(t, statistics.Items, 2)
	assert.Equal(t, "production", statistics.Items[0].TokenName)
	assert.EqualValues(t, 100, statistics.Items[0].PromptTokens)
	assert.EqualValues(t, 40, statistics.Items[0].CompletionTokens)
	assert.EqualValues(t, 10, statistics.Items[0].ReasoningTokens)
	assert.EqualValues(t, 20, statistics.Items[0].CacheReadTokens)
	assert.EqualValues(t, 5, statistics.Items[0].CacheWriteTokens)
	assert.EqualValues(t, 2_000, statistics.Items[0].LastUsedAt)
	assert.Equal(t, "unused", statistics.Items[1].TokenName)
	assert.Zero(t, statistics.Items[1].CallCount)

	secondPage, err := tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{
		PageInfo: coreReq.PageInfo{Page: 2, PageSize: 1}, StartAt: 500, EndAt: 2_500,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 2, secondPage.Total)
	assert.EqualValues(t, 2, secondPage.TokenCount)
	assert.EqualValues(t, 2, secondPage.CallCount)
	require.Len(t, secondPage.Items, 1)
	assert.Equal(t, "unused", secondPage.Items[0].TokenName)

	filtered, err := tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{Keyword: "PROD"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, filtered.Total)
	assert.EqualValues(t, 1, filtered.TokenCount)
	assert.EqualValues(t, 3, filtered.CallCount)
	require.Len(t, filtered.Items, 1)
	assert.Equal(t, "production", filtered.Items[0].TokenName)

	byTag, err := tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{TagID: tagID})
	require.NoError(t, err)
	assert.EqualValues(t, 1, byTag.Total)
	assert.EqualValues(t, 1, byTag.TokenCount)
	require.Len(t, byTag.Items, 1)
	assert.Equal(t, "production", byTag.Items[0].TokenName)

	allTime, err := tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{})
	require.NoError(t, err)
	assert.EqualValues(t, 1_164, allTime.TotalTokens)
	assert.EqualValues(t, 3, allTime.CallCount)

	_, err = tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{StartAt: 2_000})
	require.EqualError(t, err, "统计开始时间和结束时间必须同时提供")
	_, err = tokenService.Statistics(&aiReq.AccessTokenStatisticsParams{StartAt: 2_000, EndAt: 1_000})
	require.EqualError(t, err, "统计结束时间不能早于开始时间")
}

func TestAccessTokenUpdateTagsReplacesRelations(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:access-token-update-tags?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.AccessToken{},
		&model.AccessTokenTag{},
		&model.AccessTokenTagRelation{},
	))

	token := model.AccessToken{
		MODEL: coreAPI.MODEL{ID: snowflake.ID(3501)},
		Name:  "tag target",
		Token: "sk-tag-target",
		RPM:   60,
	}
	tags := []model.AccessTokenTag{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3502)}, Name: "first", Sort: 1},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3503)}, Name: "second", Sort: 2},
	}
	require.NoError(t, database.Create(&token).Error)
	require.NoError(t, database.Create(&tags).Error)

	tokenService := &AccessTokenService{store: newGormAIStore(func() *gorm.DB { return database })}
	success, err := tokenService.UpdateTags(&aiReq.AccessTokenTagUpdateParams{
		ID:     token.ID,
		TagIDs: []snowflake.ID{tags[1].ID, tags[0].ID, tags[1].ID},
	})
	require.NoError(t, err)
	require.True(t, success)

	var relations []model.AccessTokenTagRelation
	require.NoError(t, database.Where("access_token_id = ?", token.ID).Find(&relations).Error)
	require.Len(t, relations, 2)

	success, err = tokenService.UpdateTags(&aiReq.AccessTokenTagUpdateParams{ID: token.ID})
	require.NoError(t, err)
	require.True(t, success)
	relations = nil
	require.NoError(t, database.Where("access_token_id = ?", token.ID).Find(&relations).Error)
	require.Empty(t, relations)

	success, err = tokenService.UpdateTags(&aiReq.AccessTokenTagUpdateParams{
		ID: token.ID, TagIDs: []snowflake.ID{snowflake.ID(999999)},
	})
	require.Error(t, err)
	require.False(t, success)
}

func TestAccessTokenDeleteRemovesTagRelations(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:access-token-delete-tags?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.AccessToken{},
		&model.AccessTokenTag{},
		&model.AccessTokenTagRelation{},
	))

	tokens := []model.AccessToken{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3701)}, Name: "delete target", Token: "sk-delete-target", RPM: 60},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3702)}, Name: "keep target", Token: "sk-keep-target", RPM: 60},
	}
	tags := []model.AccessTokenTag{
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3711)}, Name: "shared", Sort: 1},
		{MODEL: coreAPI.MODEL{ID: snowflake.ID(3712)}, Name: "exclusive", Sort: 2},
	}
	require.NoError(t, database.Create(&tokens).Error)
	require.NoError(t, database.Create(&tags).Error)
	require.NoError(t, database.Create(&[]model.AccessTokenTagRelation{
		{AccessTokenID: tokens[0].ID, TagID: tags[0].ID},
		{AccessTokenID: tokens[0].ID, TagID: tags[1].ID},
		{AccessTokenID: tokens[1].ID, TagID: tags[0].ID},
	}).Error)

	tokenService := &AccessTokenService{store: newGormAIStore(func() *gorm.DB { return database })}
	success, err := tokenService.Delete(&coreReq.IdsReq{Ids: []snowflake.ID{tokens[0].ID}})
	require.NoError(t, err)
	require.True(t, success)

	var deletedTokenRelationCount int64
	require.NoError(t, database.Model(&model.AccessTokenTagRelation{}).
		Where("access_token_id = ?", tokens[0].ID).
		Count(&deletedTokenRelationCount).Error)
	assert.Zero(t, deletedTokenRelationCount)

	var remainingTokenRelationCount int64
	require.NoError(t, database.Model(&model.AccessTokenTagRelation{}).
		Where("access_token_id = ?", tokens[1].ID).
		Count(&remainingTokenRelationCount).Error)
	assert.EqualValues(t, 1, remainingTokenRelationCount)

	var tagCount int64
	require.NoError(t, database.Model(&model.AccessTokenTag{}).Count(&tagCount).Error)
	assert.EqualValues(t, 2, tagCount)
}
