package api

import (
	"bufio"
	"encoding/json"
	"strconv"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/app/apps/remote-agent/service"
	coreAPI "apipig/core/api"
	"apipig/core/api/response"
	"apipig/toolkit/snowflake"

	"github.com/gofiber/fiber/v2"
)

type ConversationApi struct {
	coreAPI.API
	service *service.ConversationService
}

func (a *ConversationApi) Create(c *fiber.Ctx) error {
	var request remoteReq.ConversationCreateRequest
	err := a.BodyParser(c, &request, "Remote Agent conversation create")
	return response.Execute(c, a.service.Create, &request, err)
}

func (a *ConversationApi) Page(c *fiber.Ctx) error {
	var request remoteReq.ConversationPageParams
	err := a.BodyParser(c, &request, "Remote Agent conversation page")
	return response.Execute(c, a.service.Page, &request, err)
}

func (a *ConversationApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

func (a *ConversationApi) Pin(c *fiber.Ctx) error {
	var request remoteReq.ConversationPinRequest
	err := a.BodyParser(c, &request, "Remote Agent conversation pin")
	return response.Execute(c, a.service.Pin, &request, err)
}

func (a *ConversationApi) Rename(c *fiber.Ctx) error {
	var request remoteReq.ConversationRenameRequest
	err := a.BodyParser(c, &request, "Remote Agent conversation rename")
	return response.Execute(c, a.service.Rename, &request, err)
}

func (a *ConversationApi) Delete(c *fiber.Ctx) error {
	var request remoteReq.ConversationDeleteRequest
	err := a.BodyParser(c, &request, "Remote Agent conversation delete")
	return response.Execute(c, a.service.Delete, &request, err)
}

func (a *ConversationApi) Send(c *fiber.Ctx) error {
	var request remoteReq.SendMessageRequest
	err := a.BodyParser(c, &request, "Remote Agent conversation message")
	return response.Execute(c, a.service.Send, &request, err)
}

func (a *ConversationApi) NextCommand(c *fiber.Ctx) error {
	wait, _ := strconv.Atoi(c.Query("waitSeconds", "0"))
	params := &remoteReq.NextCommandParams{AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), WaitSeconds: wait}
	return response.Execute(c, a.service.NextCommand, params, nil)
}

func (a *ConversationApi) Acknowledge(c *fiber.Ctx) error {
	var body struct {
		CommandID snowflake.ID `json:"commandId"`
	}
	err := a.BodyParser(c, &body, "Remote Agent command acknowledgement")
	params := &remoteReq.AcknowledgeCommandParams{AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), CommandID: body.CommandID}
	return response.Execute(c, a.service.Acknowledge, params, err)
}

func (a *ConversationApi) AppendChunks(c *fiber.Ctx) error {
	var request remoteReq.MessageChunkUploadRequest
	err := a.BodyParser(c, &request, "Remote Agent message chunks")
	params := &remoteReq.MessageChunkUploadParams{AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), Request: request}
	return response.Execute(c, a.service.AppendChunks, params, err)
}

func (a *ConversationApi) Complete(c *fiber.Ctx) error {
	var request remoteReq.MessageResultRequest
	err := a.BodyParser(c, &request, "Remote Agent message result")
	params := &remoteReq.MessageResultParams{AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)), Request: request}
	return response.Execute(c, a.service.Complete, params, err)
}

func (a *ConversationApi) Stream(c *fiber.Ctx) error {
	var params remoteReq.MessageStreamParams
	if err := a.BodyParser(c, &params, "Remote Agent message stream"); err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, "text/event-stream; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		sequence := params.AfterSequence
		for {
			event, err := a.service.Stream(&remoteReq.MessageStreamParams{MessageID: params.MessageID, AfterSequence: sequence})
			if err != nil {
				_ = writeSSE(writer, "error", map[string]string{"message": err.Error()})
				return
			}
			for _, chunk := range event.Chunks {
				if err := writeSSE(writer, "chunk", chunk); err != nil {
					return
				}
				sequence = chunk.Sequence
			}
			if event.Message.Status == remoteModel.MessageStatusCompleted || event.Message.Status == remoteModel.MessageStatusFailed {
				_ = writeSSE(writer, "done", event.Message)
				return
			}
			if err := writeSSE(writer, "status", event.Message); err != nil {
				return
			}
			time.Sleep(350 * time.Millisecond)
		}
	})
	return nil
}

func writeSSE(writer *bufio.Writer, event string, value any) error {
	data, _ := json.Marshal(value)
	if _, err := writer.WriteString("event: " + event + "\n"); err != nil {
		return err
	}
	if _, err := writer.WriteString("data: " + string(data) + "\n\n"); err != nil {
		return err
	}
	return writer.Flush()
}
