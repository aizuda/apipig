package router

import (
	"apipig/app/apps/remote-agent/api"

	"github.com/gofiber/fiber/v2"
)

type RemoteAgentRouter struct{}

func (r *RemoteAgentRouter) InitAgentRouter(router fiber.Router) {
	router.Post("register", api.RemoteAgentApi.AgentApi.Register)
	router.Post("heartbeat", api.RemoteAgentApi.AgentApi.Heartbeat)
	router.Get("command/next", api.RemoteAgentApi.TaskApi.NextCommand)
	router.Post("command/acknowledge", api.RemoteAgentApi.TaskApi.Acknowledge)
	router.Post("task/result", api.RemoteAgentApi.TaskApi.Complete)
	router.Post("task/logs", api.RemoteAgentApi.TaskApi.AppendLogs)
}

func (r *RemoteAgentRouter) InitAdminRouter(router fiber.Router) {
	agent := router.Group("agent/")
	agent.Post("page", api.RemoteAgentApi.AgentApi.Page)
	agent.Get("get", api.RemoteAgentApi.AgentApi.Get)
	task := router.Group("task/")
	task.Post("create", api.RemoteAgentApi.TaskApi.Create)
	task.Post("page", api.RemoteAgentApi.TaskApi.Page)
	task.Get("get", api.RemoteAgentApi.TaskApi.Get)
	task.Post("cancel", api.RemoteAgentApi.TaskApi.Cancel)
	task.Post("log/stream", api.RemoteAgentApi.TaskApi.StreamLogs)
}
