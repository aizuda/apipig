package request

import (
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"
)

type RegisterRequest struct {
	AgentKey         string       `json:"agentKey"`
	Hostname         string       `json:"hostname"`
	OperatingSystem  string       `json:"operatingSystem"`
	Architecture     string       `json:"architecture"`
	CPUInfo          string       `json:"cpuInfo"`
	MemoryTotal      int64        `json:"memoryTotal"`
	CodexVersion     string       `json:"codexVersion"`
	AgentVersion     string       `json:"agentVersion"`
	CurrentMessageID snowflake.ID `json:"currentMessageId,omitempty" swaggertype:"string"`
}

type RegisterParams struct {
	Request        RegisterRequest
	IPAddress      string
	BootstrapToken string
}

type HeartbeatRequest struct {
	CPUUsage     float64 `json:"cpuUsage"`
	MemoryUsed   int64   `json:"memoryUsed"`
	CodexVersion string  `json:"codexVersion"`
	Busy         bool    `json:"busy"`
}

type HeartbeatParams struct {
	Request    HeartbeatRequest
	AgentToken string
	IPAddress  string
}

type AgentSaveRequest struct {
	ID                    snowflake.ID `json:"id,omitempty" swaggertype:"string"`
	AgentKey              string       `json:"agentKey"`
	Name                  string       `json:"name"`
	WorkspaceRoot         string       `json:"workspaceRoot"`
	CodexCommand          string       `json:"codexCommand"`
	CodexArgs             []string     `json:"codexArgs"`
	PollWaitSeconds       int          `json:"pollWaitSeconds"`
	RequestTimeoutSeconds int          `json:"requestTimeoutSeconds"`
	LogFile               string       `json:"logFile"`
}

type AgentCreateParams struct {
	Request       AgentSaveRequest
	ControllerURL string
}

type AgentRotateTokenParams struct {
	ID            snowflake.ID
	ControllerURL string
}

type AgentPageParams struct {
	coreReq.PageInfo
	Keyword string `json:"keyword"`
	Status  string `json:"status"`
}

func (p *AgentPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}

type AgentStatusRequest struct {
	ID      snowflake.ID `json:"id" swaggertype:"string"`
	Enabled bool         `json:"enabled"`
}

type AgentDeleteRequest struct {
	ID snowflake.ID `json:"id" swaggertype:"string"`
}

type ConversationCreateRequest struct {
	AgentID snowflake.ID `json:"agentId" swaggertype:"string"`
	Title   string       `json:"title"`
}

type ConversationPageParams struct {
	coreReq.PageInfo
	AgentID snowflake.ID `json:"agentId" swaggertype:"string"`
}

func (p *ConversationPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 20}
	}
	return p.PageInfo
}

type ConversationPinRequest struct {
	ID     snowflake.ID `json:"id" swaggertype:"string"`
	Pinned bool         `json:"pinned"`
}

type ConversationRenameRequest struct {
	ID    snowflake.ID `json:"id" swaggertype:"string"`
	Title string       `json:"title"`
}

type ConversationDeleteRequest struct {
	ID snowflake.ID `json:"id" swaggertype:"string"`
}

type SendMessageRequest struct {
	ConversationID snowflake.ID `json:"conversationId" swaggertype:"string"`
	Content        string       `json:"content"`
}

type MessageStreamParams struct {
	MessageID     snowflake.ID `json:"messageId" swaggertype:"string"`
	AfterSequence int64        `json:"afterSequence"`
}

type NextCommandParams struct {
	AgentToken  string
	WaitSeconds int
}

type AcknowledgeCommandParams struct {
	AgentToken string
	CommandID  snowflake.ID
}

type MessageChunkEntry struct {
	Sequence int64  `json:"sequence"`
	Content  string `json:"content"`
}

type MessageChunkUploadRequest struct {
	MessageID snowflake.ID        `json:"messageId" swaggertype:"string"`
	Chunks    []MessageChunkEntry `json:"chunks"`
}

type MessageChunkUploadParams struct {
	AgentToken string
	Request    MessageChunkUploadRequest
}

type MessageResultRequest struct {
	MessageID    snowflake.ID `json:"messageId" swaggertype:"string"`
	Success      bool         `json:"success"`
	Content      string       `json:"content"`
	ErrorMessage string       `json:"errorMessage"`
}

type MessageResultParams struct {
	AgentToken string
	Request    MessageResultRequest
}
