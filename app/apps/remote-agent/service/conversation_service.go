package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	remoteResp "apipig/app/apps/remote-agent/model/response"
	coreAPI "apipig/core/api"
	coreResp "apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

const (
	maxConversationPromptBytes = 256 * 1024
	maxMessageChunkBytes       = 32 * 1024
	maxAssistantMessageBytes   = 1024 * 1024
	maxMessageErrorBytes       = 64 * 1024
)

type ConversationService struct {
	agentService *AgentService
	repository   conversationRepository
	now          func() time.Time
}

func NewConversationService(agentService *AgentService) *ConversationService {
	return &ConversationService{agentService: agentService, repository: conversationRepository{}, now: time.Now}
}

func (s *ConversationService) Create(request *remoteReq.ConversationCreateRequest) (remoteModel.Conversation, error) {
	if request == nil || request.AgentID == 0 {
		return remoteModel.Conversation{}, errors.New("agent ID is required")
	}
	if _, err := s.agentService.repository.Get(request.AgentID); err != nil {
		return remoteModel.Conversation{}, err
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "新会话"
	}
	if len([]rune(title)) > 200 {
		return remoteModel.Conversation{}, errors.New("title cannot exceed 200 characters")
	}
	now := s.now().UnixMilli()
	conversation := remoteModel.Conversation{
		MODEL:   coreAPI.MODEL{ID: db.GetId(), CreatedBy: "admin", CreatedAt: now, UpdatedAt: now},
		AgentID: request.AgentID, Title: title, Status: remoteModel.ConversationStatusActive, LastMessageAt: now,
	}
	return conversation, s.repository.Create(&conversation)
}

func (s *ConversationService) Page(params *remoteReq.ConversationPageParams) (coreResp.PageResult, error) {
	if params == nil || params.AgentID == 0 {
		return coreResp.PageResult{}, errors.New("agent ID is required")
	}
	return s.repository.Page(params)
}

func (s *ConversationService) Get(id snowflake.ID) (remoteResp.ConversationDetail, error) {
	if id == 0 {
		return remoteResp.ConversationDetail{}, errors.New("conversation ID is required")
	}
	conversation, err := s.repository.Get(id)
	if err != nil {
		return remoteResp.ConversationDetail{}, err
	}
	messages, err := s.repository.Messages(id)
	if err != nil {
		return remoteResp.ConversationDetail{}, err
	}
	return remoteResp.ConversationDetail{Conversation: conversation, Messages: messages}, nil
}

func (s *ConversationService) Pin(request *remoteReq.ConversationPinRequest) (remoteModel.Conversation, error) {
	if request == nil || request.ID == 0 {
		return remoteModel.Conversation{}, errors.New("conversation ID is required")
	}
	conversation, err := s.repository.Get(request.ID)
	if err != nil {
		return remoteModel.Conversation{}, err
	}
	if conversation.Pinned == request.Pinned {
		return conversation, nil
	}
	now := s.now().UnixMilli()
	pinnedAt := int64(0)
	if request.Pinned {
		pinnedAt = now
	}
	if err := s.repository.Pin(request.ID, request.Pinned, pinnedAt, now); err != nil {
		return remoteModel.Conversation{}, err
	}
	conversation.Pinned = request.Pinned
	conversation.PinnedAt = pinnedAt
	conversation.UpdatedAt = now
	return conversation, nil
}

func (s *ConversationService) Rename(request *remoteReq.ConversationRenameRequest) (remoteModel.Conversation, error) {
	if request == nil || request.ID == 0 {
		return remoteModel.Conversation{}, errors.New("conversation ID is required")
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		return remoteModel.Conversation{}, errors.New("conversation title is required")
	}
	if len([]rune(title)) > 200 {
		return remoteModel.Conversation{}, errors.New("title cannot exceed 200 characters")
	}
	conversation, err := s.repository.Get(request.ID)
	if err != nil {
		return remoteModel.Conversation{}, err
	}
	if conversation.Title == title {
		return conversation, nil
	}
	now := s.now().UnixMilli()
	if err := s.repository.Rename(request.ID, title, now); err != nil {
		return remoteModel.Conversation{}, err
	}
	conversation.Title = title
	conversation.UpdatedAt = now
	return conversation, nil
}

