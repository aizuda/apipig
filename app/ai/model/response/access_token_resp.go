package response

import (
	"apipig/app/ai/model"
	"apipig/toolkit/snowflake"
)

type AccessTokenPageRecord struct {
	model.AccessToken
	ChannelName  string                 `json:"channelName"`
	ProviderName string                 `json:"providerName"`
	Tags         []model.AccessTokenTag `json:"tags"`
}

// AccessTokenSaveResult 返回访问令牌保存结果。
// Token 仅在系统新建密钥时返回一次，后续查询和编辑不会返回明文或哈希。
type AccessTokenSaveResult struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
}

type AccessTokenSecret struct {
	Token string `json:"token"`
}

// AccessTokenStatistics 汇总指定时间范围内每个 API 密钥的 Token 用量。
type AccessTokenStatistics struct {
	StartAt          int64                        `json:"startAt"`
	EndAt            int64                        `json:"endAt"`
	Total            int64                        `json:"total"`
	Page             int                          `json:"page"`
	PageSize         int                          `json:"pageSize"`
	TokenCount       int64                        `json:"tokenCount"`
	ActiveTokenCount int64                        `json:"activeTokenCount"`
	CallCount        int64                        `json:"callCount"`
	SuccessCount     int64                        `json:"successCount"`
	FailureCount     int64                        `json:"failureCount"`
	TotalTokens      int64                        `json:"totalTokens"`
	ModelStatistics  []AccessTokenModelStatistic  `json:"modelStatistics"`
	Items            []AccessTokenStatisticRecord `json:"items"`
}

// AccessTokenModelStatistic 表示指定密钥和时间范围内单个模型的用量。
type AccessTokenModelStatistic struct {
	Model            string  `json:"model"`
	CallCount        int64   `json:"callCount"`
	SuccessCount     int64   `json:"successCount"`
	FailureCount     int64   `json:"failureCount"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	ReasoningTokens  int64   `json:"reasoningTokens"`
	CacheReadTokens  int64   `json:"cacheReadTokens"`
	CacheWriteTokens int64   `json:"cacheWriteTokens"`
	TotalTokens      int64   `json:"totalTokens"`
	Cost             float64 `json:"cost"`
	AvgLatencyMs     int64   `json:"avgLatencyMs"`
	LastUsedAt       int64   `json:"lastUsedAt"`
}

// AccessTokenStatisticRecord 表示单个 API 密钥的分项 Token 用量。
type AccessTokenStatisticRecord struct {
	TokenID          snowflake.ID `json:"tokenId" swaggertype:"string"`
	TokenName        string       `json:"tokenName"`
	Status           uint         `json:"status"`
	CallCount        int64        `json:"callCount"`
	SuccessCount     int64        `json:"successCount"`
	FailureCount     int64        `json:"failureCount"`
	PromptTokens     int64        `json:"promptTokens"`
	CompletionTokens int64        `json:"completionTokens"`
	ReasoningTokens  int64        `json:"reasoningTokens"`
	CacheReadTokens  int64        `json:"cacheReadTokens"`
	CacheWriteTokens int64        `json:"cacheWriteTokens"`
	TotalTokens      int64        `json:"totalTokens"`
	LastUsedAt       int64        `json:"lastUsedAt"`
}

// AccessTokenCallLogRecord 是 API 密钥授权页可见的调用日志字段。
// 后台路由、客户端 IP 和计价快照等内部信息不会返回给调用方。
type AccessTokenCallLogRecord struct {
	ID                snowflake.ID `json:"id" swaggertype:"string"`
	RequestID         string       `json:"requestId"`
	Model             string       `json:"model"`
	Path              string       `json:"path"`
	Method            string       `json:"method"`
	StatusCode        int          `json:"statusCode"`
	PromptTokens      int          `json:"promptTokens"`
	CompletionTokens  int          `json:"completionTokens"`
	ReasoningTokens   int          `json:"reasoningTokens"`
	CacheReadTokens   int          `json:"cacheReadTokens"`
	CacheWriteTokens  int          `json:"cacheWriteTokens"`
	InputImages       int          `json:"inputImages"`
	OutputImages      int          `json:"outputImages"`
	InputImageTokens  int          `json:"inputImageTokens"`
	OutputImageTokens int          `json:"outputImageTokens"`
	TotalTokens       int          `json:"totalTokens"`
	Cost              float64      `json:"cost"`
	LatencyMs         int64        `json:"latencyMs"`
	Success           uint         `json:"success"`
	ErrorMessage      string       `json:"errorMessage"`
	CreatedAt         int64        `json:"createdAt"`
}
