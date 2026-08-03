package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	AgentStatusOnline   = "ONLINE"
	AgentStatusOffline  = "OFFLINE"
	AgentStatusBusy     = "BUSY"
	AgentStatusDisabled = "DISABLED"
)

// Agent stores the latest identity and liveness snapshot for a worker node.
type Agent struct {
	api.MODEL
	AgentKey        string       `gorm:"size:100;not null;uniqueIndex" json:"agentKey"`
	Name            string       `gorm:"size:100;not null;index" json:"name"`
	TokenHash       string       `gorm:"size:80;not null;uniqueIndex" json:"-"`
	IPAddress       string       `gorm:"size:64" json:"ipAddress"`
	Hostname        string       `gorm:"size:255;not null" json:"hostname"`
	OperatingSystem string       `gorm:"size:100" json:"operatingSystem"`
	Architecture    string       `gorm:"size:50" json:"architecture"`
	CPUInfo         string       `gorm:"size:500" json:"cpuInfo"`
	CPUUsage        float64      `gorm:"type:decimal(6,2);not null;default:0" json:"cpuUsage"`
	MemoryTotal     int64        `gorm:"type:bigint;not null;default:0" json:"memoryTotal"`
	MemoryUsed      int64        `gorm:"type:bigint;not null;default:0" json:"memoryUsed"`
	CodexVersion    string       `gorm:"size:100" json:"codexVersion"`
	AgentVersion    string       `gorm:"size:100" json:"agentVersion"`
	Status          string       `gorm:"size:20;not null;index" json:"status"`
	CurrentTaskID   snowflake.ID `gorm:"type:bigint;index" json:"currentTaskId,omitempty" swaggertype:"string"`
	LastSeenAt      int64        `gorm:"type:bigint;not null;index" json:"lastSeenAt"`
}

func (Agent) TableName() string { return "ap_remote_agent" }
