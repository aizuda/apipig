package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"apipig/app/ai/model"
	aiReq "apipig/app/ai/model/request"
	aiResp "apipig/app/ai/model/response"
	"apipig/core/api"
	"apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit"
	"apipig/toolkit/snowflake"
)

// AccessTokenService 负责访问令牌表的数据校验、持久化和查询。
type AccessTokenService struct {
	store AIStore
	vault CredentialVault
}

func (s *AccessTokenService) credentialVault() CredentialVault {
	if s.vault != nil {
		return s.vault
	}
	return newAESCredentialVault(func() string { return global.CONFIG.AI.EncryptionKey })
}

func (s *AccessTokenService) persistence() AIStore {
	return resolveAIStore(s.store)
}

func (s *AccessTokenService) generateEncryptedToken() (string, string, error) {
	rawToken, err := generateSecureGatewayToken()
	if err != nil {
		return "", "", err
	}
	vault := s.credentialVault()
	if !vault.Enabled() {
		return "", "", errors.New("APIPIG_AI_ENCRYPTION_KEY is required")
	}
	encryptedToken, err := vault.Encrypt(rawToken)
	if err != nil {
		return "", "", err
	}
	return rawToken, encryptedToken, nil
}

func (s *AccessTokenService) Authenticate(rawToken string, clientIP string) (model.AccessToken, error) {
	if strings.TrimSpace(rawToken) == "" {
		return model.AccessToken{}, errors.New("API Token is required")
	}
	candidates, err := gatewayTokenLookupCandidates(rawToken)
	if err != nil {
		return model.AccessToken{}, errors.New("API Token is invalid or disabled")
	}
	var token model.AccessToken
	err = errors.New("not found")
	var records []model.AccessToken
	if scanErr := s.persistence().Query(model.AccessToken{}).Where("status = ?", gatewayStatusNormal).Find(&records).Error; scanErr == nil {
		for _, candidate := range records {
			if !isEncryptedCredential(candidate.Token) {
				continue
			}
			value, decryptErr := s.credentialVault().Decrypt(candidate.Token)
			if decryptErr == nil && value == candidates[0] {
				token, err = candidate, nil
				break
			}
		}
	}
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
		rawToken, encryptedToken, err := s.generateEncryptedToken()
		if err != nil {
			return aiResp.AccessTokenSaveResult{}, err
		}
		m.Token = encryptedToken
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

// GetRawToken returns a decrypted token for an authenticated administrator.
// Legacy records that only contain a SHA-256 hash cannot be recovered.
func (s *AccessTokenService) GetRawToken(id snowflake.ID) (aiResp.AccessTokenSecret, error) {
	if id == 0 {
		return aiResp.AccessTokenSecret{}, errors.New("API 密钥 ID 无效")
	}
	var token model.AccessToken
	if err := s.persistence().GetByID(&token, id); err != nil {
		return aiResp.AccessTokenSecret{}, err
	}
	if isEncryptedCredential(token.Token) {
		value, err := s.credentialVault().Decrypt(token.Token)
		if err != nil {
			return aiResp.AccessTokenSecret{}, err
		}
		return aiResp.AccessTokenSecret{Token: value}, nil
	}
	return aiResp.AccessTokenSecret{}, errors.New("该 API 密钥仅保存了不可恢复的哈希，请重新创建密钥")
}

// ResetToken 轮换 API 密钥；旧密钥在更新成功后立即失效，新密钥仅在响应中返回一次。
func (s *AccessTokenService) ResetToken(params *aiReq.AccessTokenResetParams) (aiResp.AccessTokenSecret, error) {
	if params == nil || params.ID == 0 {
		return aiResp.AccessTokenSecret{}, errors.New("请选择要重置的 API 密钥")
	}
	store := s.persistence()
	var token model.AccessToken
	if err := store.GetByID(&token, params.ID); err != nil {
		return aiResp.AccessTokenSecret{}, err
	}
	rawToken, encryptedToken, err := s.generateEncryptedToken()
	if err != nil {
		return aiResp.AccessTokenSecret{}, err
	}
	result := store.Query(model.AccessToken{}).
		Where("id = ?", token.ID).
		Update("token", encryptedToken)
	if result.Error != nil {
		return aiResp.AccessTokenSecret{}, result.Error
	}
	if result.RowsAffected != 1 {
		return aiResp.AccessTokenSecret{}, errors.New("API 密钥重置失败")
	}
	return aiResp.AccessTokenSecret{Token: rawToken}, nil
}

// ChangeStatus 切换 API 密钥启用、禁用状态。
func (s *AccessTokenService) ChangeStatus(params *aiReq.StatusChangeParams) (bool, error) {
	return changeResourceStatus(s.persistence(), model.AccessToken{}, "API 密钥", params)
}

// UpdateTags 仅替换标签关联，不修改 API 密钥的其他字段。
func (s *AccessTokenService) UpdateTags(params *aiReq.AccessTokenTagUpdateParams) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("请选择要更新标签的 API 密钥")
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

// Delete 根据 ID 集合批量删除访问令牌。
func (s *AccessTokenService) Delete(idsReq *request.IdsReq) (bool, error) {
	if err := validateAIBulkIDs(idsReq, "请选择要删除的访问 Token"); err != nil {
		return false, err
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
		m.Token = ""
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
		token.Token = ""
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

// Statistics 按 API 密钥聚合指定时间范围内的调用和 Token 用量。
func (s *AccessTokenService) Statistics(params *aiReq.AccessTokenStatisticsParams) (aiResp.AccessTokenStatistics, error) {
	statistics := aiResp.AccessTokenStatistics{
		Items: []aiResp.AccessTokenStatisticRecord{}, ModelStatistics: []aiResp.AccessTokenModelStatistic{},
	}
	page, pageSize, offset := 1, 10, 0
	keyword := ""
	var tagID snowflake.ID
	var accessTokenID snowflake.ID
	if params != nil {
		safePageInfo := normalizeAIPageInfo(params.PageInfo)
		page, pageSize, offset = safePageInfo.PageOffset()
		keyword = strings.ToLower(strings.TrimSpace(params.Keyword))
		tagID = params.TagID
		accessTokenID = params.AccessTokenID
		statistics.StartAt = params.StartAt
		statistics.EndAt = params.EndAt
		if params.StartAt < 0 || params.EndAt < 0 {
			return statistics, errors.New("统计时间不能为负数")
		}
		if (params.StartAt == 0) != (params.EndAt == 0) {
			return statistics, errors.New("统计开始时间和结束时间必须同时提供")
		}
		if params.StartAt > 0 && params.EndAt < params.StartAt {
			return statistics, errors.New("统计结束时间不能早于开始时间")
		}
	}
	statistics.Page = page
	statistics.PageSize = pageSize

	var tokens []model.AccessToken
	tokenQuery := s.persistence().Query(model.AccessToken{}).
		Select("id, name, status").
		Order("created_at ASC")
	if accessTokenID > 0 {
		tokenQuery = tokenQuery.Where("id = ?", accessTokenID)
	}
	if keyword != "" {
		tokenQuery = tokenQuery.Where("LOWER(ap_ai_access_token.name) LIKE ?", "%"+keyword+"%")
	}
	if tagID > 0 {
		tokenQuery = tokenQuery.Where(
			"EXISTS (SELECT 1 FROM ap_ai_access_token_tag_relation relation WHERE relation.access_token_id = ap_ai_access_token.id AND relation.tag_id = ?)",
			tagID,
		)
	}
	if err := tokenQuery.Find(&tokens).Error; err != nil {
		return statistics, err
	}
	statistics.TokenCount = int64(len(tokens))
	statistics.Total = statistics.TokenCount
	if len(tokens) == 0 {
		return statistics, nil
	}
	tokenIDs := make([]snowflake.ID, 0, len(tokens))
	for _, token := range tokens {
		tokenIDs = append(tokenIDs, token.ID)
	}

	type usageAggregate struct {
		AccessTokenID    snowflake.ID `gorm:"column:access_token_id"`
		CallCount        int64        `gorm:"column:call_count"`
		SuccessCount     int64        `gorm:"column:success_count"`
		FailureCount     int64        `gorm:"column:failure_count"`
		PromptTokens     int64        `gorm:"column:prompt_tokens"`
		CompletionTokens int64        `gorm:"column:completion_tokens"`
		ReasoningTokens  int64        `gorm:"column:reasoning_tokens"`
		CacheReadTokens  int64        `gorm:"column:cache_read_tokens"`
		CacheWriteTokens int64        `gorm:"column:cache_write_tokens"`
		TotalTokens      int64        `gorm:"column:total_tokens"`
		LastUsedAt       int64        `gorm:"column:last_used_at"`
	}

	query := s.persistence().Query(model.CallLog{}).
		Select(`access_token_id,
			COUNT(*) AS call_count,
			COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) AS success_count,
			COALESCE(SUM(CASE WHEN success <> 1 THEN 1 ELSE 0 END), 0) AS failure_count,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(MAX(created_at), 0) AS last_used_at`).
		Where("access_token_id IN ?", tokenIDs).
		Group("access_token_id")
	if statistics.StartAt > 0 {
		query = query.Where("created_at >= ? AND created_at <= ?", statistics.StartAt, statistics.EndAt)
	}
	var aggregates []usageAggregate
	if err := query.Scan(&aggregates).Error; err != nil {
		return statistics, err
	}
	usageByTokenID := make(map[snowflake.ID]usageAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		usageByTokenID[aggregate.AccessTokenID] = aggregate
	}

	type modelUsageAggregate struct {
		Model            string  `gorm:"column:model"`
		CallCount        int64   `gorm:"column:call_count"`
		SuccessCount     int64   `gorm:"column:success_count"`
		FailureCount     int64   `gorm:"column:failure_count"`
		PromptTokens     int64   `gorm:"column:prompt_tokens"`
		CompletionTokens int64   `gorm:"column:completion_tokens"`
		ReasoningTokens  int64   `gorm:"column:reasoning_tokens"`
		CacheReadTokens  int64   `gorm:"column:cache_read_tokens"`
		CacheWriteTokens int64   `gorm:"column:cache_write_tokens"`
		TotalTokens      int64   `gorm:"column:total_tokens"`
		Cost             float64 `gorm:"column:cost"`
		LatencyTotal     int64   `gorm:"column:latency_total"`
		LastUsedAt       int64   `gorm:"column:last_used_at"`
	}
	modelQuery := s.persistence().Query(model.CallLog{}).
		Select(`model,
			COUNT(*) AS call_count,
			COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) AS success_count,
			COALESCE(SUM(CASE WHEN success <> 1 THEN 1 ELSE 0 END), 0) AS failure_count,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(cost), 0) AS cost,
			COALESCE(SUM(CASE WHEN success = 1 THEN latency_ms ELSE 0 END), 0) AS latency_total,
			COALESCE(MAX(created_at), 0) AS last_used_at`).
		Where("access_token_id IN ?", tokenIDs).
		Group("model").
		Order("total_tokens DESC, model ASC")
	if statistics.StartAt > 0 {
		modelQuery = modelQuery.Where("created_at >= ? AND created_at <= ?", statistics.StartAt, statistics.EndAt)
	}
	var modelAggregates []modelUsageAggregate
	if err := modelQuery.Scan(&modelAggregates).Error; err != nil {
		return statistics, err
	}
	for _, aggregate := range modelAggregates {
		avgLatencyMs := int64(0)
		if aggregate.SuccessCount > 0 {
			avgLatencyMs = aggregate.LatencyTotal / aggregate.SuccessCount
		}
		statistics.ModelStatistics = append(statistics.ModelStatistics, aiResp.AccessTokenModelStatistic{
			Model: aggregate.Model, CallCount: aggregate.CallCount,
			SuccessCount: aggregate.SuccessCount, FailureCount: aggregate.FailureCount,
			PromptTokens: aggregate.PromptTokens, CompletionTokens: aggregate.CompletionTokens,
			ReasoningTokens: aggregate.ReasoningTokens, CacheReadTokens: aggregate.CacheReadTokens,
			CacheWriteTokens: aggregate.CacheWriteTokens, TotalTokens: aggregate.TotalTokens,
			Cost: aggregate.Cost, AvgLatencyMs: avgLatencyMs, LastUsedAt: aggregate.LastUsedAt,
		})
	}

	items := make([]aiResp.AccessTokenStatisticRecord, 0, len(tokens))
	for _, token := range tokens {
		usage := usageByTokenID[token.ID]
		item := aiResp.AccessTokenStatisticRecord{
			TokenID: token.ID, TokenName: token.Name, Status: token.Status,
			CallCount: usage.CallCount, SuccessCount: usage.SuccessCount, FailureCount: usage.FailureCount,
			PromptTokens: usage.PromptTokens, CompletionTokens: usage.CompletionTokens,
			ReasoningTokens: usage.ReasoningTokens, CacheReadTokens: usage.CacheReadTokens,
			CacheWriteTokens: usage.CacheWriteTokens, TotalTokens: usage.TotalTokens,
			LastUsedAt: usage.LastUsedAt,
		}
		if item.CallCount > 0 {
			statistics.ActiveTokenCount++
		}
		statistics.CallCount += item.CallCount
		statistics.SuccessCount += item.SuccessCount
		statistics.FailureCount += item.FailureCount
		statistics.TotalTokens += item.TotalTokens
		items = append(items, item)
	}
	sort.SliceStable(items, func(left, right int) bool {
		if items[left].TotalTokens == items[right].TotalTokens {
			return items[left].TokenName < items[right].TokenName
		}
		return items[left].TotalTokens > items[right].TotalTokens
	})
	if offset < len(items) {
		end := min(offset+pageSize, len(items))
		statistics.Items = items[offset:end]
	}
	return statistics, nil
}

// normalizeAccessToken 统一清理访问令牌字段并补齐默认值。
func normalizeAccessToken(m *model.AccessToken) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Models = normalizeModels(m.Models)
	m.Remark = strings.TrimSpace(m.Remark)
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
	if len(tagIDs) > maxAIBulkIDs {
		return fmt.Errorf("单个 API 密钥关联标签不能超过 %d 个", maxAIBulkIDs)
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
