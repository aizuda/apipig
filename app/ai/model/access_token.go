package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// AccessToken 表示调用方访问 AI 网关时使用的令牌。
//
// 调用方通过 Authorization Bearer 或 X-API-Key 传入令牌，网关据此执行模型授权、
// 请求限流、token 限流、成本额度和有效期校验。
type AccessToken struct {
	api.MODEL                            // 通用主键、创建更新信息和软删除标记
	ChannelID             snowflake.ID   `gorm:"type:bigint;index" json:"channelId,omitempty" swaggertype:"string"` // 可选关联渠道号池
	Name                  string         `gorm:"size:80;not null;index" json:"name"`                                // 令牌名称
	Token                 string         `gorm:"size:255;not null;uniqueIndex" json:"token"`                        // 访问令牌
	Models                string         `gorm:"size:1000" json:"models"`                                           // 允许访问的模型，空值表示不限制
	IpRule                string         `gorm:"type:text" json:"ipRule"`                                           // IP 限制 JSON：enabled、whitelist、blacklist
	RateLimitRule         string         `gorm:"type:text" json:"rateLimitRule"`                                    // 消费速率限制 JSON：5 小时、1 天、7 天额度
	RPM                   int            `gorm:"type:int;not null;default:60" json:"rpm"`                           // 每分钟请求上限
	TPM                   int            `gorm:"type:int;not null;default:0" json:"tpm"`                            // 每分钟 token 上限，0 表示不限制
	QuotaAmount           float64        `gorm:"type:decimal(18,6);not null;default:0" json:"quotaAmount"`          // 可用成本额度，0 表示不限制
	UsedAmount            float64        `gorm:"type:decimal(18,6);not null;default:0" json:"usedAmount"`           // 已使用成本
	QuotaMicroUSD         int64          `gorm:"type:bigint;not null;default:0" json:"-"`                           // 可用成本额度，微美元整数
	UsedMicroUSD          int64          `gorm:"type:bigint;not null;default:0" json:"-"`                           // 已使用成本，微美元整数
	SuccessCount          int64          `gorm:"type:bigint;not null;default:0" json:"successCount"`                // 成功调用累计次数
	FailureCount          int64          `gorm:"type:bigint;not null;default:0" json:"failureCount"`                // 失败调用累计次数
	PromptTokensTotal     int64          `gorm:"type:bigint;not null;default:0" json:"promptTokensTotal"`           // 累计输入 token 数量
	CompletionTokensTotal int64          `gorm:"type:bigint;not null;default:0" json:"completionTokensTotal"`       // 累计输出 token 数量
	ReasoningTokensTotal  int64          `gorm:"type:bigint;not null;default:0" json:"reasoningTokensTotal"`        // 累计推理 token 数量
	CacheReadTokensTotal  int64          `gorm:"type:bigint;not null;default:0" json:"cacheReadTokensTotal"`        // 累计缓存读取 token 数量
	CacheWriteTokensTotal int64          `gorm:"type:bigint;not null;default:0" json:"cacheWriteTokensTotal"`       // 累计缓存写入 token 数量
	LastUsedAt            int64          `gorm:"type:bigint;not null;default:0" json:"lastUsedAt"`                  // 最近一次使用时间，毫秒时间戳；0 表示从未使用
	ExpireAt              int64          `gorm:"type:bigint;not null;default:0" json:"expireAt"`                    // 过期时间，毫秒时间戳，0 表示永不过期
	Status                uint           `gorm:"type:smallint;not null;default:1" json:"status"`                    // 状态：1 启用，2 禁用
	Remark                string         `gorm:"size:255" json:"remark"`                                            // 备注
	TagIDs                []snowflake.ID `gorm:"-" json:"tagIds,omitempty" swaggertype:"array,string"`              // 关联标签 ID
}

// TableName 返回访问令牌表名。
func (AccessToken) TableName() string { return "ap_ai_access_token" }
