package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	ConversationStatusActive      = "ACTIVE"
	ConversationControlModeWeb    = "WEB"
	ConversationControlModeWechat = "WECHAT"
	CLITypeCodex                  = "CODEX"
	CLITypeClaude                 = "CLAUDE"
)

// Conversation is a persistent, agent-scoped Web console session.
type Conversation struct {
	api.MODEL
	AgentID          snowflake.ID `gorm:"type:bigint;not null;index" json:"agentId" swaggertype:"string"`
	Title            string       `gorm:"size:200;not null;index" json:"title"`
	CLIType          string       `gorm:"size:20;not null" json:"cliType"`
	WorkingDirectory string       `gorm:"size:500;not null" json:"workingDirectory"`
	Status           string       `gorm:"size:20;not null;index" json:"status"`
	Pinned           bool         `gorm:"not null;default:false;index" json:"pinned"`
	PinnedAt         int64        `gorm:"type:bigint;not null;default:0;index" json:"pinnedAt"`
	LastMessageAt    int64        `gorm:"type:bigint;not null;index" json:"lastMessageAt"`
	ControlMode      string       `gorm:"size:20;not null;default:WEB;index" json:"controlMode"`
	WechatBotID      snowflake.ID `gorm:"type:bigint;not null;default:0;index" json:"wechatBotId,omitempty" swaggertype:"string"`
	WechatUserID     string       `gorm:"size:200;not null;default:''" json:"wechatUserId,omitempty"`
	WechatTakeoverAt int64        `gorm:"type:bigint;not null;default:0" json:"wechatTakeoverAt,omitempty"`
}

func (Conversation) TableName() string { return "ap_remote_agent_conversation" }
