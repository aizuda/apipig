package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"apipig/app/ai/model"
	aiResp "apipig/app/ai/model/response"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

type routeConfig struct {
	Providers []model.Provider
	Channels  []model.Channel
	Accounts  []model.ChannelAccount
	Proxies   []model.Proxy
}

type accessTokenCostWindows struct {
	FiveHourMicroUSD int64
	DayMicroUSD      int64
	SevenDayMicroUSD int64
}

// GatewayRepository 收口网关编排所需的持久化操作。
type GatewayRepository interface {
	Summary(startAt, endAt int64) (aiResp.GatewaySummary, error)
	FindAccessToken(candidates []string) (model.AccessToken, error)
	FindAccessTokenByID(id snowflake.ID) (model.AccessToken, error)
	AccessTokenCostWindows(tokenID snowflake.ID, now int64) (accessTokenCostWindows, error)
	RecordAccessTokenUsage(record AccessTokenUsageRecord) error
	LoadRouteConfig() (routeConfig, error)
	UpdateChannelAccountAPIKey(id snowflake.ID, encrypted string) error
	UpdateProxyPassword(id snowflake.ID, encrypted string) error
	CreateCallLogs(records []model.CallLog) error
	Ping() error
}

type gormGatewayRepository struct {
	dbProvider func() *gorm.DB
	vault      CredentialVault
}

func newGormGatewayRepository(dbProvider func() *gorm.DB, vault ...CredentialVault) GatewayRepository {
	var credentialVault CredentialVault
	if len(vault) > 0 {
		credentialVault = vault[0]
	}
	return &gormGatewayRepository{dbProvider: dbProvider, vault: credentialVault}
}

func (r *gormGatewayRepository) credentialVault() CredentialVault {
	if r.vault != nil {
		return r.vault
	}
	return newAESCredentialVault(func() string { return global.CONFIG.AI.EncryptionKey })
}

func (r *gormGatewayRepository) database() (*gorm.DB, error) {
	database := r.dbProvider()
	if database == nil {
		return nil, errors.New("数据库尚未初始化")
	}
	return database, nil
}

func (r *gormGatewayRepository) Summary(startAt, endAt int64) (summary aiResp.GatewaySummary, err error) {
	database, err := r.database()
	if err != nil {
		return summary, err
	}
	counts := []struct {
		query *gorm.DB
		value *int64
	}{
		{database.Model(&model.Provider{}), &summary.ProviderCount},
		{database.Model(&model.Channel{}), &summary.ChannelCount},
		{database.Model(&model.AccessToken{}), &summary.TokenCount},
		{database.Model(&model.Proxy{}), &summary.ProxyCount},
	}
	for _, count := range counts {
		if err = count.query.Count(count.value).Error; err != nil {
			return summary, err
		}
	}
	callRange := func() *gorm.DB {
		return database.Model(&model.CallLog{}).Where("created_at >= ? AND created_at <= ?", startAt, endAt)
	}
	callCounts := []struct {
		query *gorm.DB
		value *int64
	}{
		{callRange(), &summary.CallCount},
		{callRange().Where("success = ?", gatewayStatusNormal), &summary.SuccessCount},
		{callRange().Where("success = ?", gatewayStatusDisabled), &summary.ErrorCount},
	}
	for _, count := range callCounts {
		if err = count.query.Count(count.value).Error; err != nil {
			return summary, err
		}
	}
	if err = callRange().
		Select("COALESCE(SUM(total_tokens), 0), COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(reasoning_tokens), 0), COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_write_tokens), 0), COALESCE(SUM(standard_cost), 0), COALESCE(SUM(cost), 0)").
		Row().
		Scan(&summary.TotalTokens, &summary.PromptTokens, &summary.CompletionTokens, &summary.ReasoningTokens, &summary.CacheReadTokens, &summary.CacheWriteTokens, &summary.StandardCost, &summary.TotalCost); err != nil {
		return summary, err
	}
	modelDistributionQuery := database.Model(&model.CallLog{}).
		Select("model, COUNT(*) AS call_count, COALESCE(SUM(total_tokens), 0) AS token_count, COALESCE(SUM(cost), 0) AS cost").
		Where("success = ? AND model <> ''", gatewayStatusNormal)
	if startAt > 0 {
		modelDistributionQuery = modelDistributionQuery.Where("created_at >= ?", startAt)
	}
	if endAt > 0 {
		modelDistributionQuery = modelDistributionQuery.Where("created_at <= ?", endAt)
	}
	if err = modelDistributionQuery.
		Group("model").
		Order("token_count DESC, call_count DESC").
		Limit(8).
		Scan(&summary.ModelDistribution).Error; err != nil {
		return summary, err
	}
	var trendLogs []model.CallLog
	if err = database.Model(&model.CallLog{}).
		Select("created_at, channel_id, success, prompt_tokens, completion_tokens, cache_read_tokens, cache_write_tokens, total_tokens, cost, latency_ms").
		Where("created_at >= ? AND created_at <= ?", startAt, endAt).
		Find(&trendLogs).Error; err != nil {
		return summary, err
	}
	summary.TokenTrend = aggregateTokenTrend(trendLogs, time.UnixMilli(startAt), time.UnixMilli(endAt))
	channelIDs := make([]snowflake.ID, 0)
	seenChannelIDs := make(map[snowflake.ID]struct{})
	for _, logRecord := range trendLogs {
		if logRecord.ChannelID == 0 {
			continue
		}
		if _, exists := seenChannelIDs[logRecord.ChannelID]; exists {
			continue
		}
		seenChannelIDs[logRecord.ChannelID] = struct{}{}
		channelIDs = append(channelIDs, logRecord.ChannelID)
	}
	var channels []model.Channel
	if len(channelIDs) > 0 {
		if err = database.Unscoped().Where("id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return summary, err
		}
	}
	summary.ChannelStatistics = aggregateChannelStatistics(trendLogs, channels)
	return summary, nil
}

