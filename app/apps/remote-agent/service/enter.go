package service

type RemoteAgentServiceGroup struct {
	AgentService *AgentService
	TaskService  *TaskService
}

func NewRemoteAgentServiceGroup() *RemoteAgentServiceGroup {
	agentService := NewAgentService()
	return &RemoteAgentServiceGroup{AgentService: agentService, TaskService: NewTaskService(agentService)}
}

var RemoteAgentService = NewRemoteAgentServiceGroup()
