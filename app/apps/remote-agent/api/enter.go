package api

import "apipig/app/apps/remote-agent/service"

type RemoteAgentApiGroup struct {
	AgentApi        *AgentApi
	ConversationApi *ConversationApi
}

func NewRemoteAgentApiGroup(services *service.RemoteAgentServiceGroup) *RemoteAgentApiGroup {
	return &RemoteAgentApiGroup{
		AgentApi:        &AgentApi{service: services.AgentService},
		ConversationApi: &ConversationApi{service: services.ConversationService},
	}
}

var RemoteAgentApi = NewRemoteAgentApiGroup(service.RemoteAgentService)
