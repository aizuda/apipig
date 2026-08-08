package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	MessageRoleUser      = "USER"
	MessageRoleAssistant = "ASSISTANT"

	MessageStatusPending   = "PENDING"
	MessageStatusStreaming = "STREAMING"
	MessageStatusPausing   = "PAUSING"
	MessageStatusPaused    = "PAUSED"
	MessageStatusCompleted = "COMPLETED"
	MessageStatusFailed    = "FAILED"
)

type Message struct {
	api.MODEL
	ConversationID snowflake.ID `gorm:"type:bigint;not null;uniqueIndex:uk_remote_message_sequence,priority:1;index" json:"conversationId" swaggertype:"string"`
	AgentID        snowflake.ID `gorm:"type:bigint;not null;index" json:"agentId" swaggertype:"string"`
	Sequence       int64        `gorm:"type:bigint;not null;uniqueIndex:uk_remote_message_sequence,priority:2" json:"sequence"`
	Role           string       `gorm:"size:20;not null" json:"role"`
	Status         string       `gorm:"size:20;not null;index" json:"status"`
	Content        string       `gorm:"type:text" json:"content"`
	ErrorMessage   string       `gorm:"type:text" json:"errorMessage"`
}

func (Message) TableName() string { return "ap_remote_agent_message" }

type MessageChunk struct {
	api.MODEL
	MessageID snowflake.ID `gorm:"type:bigint;not null;uniqueIndex:uk_remote_message_chunk,priority:1;index" json:"messageId" swaggertype:"string"`
	Sequence  int64        `gorm:"type:bigint;not null;uniqueIndex:uk_remote_message_chunk,priority:2" json:"sequence"`
	Content   string       `gorm:"type:text;not null" json:"content"`
}

func (MessageChunk) TableName() string { return "ap_remote_agent_message_chunk" }
