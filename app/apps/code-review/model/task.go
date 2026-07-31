package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusIgnored   = "ignored"
)

// Task 保存一次 Push 或 Pull Request/Merge Request 的评审执行记录。
type Task struct {
	api.MODEL
	ProjectID      snowflake.ID `gorm:"type:bigint;not null;index;uniqueIndex:uk_review_event" json:"projectId" swaggertype:"string"`
	EventKey       string       `gorm:"size:255;not null;uniqueIndex:uk_review_event" json:"eventKey"`
	EventType      string       `gorm:"size:30;not null;index" json:"eventType"`
	Provider       string       `gorm:"size:20;not null" json:"provider"`
	RepositoryName string       `gorm:"size:300" json:"repositoryName"`
	RepositoryURL  string       `gorm:"size:1000" json:"repositoryUrl"`
	Ref            string       `gorm:"size:500" json:"ref"`
	BaseSHA        string       `gorm:"size:64" json:"baseSha"`
	HeadSHA        string       `gorm:"size:64;not null;index" json:"headSha"`
	Author         string       `gorm:"size:200" json:"author"`
	Title          string       `gorm:"size:500" json:"title"`
	Status         string       `gorm:"size:20;not null;index" json:"status"`
	Attempt        int          `gorm:"not null;default:0" json:"attempt"`
	ChangedFiles   int          `gorm:"not null;default:0" json:"changedFiles"`
	Additions      int          `gorm:"not null;default:0" json:"additions"`
	Deletions      int          `gorm:"not null;default:0" json:"deletions"`
	RiskLevel      string       `gorm:"size:20" json:"riskLevel"`
	Summary        string       `gorm:"type:text" json:"summary"`
	Report         string       `gorm:"type:text" json:"report"`
	Findings       string       `gorm:"type:text" json:"findings"`
	DiffTruncated  bool         `gorm:"not null;default:false" json:"diffTruncated"`
	ErrorMessage   string       `gorm:"type:text" json:"errorMessage"`
	StartedAt      int64        `gorm:"type:bigint;not null;default:0" json:"startedAt"`
	FinishedAt     int64        `gorm:"type:bigint;not null;default:0" json:"finishedAt"`
}

func (Task) TableName() string { return "ap_review_task" }
