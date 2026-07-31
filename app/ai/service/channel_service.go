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
	"apipig/toolkit/snowflake"
)

// ChannelService 负责渠道账号表的数据校验、持久化和查询。
type ChannelService struct {
	gateway *GatewayService
	store   AIStore
}

func (s *ChannelService) persistence() AIStore {
	return resolveAIStore(s.store)
}

// Save 创建或更新渠道账号。
func (s *ChannelService) Save(params *aiReq.ChannelSaveParams) (bool, error) {
	if params == nil || params.Channel == nil {
		return false, errors.New("渠道参数不能为空")
	}
	m := params.Channel
	if err := normalizeChannel(m); err != nil {
		return false, err
	}
	if err := validateChannel(m, s.persistence()); err != nil {
		return false, err
	}
	if m.ID == 0 {
		m.MODEL = db.NewModel(params.Ctx)
		success, err := s.persistence().Create(m)
		if success {
			if s.gateway != nil {
				s.gateway.invalidateRouteTargets()
			}
		}
		return success, err
	}
	success, err := s.persistence().Update(m)
	if success {
		if s.gateway != nil {
			s.gateway.invalidateRouteTargets()
		}
	}
	return success, err
}

// ChangeStatus 切换渠道账号启用、禁用状态。
func (s *ChannelService) ChangeStatus(params *aiReq.StatusChangeParams) (bool, error) {
	success, err := changeResourceStatus(s.persistence(), model.Channel{}, "渠道", params)
	if success && s.gateway != nil {
		s.gateway.invalidateRouteTargets()
	}
	return success, err
}

// Delete 根据 ID 集合批量删除渠道账号。
func (s *ChannelService) Delete(idsReq *request.IdsReq) (bool, error) {
	if idsReq == nil || len(idsReq.Ids) == 0 {
		return false, errors.New("请选择要删除的渠道")
	}
	var accountCount int64
	if err := s.persistence().Query(model.ChannelAccount{}).Where("channel_id IN ?", idsReq.Ids).Count(&accountCount).Error; err != nil {
		return false, err
	}
	if accountCount > 0 {
		return false, errors.New("渠道仍有关联账户，不能删除")
	}
	var tokenCount int64
	if err := s.persistence().Query(model.AccessToken{}).Where("channel_id IN ?", idsReq.Ids).Count(&tokenCount).Error; err != nil {
		return false, err
	}
	if tokenCount > 0 {
		return false, errors.New("渠道仍有关联 API 密钥，不能删除")
	}
	success, err := s.persistence().DeleteByIDs(model.Channel{}, idsReq.Ids)
	if success {
		if s.gateway != nil {
			s.gateway.invalidateRouteTargets()
		}
	}
	return success, err
}

// Get 根据 ID 查询渠道账号。
func (s *ChannelService) Get(id snowflake.ID) (m model.Channel, err error) {
	err = s.persistence().GetByID(&m, id)
	return
}

