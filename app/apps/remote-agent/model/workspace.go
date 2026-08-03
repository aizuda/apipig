package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// Workspace records an Agent-owned directory allowed for task execution.
type Workspace struct {
	api.MODEL
	AgentID       snowflake.ID `gorm:"type:bigint;not null;uniqueIndex:uk_remote_workspace" json:"agentId" swaggertype:"string"`
	Name          string       `gorm:"size:100;not null" json:"name"`
	Path          string       `gorm:"size:1000;not null;uniqueIndex:uk_remote_workspace" json:"path"`
	RepositoryURL string       `gorm:"size:1000" json:"repositoryUrl"`
	Status        uint         `gorm:"type:smallint;not null;default:1" json:"status"`
}

func (Workspace) TableName() string { return "ap_remote_workspace" }
