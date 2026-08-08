package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
	"strings"

	"gorm.io/gorm"
)

// Channel 表示一个可参与网关路由的上游渠道账号。
//
// 一个供应商可以关联多个渠道账号；网关按模型、状态、熔断状态、优先级和权重选择渠道。
type Channel struct {
	api.MODEL                   // 通用主键、创建更新信息和软删除标记
	ProviderID     snowflake.ID `gorm:"type:bigint;not null;index" json:"providerId" swaggertype:"string"` // 供应商 ID
	Name           string       `gorm:"size:80;not null;index" json:"name"`                                // 渠道名称
	ModelPricing   string       `gorm:"type:text;not null" json:"modelPricing"`                            // 按上游模型配置的计价规则 JSON
	Priority       int          `gorm:"type:int;not null;default:0" json:"priority"`                       // 路由优先级，数值越大越优先
	Weight         int          `gorm:"type:int;not null;default:1" json:"weight"`                         // 同优先级渠道的加权路由权重
	ProxyID        snowflake.ID `gorm:"type:bigint" json:"proxyId,omitempty" swaggertype:"string"`         // 代理节点 ID，零值表示不使用代理
	RPM            int          `gorm:"type:int;not null;default:0" json:"rpm"`                            // 每分钟请求上限，0 表示不限制
	TPM            int          `gorm:"type:int;not null;default:0" json:"tpm"`                            // 每分钟 token 上限，0 表示不限制
	CostMultiplier float64      `gorm:"type:decimal(12,6);not null;default:1" json:"costMultiplier"`       // 渠道成本倍率，<=0 按 1 处理
	Status         uint         `gorm:"type:smallint;not null;default:1" json:"status"`                    // 状态：1 启用，2 禁用
	Remark         string       `gorm:"size:255" json:"remark"`                                            // 备注
}

// BeforeCreate keeps the JSON default in application code. MySQL does not
// consistently support defaults on TEXT/JSON columns across supported versions.
func (channel *Channel) BeforeCreate(*gorm.DB) error {
	if strings.TrimSpace(channel.ModelPricing) == "" {
		channel.ModelPricing = "[]"
	}
	return nil
}

// TableName 返回渠道账号表名。
func (Channel) TableName() string { return "ap_ai_channel" }
