package service

import (
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	coreAPI "apipig/core/api"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateAndRegisterUsesOneAgentRecord(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	service := NewAgentService()
	service.now = func() time.Time { return time.UnixMilli(1_800_000_000_000) }
	credential, err := service.Create(&remoteReq.AgentCreateParams{
		ControllerURL: "https://controller.example.com/admin/",
		Request: remoteReq.AgentSaveRequest{
			AgentKey: "node-key-1", Name: "builder-1",
			WorkspaceRoot: "./workspaces", CodexCommand: "codex",
		},
	})
	require.NoError(t, err)
	assert.Contains(t, credential.ConfigYAML, "controller-url: https://controller.example.com/admin")
	assert.Contains(t, credential.ConfigYAML, "sandbox_workspace_write.network_access=true")
	assert.NotContains(t, credential.ConfigYAML, "--full-auto")
	registration, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: credential.RegistrationToken, IPAddress: "192.0.2.10",
		Request: remoteReq.RegisterRequest{AgentKey: "node-key-1", Hostname: "workstation-1", OperatingSystem: "linux"},
	})
	require.NoError(t, err)
	assert.Equal(t, credential.Agent.ID, registration.AgentID)
	var count int64
	require.NoError(t, database.Model(&remoteModel.Agent{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	var stored remoteModel.Agent
	require.NoError(t, database.First(&stored, registration.AgentID).Error)
	assert.Equal(t, remoteModel.AgentStatusOnline, stored.Status)
	assert.Equal(t, "workstation-1", stored.Hostname)
	assert.Equal(t, hashAgentToken(registration.AgentToken), stored.TokenHash)
}

func TestRegistrationRequiresPreconfiguredAgent(t *testing.T) {
	setupAgentServiceTestDB(t)
	service := NewAgentService()
	_, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: "unknown",
		Request:        remoteReq.RegisterRequest{AgentKey: "missing", Hostname: "host"},
	})
	require.EqualError(t, err, "控制端尚未添加远程 Agent \"missing\"")
}

func TestAgentKeyAllowsOnlyLettersNumbersHyphensAndUnderscores(t *testing.T) {
	for _, valid := range []string{"agent01", "Agent-01", "agent_node_01"} {
		assert.NoError(t, validateAgentKey(valid), valid)
	}
	for _, invalid := range []string{"agent key", "agent.key", "节点-01", "agent@01"} {
		require.EqualError(t, validateAgentKey(invalid), "Agent Key 只能包含字母、数字、连字符和下划线", invalid)
	}
}

func TestRotateTokenRejectsOnlineAgent(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "online-node", "secret", remoteModel.AgentStatusOnline)
	service := NewAgentService()
	_, err := service.RotateToken(&remoteReq.AgentRotateTokenParams{
		ID: agent.ID, ControllerURL: "https://controller.example.com",
	})
	require.EqualError(t, err, "只有离线或已禁用的 Agent 才能重置注册令牌")
}

func TestDeleteAgentRemovesAssociatedData(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	now := time.Now().UnixMilli()
	agent := seedTestAgent(t, database, "delete-node", "secret", remoteModel.AgentStatusOffline)
	conversation := remoteModel.Conversation{
		MODEL: coreAPI.MODEL{ID: 201, CreatedBy: "test", CreatedAt: now}, AgentID: agent.ID,
		Title: "Active conversation", CLIType: remoteModel.CLITypeCodex,
		WorkingDirectory: ".apipig/conversations/200", Status: remoteModel.ConversationStatusActive, LastMessageAt: now,
	}
	messages := []remoteModel.Message{
		{MODEL: coreAPI.MODEL{ID: 202, CreatedBy: "test", CreatedAt: now}, ConversationID: conversation.ID, AgentID: agent.ID, Sequence: 1, Role: remoteModel.MessageRoleUser, Status: remoteModel.MessageStatusCompleted, Content: "request"},
		{MODEL: coreAPI.MODEL{ID: 203, CreatedBy: "test", CreatedAt: now}, ConversationID: conversation.ID, AgentID: agent.ID, Sequence: 2, Role: remoteModel.MessageRoleAssistant, Status: remoteModel.MessageStatusCompleted, Content: "response"},
	}
	chunk := remoteModel.MessageChunk{
		MODEL: coreAPI.MODEL{ID: 204, CreatedBy: "test", CreatedAt: now}, MessageID: messages[1].ID, Sequence: 1, Content: "response",
	}
	command := remoteModel.Command{
		MODEL: coreAPI.MODEL{ID: 205, CreatedBy: "test", CreatedAt: now}, AgentID: agent.ID,
		ConversationID: conversation.ID, UserMessageID: messages[0].ID, AssistantMessageID: messages[1].ID,
		Type: remoteModel.CommandTypeConversationTurn, Status: remoteModel.CommandStatusCompleted,
	}
	heartbeat := remoteModel.Heartbeat{
		MODEL: coreAPI.MODEL{ID: 206, CreatedBy: "test", CreatedAt: now}, AgentID: agent.ID,
		Status: remoteModel.AgentStatusOffline, OccurredAt: now,
	}
	require.NoError(t, database.Create(&conversation).Error)
	require.NoError(t, database.Create(&messages).Error)
	require.NoError(t, database.Create(&chunk).Error)
	require.NoError(t, database.Create(&command).Error)
	require.NoError(t, database.Create(&heartbeat).Error)

	service := NewAgentService()
	deleted, err := service.Delete(&remoteReq.AgentDeleteRequest{ID: agent.ID})
	require.NoError(t, err)
	assert.True(t, deleted)

	checks := []struct {
		model any
		where string
		value any
	}{
		{model: &remoteModel.Agent{}, where: "id = ?", value: agent.ID},
		{model: &remoteModel.Heartbeat{}, where: "agent_id = ?", value: agent.ID},
		{model: &remoteModel.Conversation{}, where: "agent_id = ?", value: agent.ID},
		{model: &remoteModel.Message{}, where: "agent_id = ?", value: agent.ID},
		{model: &remoteModel.MessageChunk{}, where: "message_id = ?", value: messages[1].ID},
		{model: &remoteModel.Command{}, where: "agent_id = ?", value: agent.ID},
	}
	for _, check := range checks {
		var count int64
		require.NoError(t, database.Unscoped().Model(check.model).Where(check.where, check.value).Count(&count).Error)
		assert.Zero(t, count)
	}

	_, err = service.Create(&remoteReq.AgentCreateParams{
		ControllerURL: "https://controller.example.com",
		Request:       remoteReq.AgentSaveRequest{AgentKey: agent.AgentKey, Name: "replacement"},
	})
	require.NoError(t, err)
}

