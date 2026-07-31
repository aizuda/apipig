package service

import (
	"errors"
	"strings"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"
)

// ChannelAccountService 负责渠道注册账户的数据校验、持久化和查询。
type ChannelAccountService struct {
	gateway *GatewayService
	vault   CredentialVault
	store   AIStore
}

func (s *ChannelAccountService) dependencies() (*GatewayService, CredentialVault) {
	vault := s.vault
	if vault == nil {
		vault = newAESCredentialVault(func() string { return global.CONFIG.AI.EncryptionKey })
	}
	return s.gateway, vault
}

func (s *ChannelAccountService) persistence() AIStore {
	return resolveAIStore(s.store)
}

// Save 创建或更新渠道账户。
func (s *ChannelAccountService) Save(params *aiReq.ChannelAccountSaveParams) (bool, error) {
	if params == nil || params.Account == nil {
		return false, errors.New("账户参数不能为空")
	}
	m := params.Account
	m.Name = strings.TrimSpace(m.Name)
	m.Remark = strings.TrimSpace(m.Remark)
	m.Status = api.NormalDisable(m.Status)
	models, err := normalizeAccountModels(m.Models)
	if err != nil {
		return false, err
	}
	m.Models = models
	if m.ChannelID == 0 || m.Name == "" {
		return false, errors.New("关联渠道和账户名称不能为空")
	}
	var channel model.Channel
	if err := s.persistence().GetByID(&channel, m.ChannelID); err != nil {
		return false, errors.New("关联渠道不存在")
	}
	var providerModel model.Provider
	if err := s.persistence().GetByID(&providerModel, channel.ProviderID); err != nil {
		return false, errors.New("关联渠道的供应商不存在")
	}
	mappings, _ := parseAccountModelMappings(m.Models)
	if len(mappings) == 0 {
		return false, errors.New("账户模型映射不能为空")
	}
	for _, upstreamModel := range mappings {
		if upstreamModel != "*" && !containsModel(providerModel.Models, upstreamModel) {
			return false, errors.New("账户上游模型不在供应商支持列表中")
		}
	}
	if m.ID == 0 {
		if err := requireCredential(m.APIKey, "账户 API Key"); err != nil {
			return false, err
		}
		_, vault := s.dependencies()
		storedKey, err := vault.Encrypt(m.APIKey)
		if err != nil {
			return false, err
		}
		m.APIKey = storedKey
		m.MODEL = db.NewModel(params.Ctx)
		success, err := s.persistence().Create(m)
		if success && s.gateway != nil {
			s.gateway.invalidateRouteTargets()
		}
		return success, err
	}
	preserveCredential := strings.TrimSpace(m.APIKey) == "" || isMaskedCredential(m.APIKey)
	if preserveCredential {
		var existing model.ChannelAccount
		if err := s.persistence().GetByID(&existing, m.ID); err != nil {
			return false, err
		}
		m.APIKey = existing.APIKey
	}
	_, vault := s.dependencies()
	if !preserveCredential || vault.Enabled() {
		storedKey, err := vault.Encrypt(m.APIKey)
		if err != nil {
			return false, err
		}
		m.APIKey = storedKey
	}
	success, err := s.persistence().Update(m)
	if success && s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
	return success, err
}

// ChangeStatus 切换渠道账户启用、禁用状态。
func (s *ChannelAccountService) ChangeStatus(params *aiReq.StatusChangeParams) (bool, error) {
	success, err := changeResourceStatus(s.persistence(), model.ChannelAccount{}, "渠道账户", params)
	if success && s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
	return success, err
}

// Delete 根据 ID 集合批量删除渠道账户。
func (s *ChannelAccountService) Delete(idsReq *request.IdsReq) (bool, error) {
	if idsReq == nil || len(idsReq.Ids) == 0 {
		return false, errors.New("请选择要删除的账户")
	}
	success, err := s.persistence().DeleteByIDs(model.ChannelAccount{}, idsReq.Ids)
	if success && s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
	return success, err
}

