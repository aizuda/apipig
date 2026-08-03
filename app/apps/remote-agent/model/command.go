package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	CommandTypeExecuteTask = "EXECUTE_TASK"
	CommandTypeStopTask    = "STOP_TASK"

	CommandStatusPending      = "PENDING"
	CommandStatusDispatched   = "DISPATCHED"
	CommandStatusAcknowledged = "ACKNOWLEDGED"
	CommandStatusCompleted    = "COMPLETED"
	CommandStatusFailed       = "FAILED"
	CommandStatusCancelled    = "CANCELLED"
)

// Command is an auditable Controller instruction; it is not an arbitrary shell command.
type Command struct {
	api.MODEL
	TaskID         snowflake.ID `gorm:"type:bigint;not null;index" json:"taskId" swaggertype:"string"`
	AgentID        snowflake.ID `gorm:"type:bigint;not null;index" json:"agentId" swaggertype:"string"`
	Type           string       `gorm:"size:30;not null;index" json:"type"`
	Payload        string       `gorm:"type:text" json:"payload"`
	Status         string       `gorm:"size:20;not null;index" json:"status"`
	Attempt        int          `gorm:"not null;default:0" json:"attempt"`
	DispatchedAt   int64        `gorm:"type:bigint;not null;default:0" json:"dispatchedAt"`
	AcknowledgedAt int64        `gorm:"type:bigint;not null;default:0" json:"acknowledgedAt"`
}

func (Command) TableName() string { return "ap_remote_command" }
