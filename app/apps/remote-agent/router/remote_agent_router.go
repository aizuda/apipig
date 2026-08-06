package router

import (
	"apipig/app/apps/remote-agent/api"

	"github.com/gofiber/fiber/v2"
)

type RemoteAgentRouter struct{}

func (r *RemoteAgentRouter) InitAgentRouter(router fiber.Router) {
	router.Post("register", api.RemoteAgentApi.AgentApi.Register)
	router.Post("heartbeat", api.RemoteAgentApi.AgentApi.Heartbeat)
	router.Post("disconnect", api.RemoteAgentApi.AgentApi.Disconnect)
	router.Get("command/next", api.RemoteAgentApi.ConversationApi.NextCommand)
	router.Post("command/acknowledge", api.RemoteAgentApi.ConversationApi.Acknowledge)
	router.Post("message/chunks", api.RemoteAgentApi.ConversationApi.AppendChunks)
	router.Post("message/result", api.RemoteAgentApi.ConversationApi.Complete)
}

func (r *RemoteAgentRouter) InitAdminRouter(router fiber.Router) {
	agent := router.Group("agent/")
	agent.Post("page", api.RemoteAgentApi.AgentApi.Page)
	agent.Get("get", api.RemoteAgentApi.AgentApi.Get)
	agent.Get("status", api.RemoteAgentApi.AgentApi.Status)
	agent.Get("events", api.RemoteAgentApi.AgentApi.Events)
	agent.Post("create", api.RemoteAgentApi.AgentApi.Create)
	agent.Post("update", api.RemoteAgentApi.AgentApi.Update)
	agent.Post("rotate-token", api.RemoteAgentApi.AgentApi.RotateToken)
	agent.Post("status", api.RemoteAgentApi.AgentApi.SetStatus)
	agent.Post("delete", api.RemoteAgentApi.AgentApi.Delete)
	conversation := router.Group("conversation/")
	conversation.Post("create", api.RemoteAgentApi.ConversationApi.Create)
	conversation.Post("page", api.RemoteAgentApi.ConversationApi.Page)
	conversation.Get("get", api.RemoteAgentApi.ConversationApi.Get)
	conversation.Post("pin", api.RemoteAgentApi.ConversationApi.Pin)
	conversation.Post("rename", api.RemoteAgentApi.ConversationApi.Rename)
	conversation.Post("delete", api.RemoteAgentApi.ConversationApi.Delete)
	conversation.Post("message/send", api.RemoteAgentApi.ConversationApi.Send)
	conversation.Post("takeover/start", api.RemoteAgentApi.ConversationApi.StartTakeover)
	conversation.Post("takeover/stop", api.RemoteAgentApi.ConversationApi.StopTakeover)
	conversation.Post("message/stream", api.RemoteAgentApi.ConversationApi.Stream)
}
