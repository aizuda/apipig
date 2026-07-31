package response

import "apipig/toolkit/snowflake"

// GatewaySummary 表示 AI 网关管理首页的跨表汇总数据。
type GatewaySummary struct {
	GatewaySummaryAnalytics
	ProviderCount    int64   `json:"providerCount"`    // 供应商数量
	ChannelCount     int64   `json:"channelCount"`     // 渠道账号数量
	TokenCount       int64   `json:"tokenCount"`       // 访问令牌数量
	ProxyCount       int64   `json:"proxyCount"`       // 代理节点数量
	CallCount        int64   `json:"callCount"`        // 总调用次数
	SuccessCount     int64   `json:"successCount"`     // 成功调用次数
	ErrorCount       int64   `json:"errorCount"`       // 失败调用次数
	TotalTokens      int64   `json:"totalTokens"`      // 总 token 消耗
	PromptTokens     int64   `json:"promptTokens"`     // 可计费输入 token
	CompletionTokens int64   `json:"completionTokens"` // 输出 token
	ReasoningTokens  int64   `json:"reasoningTokens"`  // 推理 token
	CacheReadTokens  int64   `json:"cacheReadTokens"`  // 缓存读取 token
	CacheWriteTokens int64   `json:"cacheWriteTokens"` // 缓存写入 token
	StandardCost     float64 `json:"standardCost"`     // 渠道倍率前标准成本
	TotalCost        float64 `json:"totalCost"`        // 渠道倍率后有效成本
}

type GatewaySummaryAnalytics struct {
	ModelDistribution []GatewayModelDistribution `json:"modelDistribution"`
	TokenTrend        []GatewayTokenTrend        `json:"tokenTrend"`
	ChannelStatistics []GatewayChannelStatistic  `json:"channelStatistics"`
}

type GatewayChannelStatistic struct {
	ChannelID    snowflake.ID `json:"channelId" swaggertype:"string"`
	ChannelName  string       `json:"channelName"`
	CallCount    int64        `json:"callCount"`
	SuccessCount int64        `json:"successCount"`
	SuccessRate  float64      `json:"successRate"`
	TokenCount   int64        `json:"tokenCount"`
	Cost         float64      `json:"cost"`
	AvgLatencyMs int64        `json:"avgLatencyMs"`
}

type GatewayModelDistribution struct {
	Model      string  `json:"model"`
	CallCount  int64   `json:"callCount"`
	TokenCount int64   `json:"tokenCount"`
	Cost       float64 `json:"cost"`
}

type GatewayTokenTrend struct {
	Date             string `json:"date"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
	CacheTokens      int64  `json:"cacheTokens"`
	TotalTokens      int64  `json:"totalTokens"`
}
