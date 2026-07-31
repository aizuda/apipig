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
	"apipig/toolkit"
	"apipig/toolkit/snowflake"
)

// AccessTokenService 负责访问令牌表的数据校验、持久化和查询。
type AccessTokenService struct{ store AIStore }

func (s *AccessTokenService) persistence() AIStore {
	return resolveAIStore(s.store)
}

func (s *AccessTokenService) Authenticate(rawToken string, clientIP string) (model.AccessToken, error) {
	trimmedToken := strings.TrimSpace(rawToken)
	if trimmedToken == "" {
		return model.AccessToken{}, errors.New("API Token is required")
	}
	var token model.AccessToken
	err := s.persistence().Query(model.AccessToken{}).
		Where("token IN ? AND status = ?", []string{hashGatewayToken(trimmedToken), trimmedToken}, gatewayStatusNormal).
		First(&token).Error
	if err != nil {
		return model.AccessToken{}, errors.New("API Token is invalid or disabled")
	}
	if token.ExpireAt > 0 && token.ExpireAt < toolkit.GetNowUnixMilli() {
		return model.AccessToken{}, errors.New("API Token has expired")
	}
	if !accessTokenAllowsIP(token, clientIP) {
		return model.AccessToken{}, errors.New("当前 IP 不允许使用此 API 密钥")
	}
	return token, nil
}

func (s *AccessTokenService) ValidateSession(id snowflake.ID, clientIP string) error {
	var token model.AccessToken
	if err := s.persistence().GetByID(&token, id); err != nil || token.Status != gatewayStatusNormal {
		return errors.New("API Token is invalid or disabled")
	}
	if token.ExpireAt > 0 && token.ExpireAt < toolkit.GetNowUnixMilli() {
		return errors.New("API Token has expired")
	}
	if !accessTokenAllowsIP(token, clientIP) {
		return errors.New("当前 IP 不允许使用此 API 密钥")
	}
	return nil
}

// Save 创建或更新访问令牌；新建时由系统生成密钥，更新时保留原密钥。
func (s *AccessTokenService) Save(params *aiReq.AccessTokenSaveParams) (aiResp.AccessTokenSaveResult, error) {
	if params == nil || params.AccessToken == nil {
		return aiResp.AccessTokenSaveResult{}, errors.New("访问 Token 参数不能为空")
	}
	m := params.AccessToken
	if err := normalizeAccessToken(m); err != nil {
		return aiResp.AccessTokenSaveResult{}, err
	}
	if err := validateAccessToken(m, s.persistence()); err != nil {
		return aiResp.AccessTokenSaveResult{}, err
	}
	m.TagIDs = normalizeAccessTokenTagIDs(m.TagIDs)
	if err := validateAccessTokenTagIDs(s.persistence(), m.TagIDs); err != nil {
		return aiResp.AccessTokenSaveResult{}, err
	}
	if m.ID == 0 {
		m.UsedAmount = 0
		m.UsedMicroUSD = 0
		m.QuotaMicroUSD = usdToMicroUSD(m.QuotaAmount)
		rawToken, err := generateSecureGatewayToken()
		if err != nil {
			return aiResp.AccessTokenSaveResult{}, err
		}
		m.Token = rawToken
		m.MODEL = db.NewModel(params.Ctx)
		var success bool
		err = s.persistence().Transaction(func(store AIStore) error {
			var createErr error
			success, createErr = store.Create(m)
			if createErr != nil {
				return createErr
			}
			return replaceAccessTokenTags(store, m.ID, m.TagIDs)
		})
		return aiResp.AccessTokenSaveResult{Success: success && err == nil, Token: rawToken}, err
	}
	var existing model.AccessToken
	if err := s.persistence().GetByID(&existing, m.ID); err != nil {
		return aiResp.AccessTokenSaveResult{}, err
	}
	m.UsedAmount = existing.UsedAmount
	m.UsedMicroUSD = existing.UsedMicroUSD
	m.QuotaMicroUSD = usdToMicroUSD(m.QuotaAmount)
	m.SuccessCount = existing.SuccessCount
	m.FailureCount = existing.FailureCount
	m.PromptTokensTotal = existing.PromptTokensTotal
	m.CompletionTokensTotal = existing.CompletionTokensTotal
	m.ReasoningTokensTotal = existing.ReasoningTokensTotal
	m.CacheReadTokensTotal = existing.CacheReadTokensTotal
	m.CacheWriteTokensTotal = existing.CacheWriteTokensTotal
	m.LastUsedAt = existing.LastUsedAt
	m.Token = existing.Token
	var success bool
	err := s.persistence().Transaction(func(store AIStore) error {
		var updateErr error
		success, updateErr = store.Update(m)
		if updateErr != nil {
			return updateErr
		}
		return replaceAccessTokenTags(store, m.ID, m.TagIDs)
	})
	return aiResp.AccessTokenSaveResult{Success: success && err == nil}, err
}

