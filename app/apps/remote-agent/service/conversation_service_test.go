package service

import (
	"context"
	"strings"
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWechatTakeoverBlocksWebAndRoutesMessages(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "takeover-node", "registration", remoteModel.AgentStatusOnline)
	require.NoError(t, database.Model(&agent).Updates(map[string]any{
		"token_hash": hashAgentToken("runtime-token"), "status": remoteModel.AgentStatusOnline,
		"last_seen_at": time.Now().UnixMilli(),
	}).Error)
	service := NewConversationService(NewAgentService())
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.PermissionModeFullAccess, conversation.PermissionMode)
	botID := snowflake.ID(9001)
	require.NoError(t, service.repository.SetTakeover(conversation.ID, botID, "wx-user", time.Now().UnixMilli()))

	_, err = service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "from web"})
	require.EqualError(t, err, "该会话已由微信 Bot 接管，Web 端已暂停控制")
	require.NoError(t, service.HandleWechatInbound(botID, "other-user", "ignored"))
	require.NoError(t, service.HandleWechatInbound(botID, "wx-user", "from wechat"))

	command, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, command)
	assert.Contains(t, command.Prompt, "from wechat")
	var sentBotID snowflake.ID
	var sentUserID, sentContent string
	service.SetTakeoverSender(func(_ context.Context, botID snowflake.ID, userID, content string) error {
		sentBotID, sentUserID, sentContent = botID, userID, content
		return nil
	})
	_, err = service.Complete(&remoteReq.MessageResultParams{AgentToken: "runtime-token", Request: remoteReq.MessageResultRequest{
		MessageID: command.AssistantMessageID, Success: true, Content: "reply to wechat",
	}})
	require.NoError(t, err)
	assert.Equal(t, botID, sentBotID)
	assert.Equal(t, "wx-user", sentUserID)
	assert.Equal(t, "reply to wechat", sentContent)

	stopped, err := service.StopTakeover(&remoteReq.ConversationTakeoverRequest{ID: conversation.ID})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.ConversationControlModeWeb, stopped.ControlMode)
}

func TestWechatTakeoverMirrorsBotConsoleOutboundMessage(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "takeover-mirror-node", "registration", remoteModel.AgentStatusOnline)
	service := NewConversationService(NewAgentService())
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex})
	require.NoError(t, err)
	botID := snowflake.ID(9002)
	require.NoError(t, service.repository.SetTakeover(conversation.ID, botID, "wx-user", time.Now().UnixMilli()))

	require.NoError(t, service.HandleWechatOutbound(botID, "wx-user", "sent from bot console"))
	detail, err := service.Get(conversation.ID)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
	assert.Equal(t, remoteModel.MessageRoleAssistant, detail.Messages[0].Role)
	assert.Equal(t, remoteModel.MessageStatusCompleted, detail.Messages[0].Status)
	assert.Equal(t, "wechat-bot", detail.Messages[0].CreatedBy)
	assert.Equal(t, "sent from bot console", detail.Messages[0].Content)
}

func TestWechatTakeoverNotifiesBotWhenAgentDisconnects(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "takeover-offline-node", "registration", remoteModel.AgentStatusOnline)
	require.NoError(t, database.Model(&agent).Updates(map[string]any{
		"token_hash": hashAgentToken("runtime-token"), "status": remoteModel.AgentStatusOnline,
		"last_seen_at": time.Now().UnixMilli(),
	}).Error)
	agentService := NewAgentService()
	service := NewConversationService(agentService)
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex})
	require.NoError(t, err)
	botID := snowflake.ID(9003)
	require.NoError(t, service.repository.SetTakeover(conversation.ID, botID, "wx-user", time.Now().UnixMilli()))

	var sentBotID snowflake.ID
	var sentUserID, sentContent string
	service.SetTakeoverSender(func(_ context.Context, botID snowflake.ID, userID, content string) error {
		sentBotID, sentUserID, sentContent = botID, userID, content
		return nil
	})
	disconnected, err := agentService.Disconnect("runtime-token")
	require.NoError(t, err)
	assert.True(t, disconnected)
	assert.Equal(t, botID, sentBotID)
	assert.Equal(t, "wx-user", sentUserID)
	assert.Contains(t, sentContent, "已离线")

	detail, err := service.Get(conversation.ID)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
	assert.Equal(t, "system", detail.Messages[0].CreatedBy)
	assert.Equal(t, sentContent, detail.Messages[0].Content)
}

