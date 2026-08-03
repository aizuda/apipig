package request

import (
	remoteModel "apipig/app/apps/remote-agent/model"
	coreReq "apipig/core/api/request"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

type RegisterRequest struct {
	AgentKey        string       `json:"agentKey"`
	Name            string       `json:"name"`
	Hostname        string       `json:"hostname"`
	OperatingSystem string       `json:"operatingSystem"`
	Architecture    string       `json:"architecture"`
	CPUInfo         string       `json:"cpuInfo"`
	MemoryTotal     int64        `json:"memoryTotal"`
	CodexVersion    string       `json:"codexVersion"`
	AgentVersion    string       `json:"agentVersion"`
	CurrentTaskID   snowflake.ID `json:"currentTaskId,omitempty" swaggertype:"string"`
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

type TaskCreateRequest struct {
	Name          string       `json:"name"`
	AgentID       snowflake.ID `json:"agentId" swaggertype:"string"`
	RepositoryURL string       `json:"repositoryUrl"`
	WorkingDir    string       `json:"workingDir"`
	Prompt        string       `json:"prompt"`
}

type TaskCreateParams struct {
	Ctx     *fiber.Ctx
	Request TaskCreateRequest
}

type TaskPageParams struct {
	coreReq.PageInfo
	AgentID snowflake.ID `json:"agentId" swaggertype:"string"`
	Status  string       `json:"status"`
	Keyword string       `json:"keyword"`
}

func (p *TaskPageParams) GetPageInfo() coreReq.PageInfo {
	if p == nil {
		return coreReq.PageInfo{Page: 1, PageSize: 10}
	}
	return p.PageInfo
}

type CancelTaskParams struct {
	Ctx *fiber.Ctx
	ID  snowflake.ID
}

type NextCommandParams struct {
	AgentToken  string
	WaitSeconds int
}

type AcknowledgeCommandRequest struct {
	CommandID snowflake.ID `json:"commandId" swaggertype:"string"`
}

type AcknowledgeCommandParams struct {
	AgentToken string
	CommandID  snowflake.ID
}

type TaskResultRequest struct {
	TaskID       snowflake.ID `json:"taskId" swaggertype:"string"`
	Success      bool         `json:"success"`
	Result       string       `json:"result"`
	ErrorMessage string       `json:"errorMessage"`
	ChangedFiles []string     `json:"changedFiles"`
}

type TaskResultParams struct {
	AgentToken string
	Request    TaskResultRequest
}

type TaskLogEntry struct {
	Sequence int64  `json:"sequence"`
	Stream   string `json:"stream"`
	Content  string `json:"content"`
}

type TaskLogUploadRequest struct {
	TaskID snowflake.ID   `json:"taskId" swaggertype:"string"`
	Logs   []TaskLogEntry `json:"logs"`
}

type TaskLogUploadParams struct {
	AgentToken string
	Request    TaskLogUploadRequest
}

type TaskLogStreamParams struct {
	TaskID        snowflake.ID `json:"taskId" swaggertype:"string"`
	AfterSequence int64        `json:"afterSequence"`
}

type ClaimedCommand struct {
	Command         remoteModel.Command
	Task            remoteModel.Task
	Workspace       remoteModel.Workspace
	NextLogSequence int64
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
