package service

import (
	"testing"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestChannelAccountRequiresChannelAndProtectsChannelDelete(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:channel-account?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.Provider{}, &model.Channel{}, &model.ChannelAccount{}, &model.AccessToken{}))

	providerID := snowflake.ID(2001)
	channelID := snowflake.ID(2002)
	accountID := snowflake.ID(2003)
	tokenID := snowflake.ID(2004)
	require.NoError(t, database.Create(&model.Provider{
		MODEL: api.MODEL{ID: providerID}, Name: "OpenAI", Code: "openai", BaseURL: "https://api.openai.com", Models: "gpt-test",
	}).Error)
	require.NoError(t, database.Create(&model.Channel{
		MODEL: api.MODEL{ID: channelID}, ProviderID: providerID, Name: "OpenAI 主渠道",
	}).Error)

	store := newGormAIStore(func() *gorm.DB { return database })
	vault := newAESCredentialVault(func() string { return "0123456789abcdef0123456789abcdef" })
	accountService := &ChannelAccountService{store: store, vault: vault}
	_, err = accountService.Save(&aiReq.ChannelAccountSaveParams{Account: &model.ChannelAccount{
		MODEL: api.MODEL{ID: accountID}, ChannelID: snowflake.ID(9999), Name: "主账户",
	}})
	require.EqualError(t, err, "关联渠道不存在")
	_, err = accountService.Save(&aiReq.ChannelAccountSaveParams{Account: &model.ChannelAccount{
		ChannelID: channelID, Name: "主账户", Models: "gpt-test",
	}})
	require.EqualError(t, err, "账户 API Key不能为空")

	encryptedKey, err := vault.Encrypt("account-secret")
	require.NoError(t, err)
	require.NoError(t, database.Create(&model.ChannelAccount{
		MODEL: api.MODEL{ID: accountID}, ChannelID: channelID, Name: "主账户", APIKey: encryptedKey, Models: "gpt-test", Status: 1,
	}).Error)
	pageResult, err := accountService.Page(&aiReq.ChannelAccountPageParams{})
	require.NoError(t, err)
	pageRecords := pageResult.Records.([]aiResp.ChannelAccountPageRecord)
	require.Len(t, pageRecords, 1)
	require.Equal(t, "account-secret", pageRecords[0].APIKey)
	require.Equal(t, "OpenAI 主渠道", pageRecords[0].ChannelName)
	require.Equal(t, "OpenAI", pageRecords[0].ProviderName)
	account, err := accountService.Get(accountID)
	require.NoError(t, err)
	require.Equal(t, "account-secret", account.APIKey)
	success, err := accountService.Save(&aiReq.ChannelAccountSaveParams{Account: &model.ChannelAccount{
		MODEL: api.MODEL{ID: accountID}, ChannelID: channelID, Name: "主账户更新", APIKey: maskedCredential, Models: "gpt-test", Status: 1,
	}})
	require.NoError(t, err)
	require.True(t, success)
	var storedAccount model.ChannelAccount
	require.NoError(t, database.First(&storedAccount, accountID).Error)
	require.Equal(t, encryptedKey, storedAccount.APIKey)
	channelService := &ChannelService{store: store}
	_, err = channelService.Delete(&request.IdsReq{Ids: []snowflake.ID{channelID}})
	require.EqualError(t, err, "渠道仍有关联账户，不能删除")

	success, err = accountService.Delete(&request.IdsReq{Ids: []snowflake.ID{accountID}})
	require.NoError(t, err)
	require.True(t, success)
	require.NoError(t, database.Create(&model.AccessToken{
		MODEL: api.MODEL{ID: tokenID}, ChannelID: channelID, Name: "渠道密钥", Token: "channel-token", Status: 1,
	}).Error)
	_, err = channelService.Delete(&request.IdsReq{Ids: []snowflake.ID{channelID}})
	require.EqualError(t, err, "渠道仍有关联 API 密钥，不能删除")
	require.NoError(t, database.Delete(&model.AccessToken{}, tokenID).Error)
	success, err = channelService.Delete(&request.IdsReq{Ids: []snowflake.ID{channelID}})
	require.NoError(t, err)
	require.True(t, success)
}
