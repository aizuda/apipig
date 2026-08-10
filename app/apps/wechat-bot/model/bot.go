package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	BotStatusConnecting     = "CONNECTING"
	BotStatusOnline         = "ONLINE"
	BotStatusOffline        = "OFFLINE"
	BotStatusDisabled       = "DISABLED"
	BotStatusSessionExpired = "SESSION_EXPIRED"
	BotStatusError          = "ERROR"
)

// Bot stores one iLink-backed WeChat Bot account. BotToken is always encrypted at rest.
type Bot struct {
	api.MODEL
	Name          string `gorm:"size:100;not null;index" json:"name"`
	BotID         string `gorm:"size:200;not null;uniqueIndex" json:"botId"`
	BotToken      string `gorm:"type:text;not null" json:"-"`
	WebhookKey    string `gorm:"size:80;index" json:"webhookKey,omitempty"`
	WebhookSecret string `gorm:"type:text" json:"-"`
	BaseURL       string `gorm:"size:1000;not null" json:"baseUrl"`
	ILinkUserID   string `gorm:"size:200;not null" json:"iLinkUserId"`
	Status        string `gorm:"size:30;not null;index" json:"status"`
	Enabled       bool   `gorm:"not null;default:true;index" json:"enabled"`
	SyncBuf       string `gorm:"type:text" json:"-"`
	LastError     string `gorm:"size:1000" json:"lastError"`
	MessageCount  int64  `gorm:"type:bigint;not null;default:0" json:"messageCount"`
	LastMessageAt int64  `gorm:"type:bigint;not null;default:0;index" json:"lastMessageAt"`
	ConnectedAt   int64  `gorm:"type:bigint;not null;default:0" json:"connectedAt"`
}

func (Bot) TableName() string { return "ap_wechat_bot" }

// Contact keeps the latest reply context for one external WeChat user.
type Contact struct {
	api.MODEL
	BotRecordID  snowflake.ID `gorm:"type:bigint;not null;uniqueIndex:idx_wechat_bot_contact" json:"botRecordId" swaggertype:"string"`
	UserID       string       `gorm:"size:200;not null;uniqueIndex:idx_wechat_bot_contact" json:"userId"`
	ContextToken string       `gorm:"type:text;not null" json:"-"`
	LastMessage  string       `gorm:"size:500" json:"lastMessage"`
	MessageCount int64        `gorm:"type:bigint;not null;default:0" json:"messageCount"`
	LastActiveAt int64        `gorm:"type:bigint;not null;index" json:"lastActiveAt"`
}

func (Contact) TableName() string { return "ap_wechat_bot_contact" }

// Message stores the minimal text/message metadata required by the management console.
type Message struct {
	api.MODEL
	BotRecordID       snowflake.ID `gorm:"type:bigint;not null;index;uniqueIndex:idx_wechat_bot_message" json:"botRecordId" swaggertype:"string"`
	ExternalMessageID string       `gorm:"size:100;not null;uniqueIndex:idx_wechat_bot_message" json:"externalMessageId"`
	Direction         string       `gorm:"size:10;not null;index;uniqueIndex:idx_wechat_bot_message" json:"direction"`
	UserID            string       `gorm:"size:200;not null;index" json:"userId"`
	Content           string       `gorm:"type:text" json:"content"`
	ContentType       string       `gorm:"size:20;not null" json:"contentType"`
	SessionID         string       `gorm:"size:200" json:"sessionId"`
	GroupID           string       `gorm:"size:200" json:"groupId"`
	OccurredAt        int64        `gorm:"type:bigint;not null;index" json:"occurredAt"`
}

func (Message) TableName() string { return "ap_wechat_bot_message" }
