package service

type RemoteAgentServiceGroup struct {
	AgentService        *AgentService
	ConversationService *ConversationService
}

func NewRemoteAgentServiceGroup() *RemoteAgentServiceGroup {
	agentService := NewAgentService()
	return &RemoteAgentServiceGroup{AgentService: agentService, ConversationService: NewConversationService(agentService)}
}

var RemoteAgentService = NewRemoteAgentServiceGroup()
