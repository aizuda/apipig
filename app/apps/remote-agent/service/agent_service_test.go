package service

import (
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	coreAPI "apipig/core/api"
	"apipig/global"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRegisterAndHeartbeat(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	currentTime := time.UnixMilli(1_800_000_000_000)
	service := NewAgentService()
	service.registrationToken = func() string { return "bootstrap-secret" }
	service.now = func() time.Time { return currentTime }

	registration, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: "bootstrap-secret",
		IPAddress:      "192.0.2.10",
		Request: remoteReq.RegisterRequest{
			AgentKey: "node-key-1", Name: "builder-1", Hostname: "workstation-1",
			OperatingSystem: "linux", Architecture: "amd64", CPUInfo: "8 cores",
			MemoryTotal: 16 << 30, CodexVersion: "1.2.3", AgentVersion: "0.1.0",
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, registration.AgentToken)
	assert.Equal(t, remoteModel.AgentStatusOnline, registration.Status)

	var stored remoteModel.Agent
	require.NoError(t, database.First(&stored, registration.AgentID).Error)
	assert.Equal(t, hashAgentToken(registration.AgentToken), stored.TokenHash)
	assert.NotContains(t, stored.TokenHash, registration.AgentToken)
	assert.Equal(t, "192.0.2.10", stored.IPAddress)

	currentTime = currentTime.Add(30 * time.Second)
	heartbeat, err := service.Heartbeat(&remoteReq.HeartbeatParams{
		AgentToken: registration.AgentToken,
		IPAddress:  "192.0.2.11",
		Request: remoteReq.HeartbeatRequest{
			CPUUsage: 37.5, MemoryUsed: 8 << 30, CodexVersion: "1.2.4", Busy: true,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, remoteModel.AgentStatusBusy, heartbeat.Status)

	require.NoError(t, database.First(&stored, registration.AgentID).Error)
	assert.Equal(t, remoteModel.AgentStatusBusy, stored.Status)
	assert.Equal(t, "192.0.2.11", stored.IPAddress)
	assert.Equal(t, "1.2.4", stored.CodexVersion)
	var heartbeatCount int64
	require.NoError(t, database.Model(&remoteModel.Heartbeat{}).Where("agent_id = ?", stored.ID).Count(&heartbeatCount).Error)
	assert.EqualValues(t, 1, heartbeatCount)
}

func TestRegistrationRejectsInvalidBootstrapToken(t *testing.T) {
	setupAgentServiceTestDB(t)
	service := NewAgentService()
	service.registrationToken = func() string { return "bootstrap-secret" }

	_, err := service.Register(&remoteReq.RegisterParams{
		BootstrapToken: "wrong",
		Request:        remoteReq.RegisterRequest{AgentKey: "node-key", Name: "node", Hostname: "host"},
	})
	require.EqualError(t, err, "invalid registration token")
}

func TestReregisterRotatesAgentToken(t *testing.T) {
	setupAgentServiceTestDB(t)
	service := NewAgentService()
	service.registrationToken = func() string { return "bootstrap-secret" }

	params := &remoteReq.RegisterParams{
		BootstrapToken: "bootstrap-secret",
		Request:        remoteReq.RegisterRequest{AgentKey: "stable-node-key", Name: "node", Hostname: "host"},
	}
	first, err := service.Register(params)
	require.NoError(t, err)
	second, err := service.Register(params)
	require.NoError(t, err)
	assert.Equal(t, first.AgentID, second.AgentID)
	assert.NotEqual(t, first.AgentToken, second.AgentToken)

	_, err = service.Heartbeat(&remoteReq.HeartbeatParams{AgentToken: first.AgentToken})
	require.EqualError(t, err, "invalid agent token")
	_, err = service.Heartbeat(&remoteReq.HeartbeatParams{AgentToken: second.AgentToken})
	require.NoError(t, err)
}

func TestMarkOfflinePreservesDisabledAgents(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	currentTime := time.UnixMilli(1_800_000_000_000)
	staleTime := currentTime.Add(-2 * time.Minute).UnixMilli()
	agents := []remoteModel.Agent{
		{MODEL: coreAPI.MODEL{ID: 101, CreatedBy: "test", CreatedAt: staleTime}, AgentKey: "online", Name: "online", TokenHash: "hash-1", Hostname: "host-1", Status: remoteModel.AgentStatusOnline, LastSeenAt: staleTime},
		{MODEL: coreAPI.MODEL{ID: 102, CreatedBy: "test", CreatedAt: staleTime}, AgentKey: "disabled", Name: "disabled", TokenHash: "hash-2", Hostname: "host-2", Status: remoteModel.AgentStatusDisabled, LastSeenAt: staleTime},
	}
	require.NoError(t, database.Create(&agents).Error)
	global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds = 90
	service := NewAgentService()
	service.now = func() time.Time { return currentTime }
	require.NoError(t, service.MarkOffline())

	var stored []remoteModel.Agent
	require.NoError(t, database.Order("id").Find(&stored).Error)
	require.Len(t, stored, 2)
	assert.Equal(t, remoteModel.AgentStatusOffline, stored[0].Status)
	assert.Equal(t, currentTime.UnixMilli(), stored[0].UpdatedAt)
	assert.Equal(t, remoteModel.AgentStatusDisabled, stored[1].Status)
}

func TestRemoteModelsMigrate(t *testing.T) {
	database := setupAgentServiceTestDB(t)
	require.NoError(t, database.AutoMigrate(
		&remoteModel.Task{}, &remoteModel.TaskLog{}, &remoteModel.Workspace{}, &remoteModel.Command{},
	))
	for _, table := range []string{
		"ap_remote_agent", "ap_remote_heartbeat", "ap_remote_task", "ap_remote_task_log",
		"ap_remote_workspace", "ap_remote_command",
	} {
		assert.True(t, database.Migrator().HasTable(table), table)
	}
}

func setupAgentServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&remoteModel.Agent{}, &remoteModel.Heartbeat{}))
	previousDB := global.DB
	previousConfig := global.CONFIG
	global.DB = database
	global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds = 90
	t.Cleanup(func() {
		global.DB = previousDB
		global.CONFIG = previousConfig
	})
	return database
}