// ChangeStatus 切换 API 密钥启用、禁用状态。
func (s *AccessTokenService) ChangeStatus(params *aiReq.StatusChangeParams) (bool, error) {
	return changeResourceStatus(s.persistence(), model.AccessToken{}, "API 密钥", params)
}

// Delete 根据 ID 集合批量删除访问令牌。
// UpdateTags replaces tag relations without changing other API token fields.
func (s *AccessTokenService) UpdateTags(params *aiReq.AccessTokenTagUpdateParams) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("\u8bf7\u9009\u62e9\u8981\u66f4\u65b0\u6807\u7b7e\u7684 API \u5bc6\u94a5")
	}
	store := s.persistence()
	var token model.AccessToken
	if err := store.GetByID(&token, params.ID); err != nil {
		return false, err
	}
	tagIDs := normalizeAccessTokenTagIDs(params.TagIDs)
	if tagIDs == nil {
		tagIDs = []snowflake.ID{}
	}
	if err := validateAccessTokenTagIDs(store, tagIDs); err != nil {
		return false, err
	}
	if err := store.Transaction(func(transaction AIStore) error {
		return replaceAccessTokenTags(transaction, token.ID, tagIDs)
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (s *AccessTokenService) Delete(idsReq *request.IdsReq) (bool, error) {
	if idsReq == nil || len(idsReq.Ids) == 0 {
		return false, errors.New("请选择要删除的访问 Token")
	}
	var success bool
	err := s.persistence().Transaction(func(store AIStore) error {
		if err := store.Query(model.AccessTokenTagRelation{}).
			Where("access_token_id IN ?", idsReq.Ids).
			Delete(&model.AccessTokenTagRelation{}).Error; err != nil {
			return err
		}
		var deleteErr error
		success, deleteErr = store.DeleteByIDs(model.AccessToken{}, idsReq.Ids)
		return deleteErr
	})
	return success && err == nil, err
}

// Get 根据 ID 查询访问令牌。
func (s *AccessTokenService) Get(id snowflake.ID) (m model.AccessToken, err error) {
	err = s.persistence().GetByID(&m, id)
	if err == nil {
		syncAccessTokenAmounts(&m)
		err = loadAccessTokenTagIDs(s.persistence(), &m)
	}
	return
}

// Page 按名称和状态分页查询访问令牌。
func (s *AccessTokenService) Page(params *aiReq.AccessTokenPageParams) (response.PageResult, error) {
	query := s.persistence().Query(model.AccessToken{}).
		Joins("LEFT JOIN ap_ai_channel ON ap_ai_channel.id = ap_ai_access_token.channel_id AND ap_ai_channel.deleted_at = 0").
		Joins("LEFT JOIN ap_ai_provider ON ap_ai_provider.id = ap_ai_channel.provider_id AND ap_ai_provider.deleted_at = 0")
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where(
				"(ap_ai_access_token.name LIKE ? OR ap_ai_access_token.models LIKE ? OR ap_ai_access_token.remark LIKE ? OR ap_ai_channel.name LIKE ? OR ap_ai_provider.name LIKE ? OR EXISTS (SELECT 1 FROM ap_ai_access_token_tag_relation relation JOIN ap_ai_access_token_tag tag ON tag.id = relation.tag_id AND tag.deleted_at = 0 WHERE relation.access_token_id = ap_ai_access_token.id AND tag.name LIKE ?))",
				likeKeyword, likeKeyword, likeKeyword, likeKeyword, likeKeyword, likeKeyword,
			)
		}
		if params.Name != "" {
			query = query.Where("ap_ai_access_token.name LIKE ?", "%"+params.Name+"%")
		}
		if params.Status > 0 {
			query = query.Where("ap_ai_access_token.status = ?", params.Status)
		}
	}
	var arr []model.AccessToken
	result, err := s.persistence().Page(query.Order("ap_ai_access_token.created_at DESC"), pageInfo(params), arr)
	if err != nil {
		return result, err
	}
	if result.Total == 0 {
		result.Records = []aiResp.AccessTokenPageRecord{}
		return result, nil
	}
	tokens, ok := result.Records.([]model.AccessToken)
	if !ok {
		return result, errors.New("API key pagination data format error")
	}
	channelIDs := make([]snowflake.ID, 0, len(tokens))
	channelIDSet := make(map[snowflake.ID]struct{}, len(tokens))
	for index := range tokens {
		syncAccessTokenAmounts(&tokens[index])
		if tokens[index].ChannelID == 0 {
			continue
		}
		if _, exists := channelIDSet[tokens[index].ChannelID]; !exists {
			channelIDSet[tokens[index].ChannelID] = struct{}{}
			channelIDs = append(channelIDs, tokens[index].ChannelID)
		}
	}
	channelsByID := make(map[snowflake.ID]model.Channel, len(channelIDs))
	providerIDs := make([]snowflake.ID, 0, len(channelIDs))
	providerIDSet := make(map[snowflake.ID]struct{}, len(channelIDs))
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
	tagsByTokenID, tagIDsByTokenID, err := loadAccessTokenTagsByTokenIDs(s.persistence(), tokens)
	if err != nil {
		return result, err
	}
	pageRecords := make([]aiResp.AccessTokenPageRecord, 0, len(tokens))
	for _, token := range tokens {
		channel := channelsByID[token.ChannelID]
		token.TagIDs = tagIDsByTokenID[token.ID]
		pageRecords = append(pageRecords, aiResp.AccessTokenPageRecord{
			AccessToken:  token,
			ChannelName:  channel.Name,
			ProviderName: providerNames[channel.ProviderID],
			Tags:         tagsByTokenID[token.ID],
		})
	}
	result.Records = pageRecords
	return result, nil
}

