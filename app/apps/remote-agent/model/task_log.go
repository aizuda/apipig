package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// TaskLog stores an ordered stdout/stderr fragment from a remote execution.
type TaskLog struct {
	api.MODEL
	TaskID   snowflake.ID `gorm:"type:bigint;not null;index:idx_remote_log_order,priority:1;uniqueIndex:uk_remote_log_sequence" json:"taskId" swaggertype:"string"`
	AgentID  snowflake.ID `gorm:"type:bigint;not null;index" json:"agentId" swaggertype:"string"`
	Sequence int64        `gorm:"type:bigint;not null;index:idx_remote_log_order,priority:2;uniqueIndex:uk_remote_log_sequence" json:"sequence"`
	Stream   string       `gorm:"size:10;not null" json:"stream"`
	Content  string       `gorm:"type:text;not null" json:"content"`
}

func (TaskLog) TableName() string { return "ap_remote_task_log" }