func aggregateTokenTrend(logs []model.CallLog, start, end time.Time) []aiResp.GatewayTokenTrend {
	start = startOfLocalDay(start)
	end = startOfLocalDay(end)
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	result := make([]aiResp.GatewayTokenTrend, days)
	indexes := make(map[string]int, len(result))
	for index := range result {
		date := start.AddDate(0, 0, index).Format("2006-01-02")
		result[index].Date = date
		indexes[date] = index
	}
	for _, logRecord := range logs {
		if logRecord.Success != gatewayStatusNormal {
			continue
		}
		date := time.UnixMilli(logRecord.CreatedAt).In(start.Location()).Format("2006-01-02")
		index, ok := indexes[date]
		if !ok {
			continue
		}
		result[index].PromptTokens += int64(logRecord.PromptTokens)
		result[index].CompletionTokens += int64(logRecord.CompletionTokens)
		result[index].CacheTokens += int64(logRecord.CacheReadTokens + logRecord.CacheWriteTokens)
		result[index].TotalTokens += int64(logRecord.TotalTokens)
	}
	return result
}

func aggregateChannelStatistics(logs []model.CallLog, channels []model.Channel) []aiResp.GatewayChannelStatistic {
	type channelAccumulator struct {
		stat         aiResp.GatewayChannelStatistic
		latencyTotal int64
	}
	names := make(map[snowflake.ID]string, len(channels))
	for _, channel := range channels {
		names[channel.ID] = channel.Name
	}
	accumulators := make(map[snowflake.ID]*channelAccumulator)
	for _, logRecord := range logs {
		if logRecord.ChannelID == 0 {
			continue
		}
		accumulator := accumulators[logRecord.ChannelID]
		if accumulator == nil {
			name := names[logRecord.ChannelID]
			if name == "" {
				name = logRecord.ChannelID.String()
			}
			accumulator = &channelAccumulator{stat: aiResp.GatewayChannelStatistic{
				ChannelID: logRecord.ChannelID, ChannelName: name,
			}}
			accumulators[logRecord.ChannelID] = accumulator
		}
		accumulator.stat.CallCount++
		if logRecord.Success != gatewayStatusNormal {
			continue
		}
		accumulator.stat.SuccessCount++
		accumulator.stat.TokenCount += int64(logRecord.TotalTokens)
		accumulator.stat.Cost += logRecord.Cost
		accumulator.latencyTotal += logRecord.LatencyMs
	}
	result := make([]aiResp.GatewayChannelStatistic, 0, len(accumulators))
	for _, accumulator := range accumulators {
		if accumulator.stat.CallCount > 0 {
			accumulator.stat.SuccessRate = float64(accumulator.stat.SuccessCount) / float64(accumulator.stat.CallCount) * 100
		}
		if accumulator.stat.SuccessCount > 0 {
			accumulator.stat.AvgLatencyMs = accumulator.latencyTotal / accumulator.stat.SuccessCount
		}
		result = append(result, accumulator.stat)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].TokenCount == result[right].TokenCount {
			return result[left].CallCount > result[right].CallCount
		}
		return result[left].TokenCount > result[right].TokenCount
	})
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