// Page 按供应商、名称、模型和状态分页查询渠道账号。
func (s *ChannelService) Page(params *aiReq.ChannelPageParams) (response.PageResult, error) {
	query := s.persistence().Query(model.Channel{})
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where("(name LIKE ? OR model_pricing LIKE ?)", likeKeyword, likeKeyword)
		}
		if params.ProviderID > 0 {
			query = query.Where("provider_id = ?", params.ProviderID)
		}
		if params.Name != "" {
			query = query.Where("name LIKE ?", "%"+params.Name+"%")
		}
		if params.Model != "" {
			query = query.Where("model_pricing = '' OR model_pricing LIKE ?", "%"+params.Model+"%")
		}
		if params.Status > 0 {
			query = query.Where("status = ?", params.Status)
		}
	}
	var arr []model.Channel
	result, err := s.persistence().Page(query.Order("priority DESC, created_at DESC"), pageInfo(params), arr)
	if err != nil {
		return result, err
	}
	if result.Total == 0 {
		result.Records = []aiResp.ChannelPageRecord{}
		return result, nil
	}
	channels, ok := result.Records.([]model.Channel)
	if !ok {
		return result, errors.New("渠道分页数据格式错误")
	}
	providerIDs := make([]snowflake.ID, 0, len(channels))
	proxyIDs := make([]snowflake.ID, 0, len(channels))
	providerIDSet := make(map[snowflake.ID]struct{}, len(channels))
	proxyIDSet := make(map[snowflake.ID]struct{}, len(channels))
	for _, channel := range channels {
		if _, exists := providerIDSet[channel.ProviderID]; !exists {
			providerIDSet[channel.ProviderID] = struct{}{}
			providerIDs = append(providerIDs, channel.ProviderID)
		}
		if channel.ProxyID > 0 {
			if _, exists := proxyIDSet[channel.ProxyID]; !exists {
				proxyIDSet[channel.ProxyID] = struct{}{}
				proxyIDs = append(proxyIDs, channel.ProxyID)
			}
		}
	}
	providerNames := make(map[snowflake.ID]string, len(providerIDs))
	providerModels := make(map[snowflake.ID]string, len(providerIDs))
	if len(providerIDs) > 0 {
		var providers []model.Provider
		if err := s.persistence().Query(model.Provider{}).Where("id IN ?", providerIDs).Find(&providers).Error; err != nil {
			return result, err
		}
		for _, provider := range providers {
			providerNames[provider.ID] = provider.Name
			providerModels[provider.ID] = provider.Models
		}
	}
	gatewayModelsByChannel := make(map[snowflake.ID][]string, len(channels))
	if len(channels) > 0 {
		channelIDs := make([]snowflake.ID, 0, len(channels))
		for _, channel := range channels {
			channelIDs = append(channelIDs, channel.ID)
		}
		var accounts []model.ChannelAccount
		if err := s.persistence().Query(model.ChannelAccount{}).Where("channel_id IN ? AND status = ?", channelIDs, gatewayStatusNormal).Find(&accounts).Error; err != nil {
			return result, err
		}
		for _, account := range accounts {
			models, err := accountGatewayModels(account.Models)
			if err != nil {
				return result, err
			}
			gatewayModelsByChannel[account.ChannelID] = append(gatewayModelsByChannel[account.ChannelID], models...)
		}
	}
	proxyNames := make(map[snowflake.ID]string, len(proxyIDs))
	if len(proxyIDs) > 0 {
		var proxies []model.Proxy
		if err := s.persistence().Query(model.Proxy{}).Where("id IN ?", proxyIDs).Find(&proxies).Error; err != nil {
			return result, err
		}
		for _, proxy := range proxies {
			proxyNames[proxy.ID] = proxy.Name
		}
	}
	records := make([]aiResp.ChannelPageRecord, 0, len(channels))
	for _, channel := range channels {
		availableModels := uniqueStrings(gatewayModelsByChannel[channel.ID])
		records = append(records, aiResp.ChannelPageRecord{
			Channel:         channel,
			ProviderName:    providerNames[channel.ProviderID],
			ProxyName:       proxyNames[channel.ProxyID],
			AvailableModels: availableModels,
			ProviderModels:  splitModels(providerModels[channel.ProviderID]),
		})
	}
	result.Records = records
	return result, nil
}

// normalizeChannel 统一清理渠道字段并补齐默认值。
func normalizeChannel(m *model.Channel) error {
	m.Name = strings.TrimSpace(m.Name)
	pricing, err := normalizeModelPricing(m.ModelPricing)
	if err != nil {
		return err
	}
	m.ModelPricing = pricing
	if m.Weight <= 0 {
		m.Weight = 1
	}
	m.CostMultiplier = normalizedCostMultiplier(m.CostMultiplier)
	m.Status = api.NormalDisable(m.Status)
	return nil
}