func (s *ConversationService) Delete(request *remoteReq.ConversationDeleteRequest) (bool, error) {
	if request == nil || request.ID == 0 {
		return false, errors.New("conversation ID is required")
	}
	return true, s.repository.Delete(request.ID)
}

func (s *ConversationService) Send(request *remoteReq.SendMessageRequest) (remoteResp.SendMessageResult, error) {
	if request == nil || request.ConversationID == 0 {
		return remoteResp.SendMessageResult{}, errors.New("conversation ID is required")
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return remoteResp.SendMessageResult{}, errors.New("message content is required")
	}
	if len([]byte(content)) > 128*1024 {
		return remoteResp.SendMessageResult{}, errors.New("message content cannot exceed 128 KB")
	}
	if err := s.agentService.MarkOffline(); err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	conversation, err := s.repository.Get(request.ConversationID)
	if err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	messages, err := s.repository.Messages(conversation.ID)
	if err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	now := s.now().UnixMilli()
	sequence := int64(1)
	if len(messages) > 0 {
		sequence = messages[len(messages)-1].Sequence + 1
	}
	userMessage := remoteModel.Message{
		MODEL:          coreAPI.MODEL{ID: db.GetId(), CreatedBy: "admin", CreatedAt: now, UpdatedAt: now},
		ConversationID: conversation.ID, AgentID: conversation.AgentID, Sequence: sequence,
		Role: remoteModel.MessageRoleUser, Status: remoteModel.MessageStatusCompleted, Content: content,
	}
	assistantMessage := remoteModel.Message{
		MODEL:          coreAPI.MODEL{ID: db.GetId(), CreatedBy: "remote-agent", CreatedAt: now, UpdatedAt: now},
		ConversationID: conversation.ID, AgentID: conversation.AgentID, Sequence: sequence + 1,
		Role: remoteModel.MessageRoleAssistant, Status: remoteModel.MessageStatusPending,
	}
	command := remoteModel.Command{
		MODEL:   coreAPI.MODEL{ID: db.GetId(), CreatedBy: "admin", CreatedAt: now, UpdatedAt: now},
		AgentID: conversation.AgentID, ConversationID: conversation.ID, UserMessageID: userMessage.ID,
		AssistantMessageID: assistantMessage.ID, Type: remoteModel.CommandTypeConversationTurn,
		Payload: buildConversationPrompt(messages, content), Status: remoteModel.CommandStatusPending,
	}
	if err := s.repository.CreateTurn(conversation, userMessage, assistantMessage, command, now); err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	return remoteResp.SendMessageResult{UserMessage: userMessage, AssistantMessage: assistantMessage}, nil
}

