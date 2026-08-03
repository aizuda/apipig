package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// Heartbeat stores node telemetry history for Agent detail views and diagnostics.
type Heartbeat struct {
	api.MODEL
	AgentID    snowflake.ID `gorm:"type:bigint;not null;index:idx_remote_heartbeat_agent_time,priority:1" json:"agentId" swaggertype:"string"`
	Status     string       `gorm:"size:20;not null" json:"status"`
	CPUUsage   float64      `gorm:"type:decimal(6,2);not null;default:0" json:"cpuUsage"`
	MemoryUsed int64        `gorm:"type:bigint;not null;default:0" json:"memoryUsed"`
	OccurredAt int64        `gorm:"type:bigint;not null;index:idx_remote_heartbeat_agent_time,priority:2" json:"occurredAt"`
}

func (Heartbeat) TableName() string { return "ap_remote_heartbeat" }
