package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// CallLog 表示一次 AI 网关调用的审计记录。
//
// 日志记录调用方、实际路由目标、token 消耗、成本、延迟和错误信息，
// 用于调用追踪、成本统计、故障排查和运营分析。
type CallLog struct {
	api.MODEL                         // 通用主键、创建更新信息和软删除标记
	RequestID            string       `gorm:"size:80;not null;index" json:"requestId"`                               // 网关请求 ID
	AccessTokenID        snowflake.ID `gorm:"type:bigint;index" json:"accessTokenId,omitempty" swaggertype:"string"` // 调用方访问令牌 ID
	ProviderID           snowflake.ID `gorm:"type:bigint;index" json:"providerId,omitempty" swaggertype:"string"`    // 实际供应商 ID
	ChannelID            snowflake.ID `gorm:"type:bigint;index" json:"channelId,omitempty" swaggertype:"string"`     // 实际渠道账号 ID
	Model                string       `gorm:"size:120;index" json:"model"`                                           // 请求模型名称
	Path                 string       `gorm:"size:255" json:"path"`                                                  // 上游请求路径
	Method               string       `gorm:"size:20" json:"method"`                                                 // HTTP 方法
	ClientIP             string       `gorm:"size:80;index" json:"clientIp"`                                         // 调用方 IP
	StatusCode           int          `gorm:"type:int;not null;default:0" json:"statusCode"`                         // 上游 HTTP 状态码
	PromptTokens         int          `gorm:"type:int;not null;default:0" json:"promptTokens"`                       // 输入 token 数量
	CompletionTokens     int          `gorm:"type:int;not null;default:0" json:"completionTokens"`                   // 输出 token 数量
	ReasoningTokens      int          `gorm:"type:int;not null;default:0" json:"reasoningTokens"`                    // 推理 token 数量，通常已包含在输出 token 中
	CacheReadTokens      int          `gorm:"type:int;not null;default:0" json:"cacheReadTokens"`                    // 缓存读取 token 数量
	CacheWriteTokens     int          `gorm:"type:int;not null;default:0" json:"cacheWriteTokens"`                   // 缓存写入 token 数量
	InputImages          int          `gorm:"type:int;not null;default:0" json:"inputImages"`                        // 输入图片数量
	OutputImages         int          `gorm:"type:int;not null;default:0" json:"outputImages"`                       // 输出图片数量
	InputImageTokens     int          `gorm:"type:int;not null;default:0" json:"inputImageTokens"`                   // 输入图片 token 数量
	OutputImageTokens    int          `gorm:"type:int;not null;default:0" json:"outputImageTokens"`                  // 输出图片 token 数量
	TotalTokens          int          `gorm:"type:int;not null;default:0" json:"totalTokens"`                        // 总 token 数量
	PricingModel         string       `gorm:"size:160" json:"pricingModel"`                                          // 命中的计价模型
	PricingSnapshot      string       `gorm:"type:text" json:"pricingSnapshot"`                                      // 计价规则快照
	StandardCost         float64      `gorm:"type:decimal(18,8);not null;default:0" json:"standardCost"`             // 倍率前标准成本，仅用于展示
	Cost                 float64      `gorm:"type:decimal(18,8);not null;default:0" json:"cost"`                     // 渠道倍率后的有效成本
	StandardCostMicroUSD int64        `gorm:"type:bigint;not null;default:0" json:"standardCostMicroUsd"`            // 倍率前标准成本，微美元
	CostMicroUSD         int64        `gorm:"type:bigint;not null;default:0" json:"costMicroUsd"`                    // 有效成本，微美元
	CostMultiplier       float64      `gorm:"type:decimal(12,6);not null;default:1" json:"costMultiplier"`           // 调用时渠道倍率快照
	LatencyMs            int64        `gorm:"type:bigint;not null;default:0" json:"latencyMs"`                       // 上游调用延迟，单位毫秒
	Success              uint         `gorm:"type:smallint;not null;default:1;index" json:"success"`                 // 调用结果：1 成功，2 失败
	ErrorMessage         string       `gorm:"size:1000" json:"errorMessage"`                                         // 错误信息或上游错误响应摘要
}

// TableName 返回调用日志表名。
func (CallLog) TableName() string { return "ap_ai_call_log" }