// normalizeAccessToken 统一清理访问令牌字段并补齐默认值。
func normalizeAccessToken(m *model.AccessToken) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Models = normalizeModels(m.Models)
	if err := normalizeAccessTokenIPRule(m); err != nil {
		return err
	}
	if err := normalizeAccessTokenRateLimitRule(m); err != nil {
		return err
	}
	if m.RPM <= 0 {
		m.RPM = 60
	}
	m.Status = api.NormalDisable(m.Status)
	return nil
}

func normalizeAccessTokenTagIDs(tagIDs []snowflake.ID) []snowflake.ID {
	if tagIDs == nil {
		return nil
	}
	seen := make(map[snowflake.ID]struct{}, len(tagIDs))
	result := make([]snowflake.ID, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if tagID == 0 {
			continue
		}
		if _, exists := seen[tagID]; exists {
			continue
		}
		seen[tagID] = struct{}{}
		result = append(result, tagID)
	}
	return result
}

func validateAccessTokenTagIDs(store AIStore, tagIDs []snowflake.ID) error {
	if tagIDs == nil || len(tagIDs) == 0 {
		return nil
	}
	var count int64
	if err := store.Query(model.AccessTokenTag{}).Where("id IN ?", tagIDs).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(tagIDs)) {
		return errors.New("API 密钥关联的标签不存在")
	}
	return nil
}