// Get 根据 ID 查询渠道账户。
func (s *ChannelAccountService) Get(id snowflake.ID) (m model.ChannelAccount, err error) {
	err = s.persistence().GetByID(&m, id)
	if err != nil {
		return m, err
	}
	_, vault := s.dependencies()
	m.APIKey, err = vault.Decrypt(m.APIKey)
	return
}

// Page 按渠道、名称和状态分页查询。
func (s *ChannelAccountService) Page(params *aiReq.ChannelAccountPageParams) (response.PageResult, error) {
	query := s.persistence().Query(model.ChannelAccount{}).
		Joins("LEFT JOIN ap_ai_channel ON ap_ai_channel.id = ap_ai_channel_account.channel_id AND ap_ai_channel.deleted_at = 0").
		Joins("LEFT JOIN ap_ai_provider ON ap_ai_provider.id = ap_ai_channel.provider_id AND ap_ai_provider.deleted_at = 0")
	if params != nil {
		if params.ChannelID > 0 {
			query = query.Where("ap_ai_channel_account.channel_id = ?", params.ChannelID)
		}
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where(
				"ap_ai_channel_account.name LIKE ? OR ap_ai_channel_account.remark LIKE ? OR ap_ai_channel.name LIKE ? OR ap_ai_provider.name LIKE ?",
				likeKeyword, likeKeyword, likeKeyword, likeKeyword,
			)
		}
		if params.Name != "" {
			query = query.Where("ap_ai_channel_account.name LIKE ?", "%"+strings.TrimSpace(params.Name)+"%")
		}
		if params.Status > 0 {
			query = query.Where("ap_ai_channel_account.status = ?", params.Status)
		}
	}
	var records []model.ChannelAccount
	result, err := s.persistence().Page(query.Order("ap_ai_channel_account.created_at DESC"), pageInfo(params), records)
	if err != nil {
		return result, err
	}
	if result.Total == 0 {
		result.Records = []aiResp.ChannelAccountPageRecord{}
		return result, nil
	}
	accounts, ok := result.Records.([]model.ChannelAccount)
	if !ok {
		return result, errors.New("账户分页数据格式错误")
	}
	channelIDs := make([]snowflake.ID, 0, len(accounts))
	channelIDSet := make(map[snowflake.ID]struct{}, len(accounts))
	for _, account := range accounts {
		if _, exists := channelIDSet[account.ChannelID]; !exists {
			channelIDSet[account.ChannelID] = struct{}{}
			channelIDs = append(channelIDs, account.ChannelID)
		}
	}
	channelsByID := make(map[snowflake.ID]model.Channel, len(channelIDs))
	providerIDSet := make(map[snowflake.ID]struct{}, len(channelIDs))
	providerIDs := make([]snowflake.ID, 0, len(channelIDs))
	if len(channelIDs) > 0 {
		var channels []model.Channel
		if err := s.persistence().Query(model.Channel{}).Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return result, err
		}
		for _, channel := range channels {
			channelsByID[channel.ID] = channel
			if _, exists := providerIDSet[channel.ProviderID]; !exists {
				providerIDSet[channel.ProviderID] = struct{}{}
				providerIDs = append(providerIDs, channel.ProviderID)
			}
		}
	}
	providerNames := make(map[snowflake.ID]string, len(providerIDs))
	if len(providerIDs) > 0 {
		var providers []model.Provider
		if err := s.persistence().Query(model.Provider{}).Where("id IN ?", providerIDs).Find(&providers).Error; err != nil {
			return result, err
		}
		for _, provider := range providers {
			providerNames[provider.ID] = provider.Name
		}
	}
	pageRecords := make([]aiResp.ChannelAccountPageRecord, 0, len(accounts))
	_, vault := s.dependencies()
	for _, account := range accounts {
		channel := channelsByID[account.ChannelID]
		account.APIKey, err = vault.Decrypt(account.APIKey)
		if err != nil {
			return result, err
		}
		pageRecords = append(pageRecords, aiResp.ChannelAccountPageRecord{
			ChannelAccount: account,
			ChannelName:    channel.Name,
			ProviderName:   providerNames[channel.ProviderID],
		})
	}
	result.Records = pageRecords
	return result, nil
}