func (s *ConversationService) NextCommand(params *remoteReq.NextCommandParams) (*remoteResp.CommandDispatch, error) {
	if params == nil {
		return nil, errors.New("command parameters are required")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return nil, err
	}
	maxWait := global.CONFIG.RemoteAgent.CommandPollTimeoutSeconds
	if maxWait <= 0 || maxWait > 25 {
		maxWait = 25
	}
	wait := params.WaitSeconds
	if wait <= 0 || wait > maxWait {
		wait = maxWait
	}
	lease := global.CONFIG.RemoteAgent.DispatchLeaseSeconds
	if lease <= 0 {
		lease = 30
	}
	deadline := s.now().Add(time.Duration(wait) * time.Second)
	for {
		now := s.now()
		command, claimErr := s.repository.Claim(agent.ID, now.Add(-time.Duration(lease)*time.Second).UnixMilli(), now.UnixMilli())
		if claimErr == nil {
			next, err := s.repository.NextChunkSequence(command.AssistantMessageID)
			if err != nil {
				return nil, err
			}
			return &remoteResp.CommandDispatch{
				CommandID: command.ID, ConversationID: command.ConversationID, UserMessageID: command.UserMessageID,
				AssistantMessageID: command.AssistantMessageID, Type: command.Type, Prompt: command.Payload, NextChunkSequence: next,
			}, nil
		}
		if !errors.Is(claimErr, gorm.ErrRecordNotFound) {
			return nil, claimErr
		}
		if !s.now().Before(deadline) {
			return nil, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (s *ConversationService) Acknowledge(params *remoteReq.AcknowledgeCommandParams) (bool, error) {
	if params == nil || params.CommandID == 0 {
		return false, errors.New("command ID is required")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	return true, s.repository.Acknowledge(agent.ID, params.CommandID, s.now().UnixMilli())
}

func (s *ConversationService) AppendChunks(params *remoteReq.MessageChunkUploadParams) (bool, error) {
	if params == nil || params.Request.MessageID == 0 {
		return false, errors.New("message ID is required")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	if len(params.Request.Chunks) == 0 || len(params.Request.Chunks) > 200 {
		return false, errors.New("chunks must contain between 1 and 200 entries")
	}
	now := s.now().UnixMilli()
	chunks := make([]remoteModel.MessageChunk, 0, len(params.Request.Chunks))
	for _, entry := range params.Request.Chunks {
		if entry.Sequence <= 0 || entry.Content == "" {
			return false, errors.New("invalid message chunk")
		}
		if len([]byte(entry.Content)) > maxMessageChunkBytes {
			return false, errors.New("message chunk cannot exceed 32 KB")
		}
		chunks = append(chunks, remoteModel.MessageChunk{
			MODEL:     coreAPI.MODEL{ID: db.GetId(), CreatedBy: "remote-agent", CreatedAt: now},
			MessageID: params.Request.MessageID, Sequence: entry.Sequence, Content: entry.Content,
		})
	}
	return true, s.repository.AppendChunks(agent.ID, params.Request.MessageID, chunks, now)
}

func (s *ConversationService) Complete(params *remoteReq.MessageResultParams) (bool, error) {
	if params == nil || params.Request.MessageID == 0 {
		return false, errors.New("message ID is required")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	if len([]byte(params.Request.Content)) > maxAssistantMessageBytes {
		return false, errors.New("assistant message cannot exceed 1 MB")
	}
	if len([]byte(params.Request.ErrorMessage)) > maxMessageErrorBytes {
		return false, errors.New("message error cannot exceed 64 KB")
	}
	return true, s.repository.Complete(agent.ID, params.Request, s.now().UnixMilli())
}

func (s *ConversationService) Stream(params *remoteReq.MessageStreamParams) (remoteResp.MessageStreamEvent, error) {
	if params == nil || params.MessageID == 0 {
		return remoteResp.MessageStreamEvent{}, errors.New("message ID is required")
	}
	if params.AfterSequence < 0 {
		params.AfterSequence = 0
	}
	return s.repository.StreamEvent(params.MessageID, params.AfterSequence)
}

func buildConversationPrompt(messages []remoteModel.Message, content string) string {
	const instruction = "Continue the following coding-agent conversation in the current persistent workspace. Respond to the latest user request."
	latest := "User:\n" + content
	historyBudget := maxConversationPromptBytes - len(instruction) - len(latest) - 4
	if historyBudget < 0 {
		historyBudget = 0
	}
	history := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.Status != remoteModel.MessageStatusCompleted || strings.TrimSpace(message.Content) == "" {
			continue
		}
		role := "User"
		if message.Role == remoteModel.MessageRoleAssistant {
			role = "Assistant"
		}
		history = append(history, fmt.Sprintf("%s:\n%s", role, message.Content))
	}
	selected := make([]string, 0, len(history))
	used := 0
	for index := len(history) - 1; index >= 0; index-- {
		required := len(history[index])
		if len(selected) > 0 {
			required += 2
		}
		if used+required > historyBudget {
			break
		}
		selected = append(selected, history[index])
		used += required
	}
	parts := make([]string, 0, len(selected)+2)
	parts = append(parts, instruction)
	for index := len(selected) - 1; index >= 0; index-- {
		parts = append(parts, selected[index])
	}
	parts = append(parts, latest)
	return strings.Join(parts, "\n\n")
}
