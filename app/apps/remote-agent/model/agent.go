package model

import (
	"apipig/core/api"
	"apipig/toolkit/snowflake"
)

const (
	// AgentStatusOnline 表示 Agent 已连接且当前可接收新消息。
	AgentStatusOnline = "ONLINE"
	// AgentStatusOffline 表示 Agent 未在心跳超时时间内保持连接。
	AgentStatusOffline = "OFFLINE"
	// AgentStatusBusy 表示 Agent 正在处理一条会话消息。
	AgentStatusBusy = "BUSY"
	// AgentStatusDisabled 表示 Agent 已被管理员禁用，所有运行令牌均失效。
	AgentStatusDisabled = "DISABLED"
)

// Agent 映射 ap_remote_agent 表，保存工作节点配置、身份凭据摘要和最新运行状态。
type Agent struct {
	// MODEL 包含主键、创建/更新时间和软删除字段。
	api.MODEL
	// AgentKey 是控制端与客户端共同使用的唯一节点标识，注册后不允许修改。
	AgentKey string `gorm:"size:100;not null;uniqueIndex" json:"agentKey"`
	// Name 是管理界面展示的 Agent 名称。
	Name string `gorm:"size:100;not null;index" json:"name"`
	// RegistrationTokenHash 是客户端接入注册令牌摘要，不通过接口返回。
	RegistrationTokenHash string `gorm:"size:80;index" json:"-"`
	// TokenHash 是 Agent 注册成功后的运行令牌摘要，用于接口鉴权且不返回明文。
	TokenHash string `gorm:"size:80;index:idx_remote_agent_token_hash_v2" json:"-"`
	// WorkspaceRoot 是该 Agent 创建持久化会话工作区的根目录。
	WorkspaceRoot string `gorm:"size:500;not null;default:''" json:"workspaceRoot"`
	// CodexCommand 是 Agent 执行会话任务时调用的 Codex 命令。
	CodexCommand string `gorm:"size:255;not null;default:'codex'" json:"codexCommand"`
	// CodexArgs 是 Codex 命令参数，以 JSON 序列化后存入文本列。
	CodexArgs     []string `gorm:"type:text;serializer:json" json:"codexArgs"`
	ClaudeCommand string   `gorm:"size:255;not null;default:'claude'" json:"claudeCommand"`
	ClaudeArgs    []string `gorm:"type:text;serializer:json" json:"claudeArgs"`
	// PollWaitSeconds 是客户端长轮询等待新命令的秒数。
	PollWaitSeconds int `gorm:"not null;default:25" json:"pollWaitSeconds"`
	// RequestTimeoutSeconds 是客户端请求控制端接口的超时秒数。
	RequestTimeoutSeconds int `gorm:"not null;default:40" json:"requestTimeoutSeconds"`
	// LogFile 是 Agent 本地日志文件路径。
	LogFile string `gorm:"size:500;not null;default:''" json:"logFile"`
	// IPAddress 是最近一次注册或心跳请求的来源 IP。
	IPAddress string `gorm:"size:64" json:"ipAddress"`
	// Hostname 是客户端注册时上报的主机名，非空表示该配置已完成过注册。
	Hostname string `gorm:"size:255;not null" json:"hostname"`
	// OperatingSystem 是客户端上报的操作系统名称。
	OperatingSystem string `gorm:"size:100" json:"operatingSystem"`
	// Architecture 是客户端上报的 CPU/系统架构。
	Architecture string `gorm:"size:50" json:"architecture"`
	// CPUInfo 是客户端上报的处理器描述信息。
	CPUInfo string `gorm:"size:500" json:"cpuInfo"`
	// CPUUsage 是最近一次心跳上报的 CPU 使用率百分比。
	CPUUsage float64 `gorm:"type:decimal(6,2);not null;default:0" json:"cpuUsage"`
	// MemoryTotal 是客户端物理内存总量，单位为字节。
	MemoryTotal int64 `gorm:"type:bigint;not null;default:0" json:"memoryTotal"`
	// MemoryUsed 是最近一次心跳上报的已用内存，单位为字节。
	MemoryUsed int64 `gorm:"type:bigint;not null;default:0" json:"memoryUsed"`
	// CodexVersion 是客户端当前检测到的 Codex CLI 版本。
	CodexVersion string `gorm:"size:100" json:"codexVersion"`
	// AgentVersion 是远程 Agent 客户端自身版本。
	AgentVersion string `gorm:"size:100" json:"agentVersion"`
	// Status 是节点当前状态，取值为 ONLINE、OFFLINE、BUSY 或 DISABLED。
	Status string `gorm:"size:20;not null;index" json:"status"`
	// CurrentMessageID 是当前正在处理的助手消息 ID，0 表示没有占用任务。
	CurrentMessageID snowflake.ID `gorm:"type:bigint;index" json:"currentMessageId,omitempty" swaggertype:"string"`
	// LastSeenAt 是最近一次注册或心跳成功的毫秒时间戳，用于离线判定。
	LastSeenAt int64 `gorm:"type:bigint;not null;index" json:"lastSeenAt"`
}

// TableName 返回 Agent 模型对应的数据库表名。
func (Agent) TableName() string { return "ap_remote_agent" }
