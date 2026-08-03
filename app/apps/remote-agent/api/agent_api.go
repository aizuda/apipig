package api

import (
	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/app/apps/remote-agent/service"
	coreAPI "apipig/core/api"
	"apipig/core/api/response"
	"strings"

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

func (a *AgentApi) Page(c *fiber.Ctx) error {
	var params remoteReq.AgentPageParams
	err := a.BodyParser(c, &params, "Remote Agent page")
	return response.Execute(c, a.service.Page, &params, err)
}

func (a *AgentApi) Get(c *fiber.Ctx) error {
	id, err := a.IdParser(c)
	return response.Execute(c, a.service.Get, id, err)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
