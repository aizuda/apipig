package router

import (
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var testAgentID atomic.Int64

func seedRouterAgent(t *testing.T, database *gorm.DB, agentKey, name, token string) remoteModel.Agent {
	t.Helper()
	sum := sha256.Sum256([]byte(token))
	agent := remoteModel.Agent{
		MODEL:    coreAPI.MODEL{ID: snowflake.ID(10_000 + testAgentID.Add(1)), CreatedBy: "test", CreatedAt: time.Now().UnixMilli()},
		AgentKey: agentKey, Name: name, RegistrationTokenHash: "sha256:" + hex.EncodeToString(sum[:]),
		WorkspaceRoot: "./workspaces", CodexCommand: "codex",
		CodexArgs: []string{"exec", "-"}, PollWaitSeconds: 25, RequestTimeoutSeconds: 40,
		LogFile: "agent.log", Status: remoteModel.AgentStatusOffline,
	}
	require.NoError(t, database.Create(&agent).Error)
	return agent
}