func replaceAccessTokenTags(
	store AIStore,
	accessTokenID snowflake.ID,
	tagIDs []snowflake.ID,
) error {
	if tagIDs == nil {
		return nil
	}
	if err := store.Query(model.AccessTokenTagRelation{}).
		Where("access_token_id = ?", accessTokenID).
		Delete(&model.AccessTokenTagRelation{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	relations := make([]model.AccessTokenTagRelation, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		relations = append(relations, model.AccessTokenTagRelation{
			AccessTokenID: accessTokenID,
			TagID:         tagID,
		})
	}
	_, err := store.Create(&relations)
	return err
}

func loadAccessTokenTagIDs(store AIStore, token *model.AccessToken) error {
	var relations []model.AccessTokenTagRelation
	if err := store.Query(model.AccessTokenTagRelation{}).
		Where("access_token_id = ?", token.ID).
		Find(&relations).Error; err != nil {
		return err
	}
	token.TagIDs = make([]snowflake.ID, 0, len(relations))
	for _, relation := range relations {
		token.TagIDs = append(token.TagIDs, relation.TagID)
	}
	return nil
}

func loadAccessTokenTagsByTokenIDs(
	store AIStore,
	tokens []model.AccessToken,
) (map[snowflake.ID][]model.AccessTokenTag, map[snowflake.ID][]snowflake.ID, error) {
	tokenIDs := make([]snowflake.ID, 0, len(tokens))
	for _, token := range tokens {
		tokenIDs = append(tokenIDs, token.ID)
	}
	var relations []model.AccessTokenTagRelation
	if err := store.Query(model.AccessTokenTagRelation{}).
		Where("access_token_id IN ?", tokenIDs).
		Find(&relations).Error; err != nil {
		return nil, nil, err
	}
	tagIDSet := make(map[snowflake.ID]struct{}, len(relations))
	tagIDsByTokenID := make(map[snowflake.ID][]snowflake.ID, len(tokens))
	relationSetByTokenID := make(map[snowflake.ID]map[snowflake.ID]struct{}, len(tokens))
	for _, relation := range relations {
		tagIDSet[relation.TagID] = struct{}{}
		tagIDsByTokenID[relation.AccessTokenID] = append(tagIDsByTokenID[relation.AccessTokenID], relation.TagID)
		if relationSetByTokenID[relation.AccessTokenID] == nil {
			relationSetByTokenID[relation.AccessTokenID] = make(map[snowflake.ID]struct{})
		}
		relationSetByTokenID[relation.AccessTokenID][relation.TagID] = struct{}{}
	}
	tagIDs := make([]snowflake.ID, 0, len(tagIDSet))
	for tagID := range tagIDSet {
		tagIDs = append(tagIDs, tagID)
	}
	var tags []model.AccessTokenTag
	if len(tagIDs) > 0 {
		if err := store.Query(model.AccessTokenTag{}).
			Where("id IN ?", tagIDs).
			Order("sort_order ASC, created_at ASC").
			Find(&tags).Error; err != nil {
			return nil, nil, err
		}
	}
	tagsByTokenID := make(map[snowflake.ID][]model.AccessTokenTag, len(tokens))
	for _, token := range tokens {
		for _, tag := range tags {
			if _, exists := relationSetByTokenID[token.ID][tag.ID]; exists {
				tagsByTokenID[token.ID] = append(tagsByTokenID[token.ID], tag)
			}
		}
		orderedIDs := make([]snowflake.ID, 0, len(tagsByTokenID[token.ID]))
		for _, tag := range tagsByTokenID[token.ID] {
			orderedIDs = append(orderedIDs, tag.ID)
		}
		tagIDsByTokenID[token.ID] = orderedIDs
	}
	return tagsByTokenID, tagIDsByTokenID, nil
}