func startOfLocalDay(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}

func (r *gormGatewayRepository) FindAccessToken(candidates []string) (token model.AccessToken, err error) {
	database, err := r.database()
	if err != nil {
		return token, err
	}
	err = errors.New("not found")
	if len(candidates) == 0 {
		return token, err
	}
	raw := strings.TrimSpace(candidates[0])
	var records []model.AccessToken
	if scanErr := database.Model(&model.AccessToken{}).
		Where("status = ?", gatewayStatusNormal).Find(&records).Error; scanErr != nil {
		return token, err
	}
	for _, record := range records {
		if !isEncryptedCredential(record.Token) {
			continue
		}
		value, decryptErr := r.credentialVault().Decrypt(record.Token)
		if decryptErr == nil && value == raw {
			return record, nil
		}
	}
	return token, err
}

func (r *gormGatewayRepository) FindAccessTokenByID(id snowflake.ID) (token model.AccessToken, err error) {
	database, err := r.database()
	if err != nil {
		return token, err
	}
	err = database.Model(&model.AccessToken{}).
		Where("id = ? AND status = ?", id, gatewayStatusNormal).
		First(&token).Error
	return token, err
}

func (r *gormGatewayRepository) AccessTokenCostWindows(tokenID snowflake.ID, now int64) (costs accessTokenCostWindows, err error) {
	database, err := r.database()
	if err != nil {
		return costs, err
	}
	fiveHoursAgo := now - int64(5*60*60*1000)
	dayAgo := now - int64(24*60*60*1000)
	sevenDaysAgo := now - int64(7*24*60*60*1000)
	err = database.Model(&model.CallLog{}).
		Select(`
			COALESCE(SUM(CASE WHEN created_at >= ? THEN cost_micro_usd ELSE 0 END), 0) AS five_hour_micro_usd,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN cost_micro_usd ELSE 0 END), 0) AS day_micro_usd,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN cost_micro_usd ELSE 0 END), 0) AS seven_day_micro_usd
		`, fiveHoursAgo, dayAgo, sevenDaysAgo).
		Where("access_token_id = ? AND success = ? AND created_at >= ?", tokenID, gatewayStatusNormal, sevenDaysAgo).
		Scan(&costs).Error
	return costs, err
}