func TestWechatTakeoverMirrorsInboundAndRepliesWhileAgentIsOffline(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "takeover-already-offline-node", "registration", remoteModel.AgentStatusOffline)
	agentService := NewAgentService()
	service := NewConversationService(agentService)
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex})
	require.NoError(t, err)
	botID := snowflake.ID(9004)
	require.NoError(t, service.repository.SetTakeover(conversation.ID, botID, "wx-user", time.Now().UnixMilli()))

	var sentContent string
	service.SetTakeoverSender(func(_ context.Context, _ snowflake.ID, _ string, content string) error {
		sentContent = content
		return nil
	})
	require.NoError(t, service.HandleWechatInbound(botID, "wx-user", "is anyone there?"))
	assert.Contains(t, sentContent, "已离线")

	detail, err := service.Get(conversation.ID)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 2)
	assert.Equal(t, remoteModel.MessageRoleUser, detail.Messages[0].Role)
	assert.Equal(t, "wechat-bot", detail.Messages[0].CreatedBy)
	assert.Equal(t, "is anyone there?", detail.Messages[0].Content)
	assert.Equal(t, remoteModel.MessageRoleAssistant, detail.Messages[1].Role)
	assert.Equal(t, "system", detail.Messages[1].CreatedBy)
	assert.Equal(t, sentContent, detail.Messages[1].Content)
}

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
		AgentID: agent.ID, CLIType: remoteModel.CLITypeClaude, PermissionMode: remoteModel.PermissionModeAutoEdit, WorkingDirectory: "projects/api",
	})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.CLITypeClaude, conversation.CLIType)
	assert.Equal(t, remoteModel.PermissionModeAutoEdit, conversation.PermissionMode)
	assert.Equal(t, "projects/api", conversation.WorkingDirectory)
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: "SHELL"})
	require.EqualError(t, err, "CLI 类型必须是 CODEX 或 CLAUDE")
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex, PermissionMode: "ROOT"})
	require.EqualError(t, err, "权限模式必须是 AUTO_EDIT 或 FULL_ACCESS")
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex, WorkingDirectory: "../outside"})
	require.EqualError(t, err, "工作目录不能越出 workspace-root")
	_, err = service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex, WorkingDirectory: `D:\outside`})
	require.EqualError(t, err, "工作目录必须是相对于 workspace-root 的路径")

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
	require.EqualError(t, err, "会话标题不能为空")
	_, err = service.Rename(&remoteReq.ConversationRenameRequest{ID: otherConversation.ID, Title: strings.Repeat("x", 201)})
	require.EqualError(t, err, "标题不能超过 200 个字符")
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
	require.EqualError(t, err, "Agent 正在响应时不能删除会话")
	_, err = service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "second turn"})
	require.EqualError(t, err, "Agent 不在线或正在处理其他消息")

	command, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, command)
	assert.Equal(t, turn.AssistantMessage.ID, command.AssistantMessageID)
	assert.Equal(t, remoteModel.CLITypeClaude, command.CLIType)
	assert.Equal(t, remoteModel.PermissionModeAutoEdit, command.PermissionMode)
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
	require.EqualError(t, err, "单个消息分片不能超过 32 KB")
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
	require.EqualError(t, err, "助手消息不能超过 1 MB")
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
	assert.Equal(t, remoteModel.PermissionModeFullAccess, rootCommand.PermissionMode)
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
	require.EqualError(t, err, "会话工作目录不能为空")

	var messageCount int64
	require.NoError(t, database.Model(&remoteModel.Message{}).Where("conversation_id = ?", conversation.ID).Count(&messageCount).Error)
	assert.Zero(t, messageCount)
}

