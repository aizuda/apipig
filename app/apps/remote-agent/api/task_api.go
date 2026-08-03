package api

import (
	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/app/apps/remote-agent/service"
	coreAPI "apipig/core/api"
	coreReq "apipig/core/api/request"
	"apipig/core/api/response"
	"bufio"
	"encoding/json"
	"strconv"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"

	"github.com/gofiber/fiber/v2"
)

type TaskApi struct {
	coreAPI.API
	service *service.TaskService
}

func (a *TaskApi) Create(c *fiber.Ctx) error {
	var request remoteReq.TaskCreateRequest
	err := a.BodyParser(c, &request, "Remote Agent task")
	params := &remoteReq.TaskCreateParams{Ctx: c, Request: request}
	return response.Execute(c, a.service.Create, params, err)
}

func (a *TaskApi) Page(c *fiber.Ctx) error {
	var params remoteReq.TaskPageParams
	err := a.BodyParser(c, &params, "Remote Agent task page")
	return response.Execute(c, a.service.Page, &params, err)
}

func (a *TaskApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

func (a *TaskApi) Cancel(c *fiber.Ctx) error {
	var request coreReq.GetById
	err := a.BodyParser(c, &request, "Cancel Remote Agent task")
	params := &remoteReq.CancelTaskParams{Ctx: c, ID: request.ID}
	return response.Execute(c, a.service.Cancel, params, err)
}

func (a *TaskApi) NextCommand(c *fiber.Ctx) error {
	waitSeconds, _ := strconv.Atoi(c.Query("waitSeconds"))
	params := &remoteReq.NextCommandParams{
		AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), WaitSeconds: waitSeconds,
	}
	return response.Execute(c, a.service.NextCommand, params, nil)
}

func (a *TaskApi) Acknowledge(c *fiber.Ctx) error {
	var request remoteReq.AcknowledgeCommandRequest
	err := a.BodyParser(c, &request, "Acknowledge Remote Agent command")
	params := &remoteReq.AcknowledgeCommandParams{
		AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), CommandID: request.CommandID,
	}
	return response.Execute(c, a.service.Acknowledge, params, err)
}

func (a *TaskApi) Complete(c *fiber.Ctx) error {
	var request remoteReq.TaskResultRequest
	err := a.BodyParser(c, &request, "Remote Agent task result")
	params := &remoteReq.TaskResultParams{
		AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), Request: request,
	}
	return response.Execute(c, a.service.Complete, params, err)
}

func (a *TaskApi) AppendLogs(c *fiber.Ctx) error {
	var request remoteReq.TaskLogUploadRequest
	err := a.BodyParser(c, &request, "Remote Agent task logs")
	params := &remoteReq.TaskLogUploadParams{
		AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), Request: request,
	}
	return response.Execute(c, a.service.AppendLogs, params, err)
}

func (a *TaskApi) StreamLogs(c *fiber.Ctx) error {
	var params remoteReq.TaskLogStreamParams
	if err := a.BodyParser(c, &params, "Remote Agent task log stream"); err != nil {
		return response.Failed(c, err.Error())
	}
	if _, err := a.service.Get(params.TaskID); err != nil {
		return response.Failed(c, err.Error())
	}
	c.Set(fiber.HeaderContentType, "text/event-stream; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		a.streamTaskLogs(writer, params)
	})
	return nil
}

func (a *TaskApi) streamTaskLogs(writer *bufio.Writer, params remoteReq.TaskLogStreamParams) {
	after := params.AfterSequence
	heartbeat := time.NewTicker(10 * time.Second)
	poll := time.NewTicker(500 * time.Millisecond)
	defer heartbeat.Stop()
	defer poll.Stop()
	for {
		logs, err := a.service.LogsAfter(params.TaskID, after)
		if err != nil {
			_ = writeTaskEvent(writer, "error", map[string]string{"message": err.Error()})
			return
		}
		for _, item := range logs {
			if err := writeTaskEvent(writer, "log", item); err != nil {
				return
			}
			after = item.Sequence
		}
		if len(logs) > 0 {
			if err := writer.Flush(); err != nil {
				return
			}
			continue
		}
		status, err := a.service.Status(params.TaskID)
		if err != nil {
			_ = writeTaskEvent(writer, "error", map[string]string{"message": err.Error()})
			return
		}
		if terminalTaskStatus(status) {
			_ = writeTaskEvent(writer, "done", map[string]string{"status": status})
			_ = writer.Flush()
			return
		}
		select {
		case <-heartbeat.C:
			if _, err := writer.WriteString(": keep-alive\n\n"); err != nil {
				return
			}
			if err := writer.Flush(); err != nil {
				return
			}
		case <-poll.C:
		}
	}
}

func writeTaskEvent(writer *bufio.Writer, event string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err = writer.WriteString("event: " + event + "\n"); err != nil {
		return err
	}
	_, err = writer.WriteString("data: " + string(data) + "\n\n")
	return err
}

func terminalTaskStatus(status string) bool {
	return status == remoteModel.TaskStatusSuccess || status == remoteModel.TaskStatusFailed || status == remoteModel.TaskStatusCancelled
}