func (r *gormGatewayRepository) RecordAccessTokenUsage(record AccessTokenUsageRecord) error {
	database, err := r.database()
	if err != nil {
		return err
	}
	record.InputTokens = normalizeStoredTokenCount(record.InputTokens)
	record.OutputTokens = normalizeStoredTokenCount(record.OutputTokens)
	record.ReasoningTokens = normalizeStoredTokenCount(record.ReasoningTokens)
	record.CacheReadTokens = normalizeStoredTokenCount(record.CacheReadTokens)
	record.CacheWriteTokens = normalizeStoredTokenCount(record.CacheWriteTokens)
	if record.EffectiveCostMicroUSD < 0 {
		record.EffectiveCostMicroUSD = 0
	} else if record.EffectiveCostMicroUSD > maxMicroUSDValue {
		record.EffectiveCostMicroUSD = maxMicroUSDValue
	}
	updates := map[string]any{"last_used_at": record.OccurredAt}
	if record.Success {
		updates["success_count"] = gorm.Expr("success_count + 1")
		updates["prompt_tokens_total"] = gorm.Expr("prompt_tokens_total + ?", record.InputTokens)
		updates["completion_tokens_total"] = gorm.Expr("completion_tokens_total + ?", record.OutputTokens)
		updates["reasoning_tokens_total"] = gorm.Expr("reasoning_tokens_total + ?", record.ReasoningTokens)
		updates["cache_read_tokens_total"] = gorm.Expr("cache_read_tokens_total + ?", record.CacheReadTokens)
		updates["cache_write_tokens_total"] = gorm.Expr("cache_write_tokens_total + ?", record.CacheWriteTokens)
		if record.EffectiveCostMicroUSD > 0 {
			costUSD := microUSDToUSD(record.EffectiveCostMicroUSD)
			// 注意：上限与本次费用必须先各自预计算，再作为单个参数参与比较。
			// 若写成 used_micro_usd >= ? - ?（两侧都是参数占位符），PostgreSQL 因无法推断
			// 操作符两侧类型会直接报 42725 operator is not unique: unknown - unknown。
			updates["used_micro_usd"] = gorm.Expr(
				"CASE WHEN used_micro_usd >= ? THEN ? ELSE used_micro_usd + ? END",
				maxMicroUSDValue-record.EffectiveCostMicroUSD, maxMicroUSDValue, record.EffectiveCostMicroUSD,
			)
			updates["used_amount"] = gorm.Expr(
				"CASE WHEN used_amount >= ? THEN ? ELSE used_amount + ? END",
				maxUSDValue-costUSD, maxUSDValue, costUSD,
			)
		}
	} else {
		updates["failure_count"] = gorm.Expr("failure_count + 1")
	}
	return database.Model(&model.AccessToken{}).Where("id = ?", record.TokenID).UpdateColumns(updates).Error
}

func (r *gormGatewayRepository) LoadRouteConfig() (config routeConfig, err error) {
	database, err := r.database()
	if err != nil {
		return config, err
	}
	if err = database.Model(&model.Channel{}).Where("status = ?", gatewayStatusNormal).
		Order("priority DESC, weight DESC, created_at ASC").Find(&config.Channels).Error; err != nil {
		return config, err
	}
	if err = database.Model(&model.Provider{}).Where("status = ?", gatewayStatusNormal).Find(&config.Providers).Error; err != nil {
		return config, err
	}
	if err = database.Model(&model.ChannelAccount{}).Where("status = ?", gatewayStatusNormal).
		Order("created_at ASC").Find(&config.Accounts).Error; err != nil {
		return config, err
	}
	err = database.Model(&model.Proxy{}).Where("status = ?", gatewayStatusNormal).Find(&config.Proxies).Error
	return config, err
}

func (r *gormGatewayRepository) UpdateChannelAccountAPIKey(id snowflake.ID, encrypted string) error {
	database, err := r.database()
	if err != nil {
		return err
	}
	return database.Model(&model.ChannelAccount{}).Where("id = ?", id).UpdateColumn("api_key", encrypted).Error
}

func (r *gormGatewayRepository) UpdateProxyPassword(id snowflake.ID, encrypted string) error {
	database, err := r.database()
	if err != nil {
		return err
	}
	return database.Model(&model.Proxy{}).Where("id = ?", id).UpdateColumn("password", encrypted).Error
}

func (r *gormGatewayRepository) CreateCallLogs(records []model.CallLog) error {
	if len(records) == 0 {
		return nil
	}
	database, err := r.database()
	if err != nil {
		return err
	}
	return database.CreateInBatches(records, len(records)).Error
}

func (r *gormGatewayRepository) Ping() error {
	database, err := r.database()
	if err != nil {
		return err
	}
	var value int
	if err := database.Raw("SELECT 1").Scan(&value).Error; err != nil {
		return err
	}
	if value != 1 {
		return errors.New("数据库健康检查返回异常结果")
	}
	return nil
}