func TestDeleteAgentRejectsActiveResponse(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "busy-node", "secret", remoteModel.AgentStatusBusy)
	require.NoError(t, database.Model(&remoteModel.Agent{}).Where("id = ?", agent.ID).Update("current_message_id", 999).Error)

	_, err := NewAgentService().Delete(&remoteReq.AgentDeleteRequest{ID: agent.ID})

	require.EqualError(t, err, "Agent 正在响应时不能删除")
}

func TestHeartbeatSchedulesOfflineTransition(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "node", "secret", remoteModel.AgentStatusOffline)
	service := NewAgentService()
	service.livenessTimeout = func() time.Duration { return 25 * time.Millisecond }
	t.Cleanup(service.StopLivenessTracking)
	events, unsubscribe := service.AgentEvents()
	defer unsubscribe()
	registration, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: "secret", Request: remoteReq.RegisterRequest{AgentKey: agent.AgentKey, Hostname: "host"},
	})
	require.NoError(t, err)
	heartbeat, err := service.Heartbeat(&remoteReq.HeartbeatParams{
		AgentToken: registration.AgentToken, Request: remoteReq.HeartbeatRequest{CPUUsage: 10, MemoryUsed: 20, Busy: true},
	})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.AgentStatusBusy, heartbeat.Status)

	deadline := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.Agent.ID == agent.ID && event.Agent.Status == remoteModel.AgentStatusOffline {
				goto offline
			}
		case <-deadline:
			t.Fatal("timed out waiting for Agent offline event")
		}
	}

offline:
	var stored remoteModel.Agent
	require.NoError(t, database.First(&stored, agent.ID).Error)
	assert.Equal(t, remoteModel.AgentStatusOffline, stored.Status)
	assert.Empty(t, stored.TokenHash)
}

