package service

import (
	"strings"
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversationTurnStreamsPinsAndDeletesMessages(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "console-node", "registration", remoteModel.AgentStatusOnline)
	require.NoError(t, database.Model(&agent).Updates(map[string]any{
		"token_hash": hashAgentToken("runtime-token"), "status": remoteModel.AgentStatusOnline,
		"last_seen_at": time.Now().UnixMilli(),
	}).Error)
	agentService := NewAgentService()
	service := NewConversationService(agentService)
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{
		AgentID: agent.ID, CLIType: remoteModel.CLITypeClaude, WorkingDirectory: "projects/api",
	})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.CLITypeClaude, conversation.CLIType)
	assert.Equal(t, "projects/api", conversation.WorkingDirectory)
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: "SHELL"})
	require.EqualError(t, err, "cliType must be CODEX or CLAUDE")
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex, WorkingDirectory: "../outside"})
	require.EqualError(t, err, "workingDirectory cannot traverse outside workspace-root")
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex, WorkingDirectory: `D:\outside`})
	require.EqualError(t, err, "workingDirectory must be relative to workspace-root")

	turn, err := service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "inspect this workspace"})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.MessageStatusPending, turn.AssistantMessage.Status)
	otherConversation, err := service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, Title: "Other", CLIType: remoteModel.CLITypeCodex})
	require.NoError(t, err)
	assert.Equal(t, ".", otherConversation.WorkingDirectory)
	renamedConversation, err := service.Rename(&remoteReq.ConversationRenameRequest{ID: otherConversation.ID, Title: "  Renamed conversation  "})
	require.NoError(t, err)
	assert.Equal(t, "Renamed conversation", renamedConversation.Title)
	_, err = service.Rename(&remoteReq.ConversationRenameRequest{ID: otherConversation.ID, Title: "   "})
	require.EqualError(t, err, "conversation title is required")
	_, err = service.Rename(&remoteReq.ConversationRenameRequest{ID: otherConversation.ID, Title: strings.Repeat("x", 201)})
	require.EqualError(t, err, "title cannot exceed 200 characters")
	require.NoError(t, database.Model(&remoteModel.Conversation{}).Where("id = ?", conversation.ID).
		Update("last_message_at", time.Now().Add(time.Hour).UnixMilli()).Error)
	pinnedConversation, err := service.Pin(&remoteReq.ConversationPinRequest{ID: otherConversation.ID, Pinned: true})
	require.NoError(t, err)
	assert.True(t, pinnedConversation.Pinned)
	assert.NotZero(t, pinnedConversation.PinnedAt)
	pageResult, err := service.Page(&remoteReq.ConversationPageParams{AgentID: agent.ID})
	require.NoError(t, err)
	pageRecords, ok := pageResult.Records.([]remoteModel.Conversation)
	require.True(t, ok)
	require.Len(t, pageRecords, 2)
	assert.Equal(t, otherConversation.ID, pageRecords[0].ID)

	unpinnedConversation, err := service.Pin(&remoteReq.ConversationPinRequest{ID: otherConversation.ID, Pinned: false})
	require.NoError(t, err)
	assert.False(t, unpinnedConversation.Pinned)
	assert.Zero(t, unpinnedConversation.PinnedAt)
	pageResult, err = service.Page(&remoteReq.ConversationPageParams{AgentID: agent.ID})
	require.NoError(t, err)
	pageRecords, ok = pageResult.Records.([]remoteModel.Conversation)
	require.True(t, ok)
	assert.Equal(t, conversation.ID, pageRecords[0].ID)

	_, err = service.Delete(&remoteReq.ConversationDeleteRequest{ID: conversation.ID})
	require.EqualError(t, err, "cannot delete a conversation while the agent is responding")
	_, err = service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "second turn"})
	require.EqualError(t, err, "agent is not online or is processing another message")

	command, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, command)
	assert.Equal(t, turn.AssistantMessage.ID, command.AssistantMessageID)
	assert.Equal(t, remoteModel.CLITypeClaude, command.CLIType)
	assert.Equal(t, "projects/api", command.WorkingDirectory)
	assert.Contains(t, command.Prompt, "inspect this workspace")

	_, err = service.Acknowledge(&remoteReq.AcknowledgeCommandParams{AgentToken: "runtime-token", CommandID: command.CommandID})
	require.NoError(t, err)
	_, err = service.AppendChunks(&remoteReq.MessageChunkUploadParams{
		AgentToken: "runtime-token",
		Request: remoteReq.MessageChunkUploadRequest{MessageID: command.AssistantMessageID, Chunks: []remoteReq.MessageChunkEntry{
			{Sequence: 1, Content: strings.Repeat("x", maxMessageChunkBytes+1)},
		}},
	})
	require.EqualError(t, err, "message chunk cannot exceed 32 KB")
	_, err = service.AppendChunks(&remoteReq.MessageChunkUploadParams{
		AgentToken: "runtime-token",
		Request: remoteReq.MessageChunkUploadRequest{MessageID: command.AssistantMessageID, Chunks: []remoteReq.MessageChunkEntry{
			{Sequence: 1, Content: "workspace "}, {Sequence: 2, Content: "ready"},
		}},
	})
	require.NoError(t, err)
	_, err = service.Complete(&remoteReq.MessageResultParams{
		AgentToken: "runtime-token",
		Request: remoteReq.MessageResultRequest{
			MessageID: command.AssistantMessageID, Success: true,
			Content: strings.Repeat("x", maxAssistantMessageBytes+1),
		},
	})
	require.EqualError(t, err, "assistant message cannot exceed 1 MB")
	_, err = service.Complete(&remoteReq.MessageResultParams{
		AgentToken: "runtime-token",
		Request:    remoteReq.MessageResultRequest{MessageID: command.AssistantMessageID, Success: true, Content: "workspace ready"},
	})
	require.NoError(t, err)

	rootTurn, err := service.Send(&remoteReq.SendMessageRequest{ConversationID: otherConversation.ID, Content: "start workspace root"})
	require.NoError(t, err)
	rootCommand, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, rootCommand)
	assert.Equal(t, rootTurn.AssistantMessage.ID, rootCommand.AssistantMessageID)
	assert.Equal(t, remoteModel.CLITypeCodex, rootCommand.CLIType)
	assert.Equal(t, ".", rootCommand.WorkingDirectory)
	_, err = service.Complete(&remoteReq.MessageResultParams{
		AgentToken: "runtime-token",
		Request:    remoteReq.MessageResultRequest{MessageID: rootCommand.AssistantMessageID, Success: true, Content: "workspace root ready"},
	})
	require.NoError(t, err)

	detail, err := service.Get(conversation.ID)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 2)
	assert.Equal(t, "workspace ready", detail.Messages[1].Content)
	assert.Equal(t, remoteModel.MessageStatusCompleted, detail.Messages[1].Status)
	var storedAgent remoteModel.Agent
	require.NoError(t, database.First(&storedAgent, agent.ID).Error)
	assert.Equal(t, remoteModel.AgentStatusOnline, storedAgent.Status)
	assert.Zero(t, storedAgent.CurrentMessageID)

	deleted, err := service.Delete(&remoteReq.ConversationDeleteRequest{ID: conversation.ID})
	require.NoError(t, err)
	assert.True(t, deleted)
	for _, deletedData := range []struct {
		model any
		where string
		value any
	}{
		{model: &remoteModel.Conversation{}, where: "id = ?", value: conversation.ID},
		{model: &remoteModel.Message{}, where: "conversation_id = ?", value: conversation.ID},
		{model: &remoteModel.MessageChunk{}, where: "message_id = ?", value: turn.AssistantMessage.ID},
		{model: &remoteModel.Command{}, where: "conversation_id = ?", value: conversation.ID},
	} {
		var count int64
		require.NoError(t, database.Unscoped().Model(deletedData.model).Where(deletedData.where, deletedData.value).Count(&count).Error)
		assert.Zero(t, count)
	}
	var remainingConversationCount int64
	require.NoError(t, database.Model(&remoteModel.Conversation{}).Where("id = ?", otherConversation.ID).Count(&remainingConversationCount).Error)
	assert.EqualValues(t, 1, remainingConversationCount)
}

