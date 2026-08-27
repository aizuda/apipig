package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// ChannelAccount 表示上游渠道对应的平台注册账户。
type ChannelAccount struct {
	api.MODEL              // 通用主键、创建更新信息和软删除标记
	ChannelID snowflake.ID `gorm:"type:bigint;not null;index" json:"channelId" swaggertype:"string"` // 所属渠道号池 ID
	Name      string       `gorm:"size:80;not null;index" json:"name"`                               // 渠道账号名称
	APIKey    string       `gorm:"size:4096" json:"apiKey"`                                          // 上游平台 API Key，持久化时加密保存
	Models    string       `gorm:"size:1000;not null;default:'{}'" json:"models"`                    // 网关模型到上游模型的 JSON 映射
	Status    uint         `gorm:"type:smallint;not null;default:1" json:"status"`                   // 状态：1 启用，2 禁用
	Remark    string       `gorm:"size:255" json:"remark"`                                           // 备注
}

// TableName 返回渠道账户表名。
func (ChannelAccount) TableName() string { return "ap_ai_channel_account" }