func TestRegisterRecoversInterruptedConversationTurn(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	now := time.UnixMilli(1_800_000_000_000)
	agent := seedTestAgent(t, database, "node", "secret", remoteModel.AgentStatusBusy)
	conversation := remoteModel.Conversation{
		MODEL:            coreAPI.MODEL{ID: snowflake.ID(101), CreatedBy: "test", CreatedAt: now.UnixMilli()},
		AgentID:          agent.ID,
		Title:            "Recovery",
		CLIType:          remoteModel.CLITypeCodex,
		WorkingDirectory: ".apipig/conversations/101",
		Status:           remoteModel.ConversationStatusActive,
		LastMessageAt:    now.UnixMilli(),
	}
	message := remoteModel.Message{
		MODEL:          coreAPI.MODEL{ID: snowflake.ID(102), CreatedBy: "test", CreatedAt: now.UnixMilli()},
		ConversationID: conversation.ID,
		AgentID:        agent.ID,
		Sequence:       2,
		Role:           remoteModel.MessageRoleAssistant,
		Status:         remoteModel.MessageStatusStreaming,
		Content:        "partial response",
	}
	command := remoteModel.Command{
		MODEL:              coreAPI.MODEL{ID: snowflake.ID(103), CreatedBy: "test", CreatedAt: now.UnixMilli()},
		AgentID:            agent.ID,
		ConversationID:     conversation.ID,
		AssistantMessageID: message.ID,
		Type:               remoteModel.CommandTypeConversationTurn,
		Status:             remoteModel.CommandStatusAcknowledged,
		DispatchedAt:       now.UnixMilli() - 2_000,
		AcknowledgedAt:     now.UnixMilli() - 1_000,
	}
	chunk := remoteModel.MessageChunk{
		MODEL:     coreAPI.MODEL{ID: snowflake.ID(104), CreatedBy: "test", CreatedAt: now.UnixMilli()},
		MessageID: message.ID, Sequence: 1, Content: message.Content,
	}
	require.NoError(t, database.Create(&conversation).Error)
	require.NoError(t, database.Create(&message).Error)
	require.NoError(t, database.Create(&command).Error)
	require.NoError(t, database.Create(&chunk).Error)
	require.NoError(t, database.Model(&remoteModel.Agent{}).Where("id = ?", agent.ID).
		Update("current_message_id", message.ID).Error)

	service := NewAgentService()
	service.now = func() time.Time { return now }
	registration, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: "secret",
		Request:        remoteReq.RegisterRequest{AgentKey: agent.AgentKey, Hostname: "restarted-host"},
	})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.AgentStatusOnline, registration.Status)

	var storedAgent remoteModel.Agent
	require.NoError(t, database.First(&storedAgent, agent.ID).Error)
	assert.Zero(t, storedAgent.CurrentMessageID)
	assert.Equal(t, remoteModel.AgentStatusOnline, storedAgent.Status)
	require.NoError(t, database.First(&message, message.ID).Error)
	assert.Equal(t, remoteModel.MessageStatusPending, message.Status)
	assert.Empty(t, message.Content)
	require.NoError(t, database.First(&command, command.ID).Error)
	assert.Equal(t, remoteModel.CommandStatusPending, command.Status)
	assert.Zero(t, command.DispatchedAt)
	assert.Zero(t, command.AcknowledgedAt)
	var chunkCount int64
	require.NoError(t, database.Model(&remoteModel.MessageChunk{}).Where("message_id = ?", message.ID).Count(&chunkCount).Error)
	assert.Zero(t, chunkCount)
}

func TestDisconnectImmediatelyMarksAgentOffline(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	agent := seedTestAgent(t, database, "disconnect-node", "secret", remoteModel.AgentStatusOffline)
	service := NewAgentService()
	registration, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: "secret", Request: remoteReq.RegisterRequest{AgentKey: agent.AgentKey, Hostname: "host"},
	})
	require.NoError(t, err)

	disconnected, err := service.Disconnect(registration.AgentToken)

	require.NoError(t, err)
	assert.True(t, disconnected)
	status, err := service.Status(agent.ID)
	require.NoError(t, err)
	assert.Equal(t, remoteModel.AgentStatusOffline, status.Status)
	assert.Empty(t, status.TokenHash)
}

func TestRemoteConversationModelsMigrate(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	for _, table := range []string{"ap_remote_agent", "ap_remote_heartbeat", "ap_remote_agent_conversation", "ap_remote_agent_message", "ap_remote_agent_message_chunk", "ap_remote_agent_command"} {
		assert.True(t, database.Migrator().HasTable(table), table)
	}
	assert.True(t, database.Migrator().HasColumn(&remoteModel.Conversation{}, "Pinned"))
	assert.True(t, database.Migrator().HasColumn(&remoteModel.Conversation{}, "PinnedAt"))
}

func setupAgentServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&remoteModel.Agent{}, &remoteModel.Heartbeat{}, &remoteModel.Conversation{}, &remoteModel.Message{}, &remoteModel.MessageChunk{}, &remoteModel.Command{}))
	previousDB, previousConfig := global.DB, global.CONFIG
	global.DB = database
	global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds = 90
	t.Cleanup(func() { global.DB, global.CONFIG = previousDB, previousConfig })
	return database
}

func seedTestAgent(t *testing.T, database *gorm.DB, key, registrationToken, status string) remoteModel.Agent {
	t.Helper()
	agent := remoteModel.Agent{
		MODEL:    coreAPI.MODEL{ID: snowflake.ID(time.Now().UnixNano()), CreatedBy: "test", CreatedAt: time.Now().UnixMilli()},
		AgentKey: key, Name: key, RegistrationTokenHash: hashAgentToken(registrationToken),
		WorkspaceRoot: "./workspaces", CodexCommand: "codex",
		CodexArgs: []string{"exec", "-"}, PollWaitSeconds: 25, RequestTimeoutSeconds: 40,
		LogFile: "agent.log", Status: status,
	}
	require.NoError(t, database.Create(&agent).Error)
	return agent
}
