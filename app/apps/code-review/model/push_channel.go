package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	PushChannelTypeWeCom     = "wecom_robot"
	PushChannelTypeDingTalk  = "dingtalk_robot"
	PushChannelTypeWechatBot = "wechat_bot"
)

// PushChannel stores one project notification target. Config is encrypted JSON.
type PushChannel struct {
	api.MODEL
	ProjectID snowflake.ID `gorm:"type:bigint;not null;index" json:"projectId" swaggertype:"string"`
	Type      string       `gorm:"size:40;not null;index" json:"type"`
	Name      string       `gorm:"size:100;not null" json:"name"`
	Enabled   bool         `gorm:"not null;default:true;index" json:"enabled"`
	Config    string       `gorm:"type:text;not null" json:"-"`
}

func (PushChannel) TableName() string { return "ap_review_push_channel" }

// PushChannelView is the safe API representation. Config never contains raw secrets.
type PushChannelView struct {
	ID               snowflake.ID      `json:"id" swaggertype:"string"`
	ProjectID        snowflake.ID      `json:"projectId" swaggertype:"string"`
	Type             string            `json:"type"`
	Name             string            `json:"name"`
	Enabled          bool              `json:"enabled"`
	Config           map[string]string `json:"config"`
	SecretConfigured bool              `json:"secretConfigured"`
}
