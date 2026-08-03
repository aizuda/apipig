package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	remoteModel "apipig/app/apps/remote-agent/model"
	"apipig/global"
	"apipig/middleware"
	"apipig/toolkit/snowflake"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTaskCreateDispatchCompleteAndRecover(t *testing.T) {
	app, database, cleanup := setupTaskRouterTest(t)
	defer cleanup()
	agentID, agentToken := registerTaskAgent(t, app, "task-agent")

	taskID := createRemoteTask(t, app, agentID, "first task")
	command := nextTaskCommand(t, app, agentToken)
	assert.Equal(t, remoteModel.CommandTypeExecuteTask, command.Type)
	assert.Equal(t, taskID.String(), command.TaskID)
	acknowledgeTaskCommand(t, app, agentToken, command.CommandID)
	assertTaskStatus(t, database, taskID, remoteModel.TaskStatusRunning)
	uploadTaskLogs(t, app, agentToken, taskID, []map[string]any{
		{"sequence": 1, "stream": "stdout", "content": "starting\n"},
		{"sequence": 2, "stream": "stderr", "content": "warning\n"},
	})
	// A retried batch is idempotent on task ID and sequence.
	uploadTaskLogs(t, app, agentToken, taskID, []map[string]any{
		{"sequence": 1, "stream": "stdout", "content": "starting\n"},
		{"sequence": 2, "stream": "stderr", "content": "warning\n"},
	})

	response := performJSON(t, app, http.MethodPost, "/remote-agent/task/result", map[string]any{
		"taskId": taskID.String(), "success": true, "result": "completed",
		"changedFiles": []string{"README.md"},
	}, map[string]string{"Authorization": "Bearer " + agentToken})
	assert.Equal(t, "1", response.Code)
	assertTaskStatus(t, database, taskID, remoteModel.TaskStatusSuccess)
	assertAgentAvailable(t, database, agentID)
	var logCount int64
	require.NoError(t, database.Model(&remoteModel.TaskLog{}).Where("task_id = ?", taskID).Count(&logCount).Error)
	assert.EqualValues(t, 2, logCount)
	stream := performSSE(t, app, "/admin/task/log/stream", map[string]any{"taskId": taskID.String(), "afterSequence": 0})
	assert.Contains(t, stream, "event: log")
	assert.Contains(t, stream, `"content":"starting\n"`)
	assert.Contains(t, stream, "event: done")

	recoverTaskID := createRemoteTask(t, app, agentID, "recover task")
	recoverCommand := nextTaskCommand(t, app, agentToken)
	acknowledgeTaskCommand(t, app, agentToken, recoverCommand.CommandID)
	uploadTaskLogs(t, app, agentToken, recoverTaskID, []map[string]any{
		{"sequence": 1, "stream": "stdout", "content": "before crash\n"},
	})
	assertTaskStatus(t, database, recoverTaskID, remoteModel.TaskStatusRunning)
	registeredID, rotatedToken := registerTaskAgent(t, app, "task-agent")
	assert.Equal(t, agentID, registeredID)
	assert.NotEqual(t, agentToken, rotatedToken)
	assertTaskStatus(t, database, recoverTaskID, remoteModel.TaskStatusPending)
	redelivered := nextTaskCommand(t, app, rotatedToken)
	assert.Equal(t, recoverCommand.CommandID, redelivered.CommandID)
	assert.Equal(t, int64(2), redelivered.NextLogSequence)
}

func TestCancelRunningTaskDispatchesStop(t *testing.T) {
	app, database, cleanup := setupTaskRouterTest(t)
	defer cleanup()
	agentID, agentToken := registerTaskAgent(t, app, "cancel-agent")
	taskID := createRemoteTask(t, app, agentID, "cancel task")
	command := nextTaskCommand(t, app, agentToken)
	acknowledgeTaskCommand(t, app, agentToken, command.CommandID)

	cancelResponse := performJSON(t, app, http.MethodPost, "/admin/task/cancel", map[string]string{
		"id": taskID.String(),
	}, nil)
	assert.Equal(t, "1", cancelResponse.Code)
	assertTaskStatus(t, database, taskID, remoteModel.TaskStatusCancelled)
	stop := nextTaskCommand(t, app, agentToken)
	assert.Equal(t, remoteModel.CommandTypeStopTask, stop.Type)
	assert.Equal(t, taskID.String(), stop.TaskID)
	acknowledgeTaskCommand(t, app, agentToken, stop.CommandID)

	resultResponse := performJSON(t, app, http.MethodPost, "/remote-agent/task/result", map[string]any{
		"taskId": taskID.String(), "success": false, "errorMessage": "task cancelled",
	}, map[string]string{"Authorization": "Bearer " + agentToken})
	assert.Equal(t, "1", resultResponse.Code)
	assertTaskStatus(t, database, taskID, remoteModel.TaskStatusCancelled)
	assertAgentAvailable(t, database, agentID)
}

