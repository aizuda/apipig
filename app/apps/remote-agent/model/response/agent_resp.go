package response

import (
	remoteModel "apipig/app/apps/remote-agent/model"
	"apipig/toolkit/snowflake"
)

type RegisterResult struct {
	AgentID                  snowflake.ID `json:"agentId" swaggertype:"string"`
	AgentToken               string       `json:"agentToken"`
	Status                   string       `json:"status"`
	HeartbeatIntervalSeconds int          `json:"heartbeatIntervalSeconds"`
}

type HeartbeatResult struct {
	AgentID    snowflake.ID `json:"agentId" swaggertype:"string"`
	Status     string       `json:"status"`
	ServerTime int64        `json:"serverTime"`
}

type AgentDetail struct {
	Agent      remoteModel.Agent       `json:"agent"`
	Heartbeats []remoteModel.Heartbeat `json:"heartbeats"`
	Tasks      []remoteModel.Task      `json:"tasks"`
}

type CommandDispatch struct {
	CommandID       snowflake.ID `json:"commandId" swaggertype:"string"`
	Type            string       `json:"type"`
	TaskID          snowflake.ID `json:"taskId" swaggertype:"string"`
	TaskName        string       `json:"taskName"`
	RepositoryURL   string       `json:"repositoryUrl"`
	WorkingDir      string       `json:"workingDir"`
	Prompt          string       `json:"prompt"`
	NextLogSequence int64        `json:"nextLogSequence"`
}

type TaskDetail struct {
	Task      remoteModel.Task      `json:"task"`
	Workspace remoteModel.Workspace `json:"workspace"`
	Commands  []remoteModel.Command `json:"commands"`
	Logs      []remoteModel.TaskLog `json:"logs"`
}
