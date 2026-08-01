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
// Token 仅在系统新建密钥时返回一次，后续查询和编辑不会再返回明文。
type AccessTokenSaveResult struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
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
	Items            []AccessTokenStatisticRecord `json:"items"`
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
