package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	TaskStatusPending   = "PENDING"
	TaskStatusRunning   = "RUNNING"
	TaskStatusSuccess   = "SUCCESS"
	TaskStatusFailed    = "FAILED"
	TaskStatusCancelled = "CANCELLED"
)

// Task stores a remote Codex execution request and its final result.
type Task struct {
	api.MODEL
	Name          string       `gorm:"size:200;not null;index" json:"name"`
	AgentID       snowflake.ID `gorm:"type:bigint;not null;index" json:"agentId" swaggertype:"string"`
	WorkspaceID   snowflake.ID `gorm:"type:bigint;index" json:"workspaceId,omitempty" swaggertype:"string"`
	RepositoryURL string       `gorm:"size:1000;not null" json:"repositoryUrl"`
	WorkingDir    string       `gorm:"size:1000;not null" json:"workingDir"`
	Prompt        string       `gorm:"type:text;not null" json:"prompt"`
	Status        string       `gorm:"size:20;not null;index" json:"status"`
	Result        string       `gorm:"type:text" json:"result"`
	ErrorMessage  string       `gorm:"type:text" json:"errorMessage"`
	ChangedFiles  string       `gorm:"type:text" json:"changedFiles"`
	StartedAt     int64        `gorm:"type:bigint;not null;default:0" json:"startedAt"`
	FinishedAt    int64        `gorm:"type:bigint;not null;default:0" json:"finishedAt"`
}

func (Task) TableName() string { return "ap_remote_task" }
