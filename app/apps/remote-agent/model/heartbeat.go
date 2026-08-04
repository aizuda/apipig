package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

// Heartbeat 映射 Agent 心跳历史表，用于详情页趋势展示和运行问题排查。
type Heartbeat struct {
	// MODEL 包含心跳记录主键和创建时间等公共字段。
	api.MODEL
	// AgentID 是心跳所属的 Agent ID。
	AgentID snowflake.ID `gorm:"type:bigint;not null;index:idx_remote_heartbeat_agent_time,priority:1" json:"agentId" swaggertype:"string"`
	// Status 是产生该心跳时的 Agent 状态。
	Status string `gorm:"size:20;not null" json:"status"`
	// CPUUsage 是该次采样的 CPU 使用率百分比。
	CPUUsage float64 `gorm:"type:decimal(6,2);not null;default:0" json:"cpuUsage"`
	// MemoryUsed 是该次采样的已用内存字节数。
	MemoryUsed int64 `gorm:"type:bigint;not null;default:0" json:"memoryUsed"`
	// OccurredAt 是客户端心跳到达控制端的毫秒时间戳。
	OccurredAt int64 `gorm:"type:bigint;not null;index:idx_remote_heartbeat_agent_time,priority:2" json:"occurredAt"`
}

// TableName 返回心跳模型对应的数据库表名。
func (Heartbeat) TableName() string { return "ap_remote_heartbeat" }
