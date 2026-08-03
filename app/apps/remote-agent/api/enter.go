package api

import "apipig/app/apps/remote-agent/service"

type RemoteAgentApiGroup struct {
	AgentApi *AgentApi
	TaskApi  *TaskApi
}

func NewRemoteAgentApiGroup(services *service.RemoteAgentServiceGroup) *RemoteAgentApiGroup {
	return &RemoteAgentApiGroup{
		AgentApi: &AgentApi{service: services.AgentService},
		TaskApi:  &TaskApi{service: services.TaskService},
	}
}

var RemoteAgentApi = NewRemoteAgentApiGroup(service.RemoteAgentService)