func TestConversationTaskCanPauseAndResume(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "pause-node", "registration", remoteModel.AgentStatusOnline)
	require.NoError(t, database.Model(&agent).Updates(map[string]any{
		"token_hash": hashAgentToken("runtime-token"), "status": remoteModel.AgentStatusOnline,
		"last_seen_at": time.Now().UnixMilli(),
	}).Error)
	service := NewConversationService(NewAgentService())
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{
		AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex,
	})
	require.NoError(t, err)
	turn, err := service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "implement pause"})
	require.NoError(t, err)
	command, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, command)
	_, err = service.Acknowledge(&remoteReq.AcknowledgeCommandParams{AgentToken: "runtime-token", CommandID: command.CommandID})
	require.NoError(t, err)

	pausing, err := service.Pause(&remoteReq.ConversationTaskRequest{MessageID: turn.AssistantMessage.ID})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.MessageStatusPausing, pausing.Status)
	control, err := service.CommandStatus(&remoteReq.CommandStatusParams{AgentToken: "runtime-token", CommandID: command.CommandID})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.CommandStatusPauseRequested, control.Status)

	_, err = service.Complete(&remoteReq.MessageResultParams{AgentToken: "runtime-token", Request: remoteReq.MessageResultRequest{
		MessageID: command.AssistantMessageID, Paused: true, Content: "partial output",
	}})
	require.NoError(t, err)
	detail, err := service.Get(conversation.ID)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 2)
	assert.Equal(t, remoteModel.MessageStatusPaused, detail.Messages[1].Status)
	assert.Equal(t, "partial output", detail.Messages[1].Content)

	_, err = service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "another task"})
	require.EqualError(t, err, "该会话有暂停的任务，请先恢复或删除会话")
	resumed, err := service.Resume(&remoteReq.ConversationTaskRequest{MessageID: command.AssistantMessageID})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.MessageStatusPending, resumed.Status)
	assert.Empty(t, resumed.Content)
	retry, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, retry)
	assert.Equal(t, command.CommandID, retry.CommandID)
	assert.Equal(t, int64(1), retry.NextChunkSequence)
}

func TestConversationTaskCanCancel(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "cancel-node", "registration", remoteModel.AgentStatusOnline)
	require.NoError(t, database.Model(&agent).Updates(map[string]any{
		"token_hash": hashAgentToken("runtime-token"), "last_seen_at": time.Now().UnixMilli(),
	}).Error)
	service := NewConversationService(NewAgentService())
	conversation, err := service.Create(&remoteReq.ConversationCreateRequest{AgentID: agent.ID, CLIType: remoteModel.CLITypeCodex})
	require.NoError(t, err)

	turn, err := service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "cancel before dispatch"})
	require.NoError(t, err)
	cancelled, err := service.Cancel(&remoteReq.ConversationTaskRequest{MessageID: turn.AssistantMessage.ID})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.MessageStatusCancelled, cancelled.Status)
	assert.Equal(t, "任务已取消", cancelled.ErrorMessage)
	var command remoteModel.Command
	require.NoError(t, database.Where("assistant_message_id = ?", turn.AssistantMessage.ID).First(&command).Error)
	assert.Equal(t, remoteModel.CommandStatusCancelled, command.Status)
	var storedAgent remoteModel.Agent
	require.NoError(t, database.First(&storedAgent, agent.ID).Error)
	assert.Equal(t, remoteModel.AgentStatusOnline, storedAgent.Status)
	assert.Zero(t, storedAgent.CurrentMessageID)

	nextTurn, err := service.Send(&remoteReq.SendMessageRequest{ConversationID: conversation.ID, Content: "cancel while running"})
	require.NoError(t, err)
	dispatched, err := service.NextCommand(&remoteReq.NextCommandParams{AgentToken: "runtime-token", WaitSeconds: 1})
	require.NoError(t, err)
	require.NotNil(t, dispatched)
	_, err = service.Acknowledge(&remoteReq.AcknowledgeCommandParams{AgentToken: "runtime-token", CommandID: dispatched.CommandID})
	require.NoError(t, err)
	cancelled, err = service.Cancel(&remoteReq.ConversationTaskRequest{MessageID: nextTurn.AssistantMessage.ID})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.MessageStatusCancelling, cancelled.Status)
	require.NoError(t, database.First(&storedAgent, agent.ID).Error)
	assert.Equal(t, remoteModel.AgentStatusBusy, storedAgent.Status)
	assert.Equal(t, nextTurn.AssistantMessage.ID, storedAgent.CurrentMessageID)

	_, err = service.Complete(&remoteReq.MessageResultParams{AgentToken: "runtime-token", Request: remoteReq.MessageResultRequest{
		MessageID: nextTurn.AssistantMessage.ID, Cancelled: true,
	}})
	require.NoError(t, err)
	require.NoError(t, database.First(&storedAgent, agent.ID).Error)
	assert.Equal(t, remoteModel.AgentStatusOnline, storedAgent.Status)
	assert.Zero(t, storedAgent.CurrentMessageID)
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
