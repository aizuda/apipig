package api

import (
	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/app/apps/remote-agent/service"
	coreAPI "apipig/core/api"
	"apipig/core/api/response"
	"bufio"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const registrationTokenHeader = "X-Agent-Registration-Token"

type AgentApi struct {
	coreAPI.API
	service *service.AgentService
}

func (a *AgentApi) Register(c *fiber.Ctx) error {
	var request remoteReq.RegisterRequest
	err := a.BodyParser(c, &request, "Remote Agent registration")
	params := &remoteReq.RegisterParams{
		Request: request, IPAddress: c.IP(), BootstrapToken: c.Get(registrationTokenHeader),
	}
	return response.Execute(c, a.service.Register, params, err)
}

func (a *AgentApi) Heartbeat(c *fiber.Ctx) error {
	var request remoteReq.HeartbeatRequest
	err := a.BodyParser(c, &request, "Remote Agent heartbeat")
	params := &remoteReq.HeartbeatParams{
		Request: request, IPAddress: c.IP(), AgentToken: bearerToken(c.Get(fiber.HeaderAuthorization)),
	}
	return response.Execute(c, a.service.Heartbeat, params, err)
}

func (a *AgentApi) Disconnect(c *fiber.Ctx) error {
	return response.Execute(c, a.service.Disconnect, bearerToken(c.Get(fiber.HeaderAuthorization)), nil)
}

func (a *AgentApi) Page(c *fiber.Ctx) error {
	var params remoteReq.AgentPageParams
	err := a.BodyParser(c, &params, "Remote Agent page")
	return response.Execute(c, a.service.Page, &params, err)
}

func (a *AgentApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

func (a *AgentApi) Status(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Status, id, err)
}

func (a *AgentApi) Events(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/event-stream; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		events, unsubscribe := a.service.AgentEvents()
		defer unsubscribe()
		if err := writeSSE(writer, "ready", map[string]int64{"serverTime": time.Now().UnixMilli()}); err != nil {
			return
		}
		keepalive := time.NewTicker(15 * time.Second)
		defer keepalive.Stop()
		for {
			select {
			case event := <-events:
				if err := writeSSE(writer, "agent", event); err != nil {
					return
				}
			case <-keepalive.C:
				if _, err := writer.WriteString(": keepalive\n\n"); err != nil {
					return
				}
				if err := writer.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}

func (a *AgentApi) Create(c *fiber.Ctx) error {
	var request remoteReq.AgentSaveRequest
	err := a.BodyParser(c, &request, "Remote Agent create")
	params := &remoteReq.AgentCreateParams{Request: request, ControllerURL: requestControllerURL(c)}
	return response.Execute(c, a.service.Create, params, err)
}

func (a *AgentApi) Update(c *fiber.Ctx) error {
	var request remoteReq.AgentSaveRequest
	err := a.BodyParser(c, &request, "Remote Agent update")
	return response.Execute(c, a.service.Update, &request, err)
}

func (a *AgentApi) RotateToken(c *fiber.Ctx) error {
	id, err := a.KeyParser(c, "id")
	params := &remoteReq.AgentRotateTokenParams{ID: id, ControllerURL: requestControllerURL(c)}
	return response.Execute(c, a.service.RotateToken, params, err)
}

func (a *AgentApi) SetStatus(c *fiber.Ctx) error {
	var request remoteReq.AgentStatusRequest
	err := a.BodyParser(c, &request, "Remote Agent status")
	return response.Execute(c, a.service.SetStatus, &request, err)
}

func (a *AgentApi) Delete(c *fiber.Ctx) error {
	var request remoteReq.AgentDeleteRequest
	err := a.BodyParser(c, &request, "Remote Agent delete")
	return response.Execute(c, a.service.Delete, &request, err)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func requestControllerURL(c *fiber.Ctx) string {
	baseURL := strings.TrimRight(c.BaseURL(), "/")
	if index := strings.LastIndex(c.Path(), "/v1/"); index > 0 {
		baseURL += strings.TrimRight(c.Path()[:index], "/")
	}
	return baseURL
}
