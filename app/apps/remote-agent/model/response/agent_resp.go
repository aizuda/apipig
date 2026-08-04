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

type AgentCredential struct {
	Agent             remoteModel.Agent `json:"agent"`
	RegistrationToken string            `json:"registrationToken"`
	ConfigYAML        string            `json:"configYaml"`
}

type AgentDetail struct {
	Agent         remoteModel.Agent          `json:"agent"`
	Heartbeats    []remoteModel.Heartbeat    `json:"heartbeats"`
	Conversations []remoteModel.Conversation `json:"conversations"`
}

type ConversationDetail struct {
	Conversation remoteModel.Conversation `json:"conversation"`
	Messages     []remoteModel.Message    `json:"messages"`
}

type SendMessageResult struct {
	UserMessage      remoteModel.Message `json:"userMessage"`
	AssistantMessage remoteModel.Message `json:"assistantMessage"`
}

type CommandDispatch struct {
	CommandID          snowflake.ID `json:"commandId" swaggertype:"string"`
	ConversationID     snowflake.ID `json:"conversationId" swaggertype:"string"`
	UserMessageID      snowflake.ID `json:"userMessageId" swaggertype:"string"`
	AssistantMessageID snowflake.ID `json:"assistantMessageId" swaggertype:"string"`
	Type               string       `json:"type"`
	Prompt             string       `json:"prompt"`
	NextChunkSequence  int64        `json:"nextChunkSequence"`
}

type MessageStreamEvent struct {
	Chunks  []remoteModel.MessageChunk `json:"chunks"`
	Message remoteModel.Message        `json:"message"`
}
