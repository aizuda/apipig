package model

import "apipig/core/api"

// Provider 表示大模型供应商配置。
//
// 供应商只保存厂商级别的协议、基础地址和模型能力；
// 真实的上游密钥由渠道账号表维护，从而支持一个供应商配置多个账号池。
type Provider struct {
	api.MODEL        // 通用主键、创建更新信息和软删除标记
	Name      string `gorm:"size:80;not null;index" json:"name"`               // 供应商名称
	Code      string `gorm:"size:50;not null;uniqueIndex" json:"code"`         // 供应商编码，系统内部唯一标识
	Icon      string `gorm:"size:50" json:"icon"`                              // SVG 品牌图标标识
	Protocol  string `gorm:"size:30;not null;default:openai" json:"protocol"`  // 协议类型：openai、anthropic、codex、grok、gemini、qwen、custom
	BaseURL   string `gorm:"size:255;not null" json:"baseUrl"`                 // API 基础地址，建议保存到兼容协议的 /v1 层级
	Models    string `gorm:"size:1000" json:"models"`                          // 支持模型，多个模型使用英文逗号分隔
	TimeoutMs int    `gorm:"type:int;not null;default:60000" json:"timeoutMs"` // 上游请求超时时间，单位毫秒
	Status    uint   `gorm:"type:smallint;not null;default:1" json:"status"`   // 状态：1 启用，2 禁用
	Remark    string `gorm:"size:255" json:"remark"`                           // 备注
}

// TableName 返回供应商表名。
func (Provider) TableName() string { return "ap_ai_provider" }