type testEnvelope struct {
	Code string          `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type testCommand struct {
	CommandID       string `json:"commandId"`
	Type            string `json:"type"`
	TaskID          string `json:"taskId"`
	NextLogSequence int64  `json:"nextLogSequence"`
}

func uploadTaskLogs(t *testing.T, app *fiber.App, token string, taskID snowflake.ID, logs []map[string]any) {
	t.Helper()
	response := performJSON(t, app, http.MethodPost, "/remote-agent/task/logs", map[string]any{
		"taskId": taskID.String(), "logs": logs,
	}, map[string]string{"Authorization": "Bearer " + token})
	require.Equal(t, "1", response.Code, response.Msg)
}

func performSSE(t *testing.T, app *fiber.App, path string, body any) string {
	t.Helper()
	var requestBody bytes.Buffer
	require.NoError(t, json.NewEncoder(&requestBody).Encode(body))
	request := httptest.NewRequest(http.MethodPost, path, &requestBody)
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	response, err := app.Test(request, 5000)
	require.NoError(t, err)
	defer response.Body.Close()
	var output bytes.Buffer
	_, err = output.ReadFrom(response.Body)
	require.NoError(t, err)
	assert.Contains(t, response.Header.Get(fiber.HeaderContentType), "text/event-stream")
	return output.String()
}

func setupTaskRouterTest(t *testing.T) (*fiber.App, *gorm.DB, func()) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&remoteModel.Agent{}, &remoteModel.Heartbeat{}, &remoteModel.Task{},
		&remoteModel.TaskLog{}, &remoteModel.Workspace{}, &remoteModel.Command{},
	))
	previousDB := global.DB
	previousConfig := global.CONFIG
	global.DB = database
	global.CONFIG.RemoteAgent.RegistrationToken = "bootstrap-secret"
	global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds = 90
	global.CONFIG.RemoteAgent.CommandPollTimeoutSeconds = 1
	global.CONFIG.RemoteAgent.DispatchLeaseSeconds = 1
	app := fiber.New()
	agentGroup := app.Group("/remote-agent/")
	RemoteAgentRoutes.InitAgentRouter(agentGroup)
	adminGroup := app.Group("/admin/")
	adminGroup.Use(func(c *fiber.Ctx) error {
		c.Locals("tokenClaims", &middleware.TokenClaims{ID: 1, Username: "admin"})
		return c.Next()
	})
	RemoteAgentRoutes.InitAdminRouter(adminGroup)
	cleanup := func() {
		global.DB = previousDB
		global.CONFIG = previousConfig
	}
	return app, database, cleanup
}

func registerTaskAgent(t *testing.T, app *fiber.App, key string) (snowflake.ID, string) {
	t.Helper()
	response := performJSON(t, app, http.MethodPost, "/remote-agent/register", map[string]string{
		"agentKey": key, "name": key, "hostname": key + "-host",
	}, map[string]string{"X-Agent-Registration-Token": "bootstrap-secret"})
	require.Equal(t, "1", response.Code, response.Msg)
	var result struct {
		AgentID    snowflake.ID `json:"agentId"`
		AgentToken string       `json:"agentToken"`
	}
	require.NoError(t, json.Unmarshal(response.Data, &result))
	return result.AgentID, result.AgentToken
}

func createRemoteTask(t *testing.T, app *fiber.App, agentID snowflake.ID, name string) snowflake.ID {
	t.Helper()
	response := performJSON(t, app, http.MethodPost, "/admin/task/create", map[string]string{
		"name": name, "agentId": agentID.String(), "repositoryUrl": "https://example.com/acme/repository.git",
		"workingDir": "src", "prompt": "Fix the failing test",
	}, nil)
	require.Equal(t, "1", response.Code, response.Msg)
	var result struct {
		ID snowflake.ID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(response.Data, &result))
	return result.ID
}

func nextTaskCommand(t *testing.T, app *fiber.App, token string) testCommand {
	t.Helper()
	response := performJSON(t, app, http.MethodGet, "/remote-agent/command/next?waitSeconds=1", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	require.Equal(t, "1", response.Code, response.Msg)
	var command testCommand
	require.NoError(t, json.Unmarshal(response.Data, &command))
	require.NotEmpty(t, command.CommandID)
	return command
}

func acknowledgeTaskCommand(t *testing.T, app *fiber.App, token, commandID string) {
	t.Helper()
	response := performJSON(t, app, http.MethodPost, "/remote-agent/command/acknowledge", map[string]string{
		"commandId": commandID,
	}, map[string]string{"Authorization": "Bearer " + token})
	require.Equal(t, "1", response.Code, response.Msg)
}

func performJSON(t *testing.T, app *fiber.App, method, path string, body any, headers map[string]string) testEnvelope {
	t.Helper()
	var requestBody bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&requestBody).Encode(body))
	}
	request := httptest.NewRequest(method, path, &requestBody)
	if body != nil {
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := app.Test(request, 5000)
	require.NoError(t, err)
	defer response.Body.Close()
	var envelope testEnvelope
	require.NoError(t, json.NewDecoder(response.Body).Decode(&envelope))
	return envelope
}

func assertTaskStatus(t *testing.T, database *gorm.DB, taskID snowflake.ID, expected string) {
	t.Helper()
	var task remoteModel.Task
	require.NoError(t, database.First(&task, taskID).Error)
	assert.Equal(t, expected, task.Status, fmt.Sprintf("task %s", taskID.String()))
}

func assertAgentAvailable(t *testing.T, database *gorm.DB, agentID snowflake.ID) {
	t.Helper()
	var agent remoteModel.Agent
	require.NoError(t, database.First(&agent, agentID).Error)
	assert.Equal(t, remoteModel.AgentStatusOnline, agent.Status)
	assert.Equal(t, snowflake.ID(0), agent.CurrentTaskID)
}