func TestConversationRejectsIncompleteExecutionConfiguration(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "invalid-conversation-node", "registration", remoteModel.AgentStatusOnline)
	require.NoError(t, database.Model(&agent).Updates(map[string]any{
		"token_hash": hashAgentToken("runtime-token"), "last_seen_at": time.Now().UnixMilli(),
	}).Error)
	service := NewConversationService(NewAgentService())
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{
		AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex,
	})
	require.NoError(t, err)
	require.NoError(t, database.Model(&remoteModel.Conversation{}).Where("id = ?", conversation.ID).
		Update("working_directory", "").Error)

	_, err = service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "run"})
	require.EqualError(t, err, "conversation workingDirectory is required")

	var messageCount int64
	require.NoError(t, database.Model(&remoteModel.Message{}).Where("conversation_id = ?", conversation.ID).Count(&messageCount).Error)
	assert.Zero(t, messageCount)
}

func TestBuildConversationPromptPreservesInstructionAndLatestMessage(t *testing.T) {
	messages := []remoteModel.Message{{
		Role: remoteModel.MessageRoleAssistant, Status: remoteModel.MessageStatusCompleted,
		Content: strings.Repeat("old-output", maxConversationPromptBytes),
	}}
	latest := strings.Repeat("latest-request", 8*1024)

	prompt := buildConversationPrompt(messages, latest)

	assert.LessOrEqual(t, len(prompt), maxConversationPromptBytes)
	assert.True(t, strings.HasPrefix(prompt, "Continue the following coding-agent conversation"))
	assert.True(t, strings.HasSuffix(prompt, "User:\n"+latest))
}
