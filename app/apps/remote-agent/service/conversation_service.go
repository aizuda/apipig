package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	remoteResp "apipig/app/apps/remote-agent/model/response"
	wechatModel "apipig/app/apps/wechat-bot/model"
	coreAPI "apipig/core/api"
	coreResp "apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	maxConversationPromptBytes = 256 * 1024
	maxMessageChunkBytes       = 32 * 1024
	maxAssistantMessageBytes   = 1024 * 1024
	maxMessageErrorBytes       = 64 * 1024
)

type ConversationService struct {
	agentService   *AgentService
	repository     conversationRepository
	now            func() time.Time
	takeoverSender func(context.Context, snowflake.ID, string, string) error
}

func NewConversationService(agentService *AgentService) *ConversationService {
	service := &ConversationService{agentService: agentService, repository: conversationRepository{}, now: time.Now}
	agentService.SetStatusChangeHandler(service.handleAgentStatusChange)
	return service
}

func (s *ConversationService) SetTakeoverSender(sender func(context.Context, snowflake.ID, string, string) error) {
	s.takeoverSender = sender
}

func (s *ConversationService) StartTakeover(request *remoteReq.ConversationTakeoverRequest) (remoteModel.Conversation, error) {
	if request == nil || request.ID == 0 || request.BotID == 0 || strings.TrimSpace(request.UserID) == "" {
		return remoteModel.Conversation{}, errors.New("会话、Bot 和联系人不能为空")
	}
	var bot wechatModel.Bot
	if err := global.DB.First(&bot, request.BotID).Error; err != nil {
		return remoteModel.Conversation{}, errors.New("微信 Bot 不存在")
	}
	if !bot.Enabled || bot.Status != wechatModel.BotStatusOnline {
		return remoteModel.Conversation{}, errors.New("微信 Bot 当前不在线")
	}
	var contact wechatModel.Contact
	if err := global.DB.Where("bot_record_id = ? AND user_id = ?", request.BotID, strings.TrimSpace(request.UserID)).First(&contact).Error; err != nil {
		return remoteModel.Conversation{}, errors.New("未找到联系人，请先让该用户向 Bot 发送消息")
	}
	if s.now().Sub(time.UnixMilli(contact.LastActiveAt)) >= 24*time.Hour {
		return remoteModel.Conversation{}, errors.New("联系人会话已超过 24 小时，请先让该用户向 Bot 发送消息")
	}
	if err := s.repository.SetTakeover(request.ID, request.BotID, strings.TrimSpace(request.UserID), s.now().UnixMilli()); err != nil {
		return remoteModel.Conversation{}, err
	}
	return s.repository.Get(request.ID)
}

func (s *ConversationService) StopTakeover(request *remoteReq.ConversationTakeoverRequest) (remoteModel.Conversation, error) {
	if request == nil || request.ID == 0 {
		return remoteModel.Conversation{}, errors.New("会话 ID 不能为空")
	}
	if err := s.repository.StopTakeover(request.ID, s.now().UnixMilli()); err != nil {
		return remoteModel.Conversation{}, err
	}
	return s.repository.Get(request.ID)
}

func (s *ConversationService) HandleWechatInbound(botID snowflake.ID, userID, content string) error {
	if botID == 0 || strings.TrimSpace(userID) == "" || strings.TrimSpace(content) == "" {
		return nil
	}
	var conversation remoteModel.Conversation
	if err := global.DB.Where("control_mode = ? AND wechat_bot_id = ? AND wechat_user_id = ?", remoteModel.ConversationControlModeWechat, botID, userID).First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	_, err := s.send(conversation.ID, content, "WECHAT")
	if err != nil {
		agent, agentErr := s.agentService.repository.Get(conversation.AgentID)
		if agentErr == nil && agent.Status == remoteModel.AgentStatusOffline {
			if mirrorErr := s.repository.AppendTakeoverInbound(conversation, strings.TrimSpace(content), s.now().UnixMilli()); mirrorErr != nil {
				return mirrorErr
			}
			s.notifyAgentOffline(agent, conversation)
			return nil
		}
	}
	return err
}

// HandleWechatOutbound mirrors messages sent manually from the Bot management
// console into the currently taken-over Agent conversation.
func (s *ConversationService) HandleWechatOutbound(botID snowflake.ID, userID, content string) error {
	if botID == 0 || strings.TrimSpace(userID) == "" || strings.TrimSpace(content) == "" {
		return nil
	}
	_, err := s.repository.AppendTakeoverOutbound(botID, strings.TrimSpace(userID), strings.TrimSpace(content), s.now().UnixMilli())
	return err
}

func (s *ConversationService) handleAgentStatusChange(agent remoteModel.Agent, previousStatus string) {
	if agent.Status != remoteModel.AgentStatusOffline ||
		(previousStatus != remoteModel.AgentStatusOnline && previousStatus != remoteModel.AgentStatusBusy) {
		return
	}
	conversations, err := s.repository.TakeoversByAgent(agent.ID)
	if err != nil {
		if global.LOG != nil {
			global.LOG.Error("查询 Agent 离线接管会话失败", zap.String("agentId", agent.ID.String()), zap.Error(err))
		}
		return
	}
	for _, conversation := range conversations {
		s.notifyAgentOffline(agent, conversation)
	}
}

func (s *ConversationService) notifyAgentOffline(agent remoteModel.Agent, conversation remoteModel.Conversation) {
	if s.takeoverSender == nil || conversation.WechatBotID == 0 || strings.TrimSpace(conversation.WechatUserID) == "" {
		return
	}
	name := strings.TrimSpace(agent.Name)
	if name == "" {
		name = agent.AgentKey
	}
	content := fmt.Sprintf("Agent %q 已离线，当前接管会话暂时无法处理新消息，请等待 Agent 恢复在线。", name)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	err := s.takeoverSender(ctx, conversation.WechatBotID, conversation.WechatUserID, content)
	cancel()
	if err != nil {
		if global.LOG != nil {
			global.LOG.Error("发送 Agent 离线微信通知失败", zap.String("agentId", agent.ID.String()), zap.Error(err))
		}
		return
	}
	if err := s.repository.AppendTakeoverNotice(conversation, content, "system", s.now().UnixMilli()); err != nil && global.LOG != nil {
		global.LOG.Error("同步 Agent 离线通知到接管会话失败", zap.String("agentId", agent.ID.String()), zap.Error(err))
	}
}

func (s *ConversationService) Create(request *remoteReq.ConversationCreateRequest) (remoteModel.Conversation, error) {
	if request == nil || request.AgentID == 0 {
		return remoteModel.Conversation{}, errors.New("Agent ID 不能为空")
	}
	if _, err := s.agentService.repository.Get(request.AgentID); err != nil {
		return remoteModel.Conversation{}, err
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "新会话"
	}
	if len([]rune(title)) > 200 {
		return remoteModel.Conversation{}, errors.New("标题不能超过 200 个字符")
	}
	cliType := strings.ToUpper(strings.TrimSpace(request.CLIType))
	if cliType != remoteModel.CLITypeCodex && cliType != remoteModel.CLITypeClaude {
		return remoteModel.Conversation{}, errors.New("CLI 类型必须是 CODEX 或 CLAUDE")
	}
	permissionMode := remoteModel.NormalizePermissionMode(request.PermissionMode)
	if strings.TrimSpace(request.PermissionMode) != "" && permissionMode != strings.ToUpper(strings.TrimSpace(request.PermissionMode)) {
		return remoteModel.Conversation{}, errors.New("权限模式必须是 AUTO_EDIT 或 FULL_ACCESS")
	}
	conversationID := db.GetId()
	workingDirectory := strings.TrimSpace(request.WorkingDirectory)
	if workingDirectory == "" {
		workingDirectory = "."
	}
	if len([]rune(workingDirectory)) > 500 {
		return remoteModel.Conversation{}, errors.New("工作目录不能超过 500 个字符")
	}
	if strings.ContainsRune(workingDirectory, '\x00') {
		return remoteModel.Conversation{}, errors.New("工作目录包含无效字符")
	}
	normalizedDirectory := strings.ReplaceAll(workingDirectory, "\\", "/")
	if strings.HasPrefix(normalizedDirectory, "/") ||
		(len(normalizedDirectory) >= 2 && normalizedDirectory[1] == ':') {
		return remoteModel.Conversation{}, errors.New("工作目录必须是相对于 workspace-root 的路径")
	}
	for _, segment := range strings.Split(normalizedDirectory, "/") {
		if segment == ".." {
			return remoteModel.Conversation{}, errors.New("工作目录不能越出 workspace-root")
		}
	}
	now := s.now().UnixMilli()
	conversation := remoteModel.Conversation{
		MODEL:   coreAPI.MODEL{ID: conversationID, CreatedBy: "admin", CreatedAt: now, UpdatedAt: now},
		AgentID: request.AgentID, Title: title, CLIType: cliType, PermissionMode: permissionMode, WorkingDirectory: workingDirectory,
		Status: remoteModel.ConversationStatusActive, LastMessageAt: now, ControlMode: remoteModel.ConversationControlModeWeb,
	}
	return conversation, s.repository.Create(&conversation)
}

func (s *ConversationService) Page(params *remoteReq.ConversationPageParams) (coreResp.PageResult, error) {
	if params == nil || params.AgentID == 0 {
		return coreResp.PageResult{}, errors.New("Agent ID 不能为空")
	}
	return s.repository.Page(params)
}

func (s *ConversationService) Get(id snowflake.ID) (remoteResp.ConversationDetail, error) {
	if id == 0 {
		return remoteResp.ConversationDetail{}, errors.New("会话 ID 不能为空")
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
		return remoteModel.Conversation{}, errors.New("会话 ID 不能为空")
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
		return remoteModel.Conversation{}, errors.New("会话 ID 不能为空")
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		return remoteModel.Conversation{}, errors.New("会话标题不能为空")
	}
	if len([]rune(title)) > 200 {
		return remoteModel.Conversation{}, errors.New("标题不能超过 200 个字符")
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
		return false, errors.New("会话 ID 不能为空")
	}
	return true, s.repository.Delete(request.ID)
}

func (s *ConversationService) Send(request *remoteReq.SendMessageRequest) (remoteResp.SendMessageResult, error) {
	if request == nil || request.ConversationID == 0 {
		return remoteResp.SendMessageResult{}, errors.New("会话 ID 不能为空")
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return remoteResp.SendMessageResult{}, errors.New("消息内容不能为空")
	}
	if len([]byte(content)) > 128*1024 {
		return remoteResp.SendMessageResult{}, errors.New("消息内容不能超过 128 KB")
	}
	return s.send(request.ConversationID, content, "WEB")
}

func (s *ConversationService) Pause(request *remoteReq.ConversationTaskRequest) (remoteModel.Message, error) {
	if request == nil || request.MessageID == 0 {
		return remoteModel.Message{}, errors.New("任务消息 ID 不能为空")
	}
	message, err := s.repository.Pause(request.MessageID, s.now().UnixMilli())
	if err == nil && message.Status == remoteModel.MessageStatusPaused {
		s.agentService.publishAgentByID(message.AgentID, remoteModel.AgentStatusBusy)
	}
	return message, err
}

func (s *ConversationService) Resume(request *remoteReq.ConversationTaskRequest) (remoteModel.Message, error) {
	if request == nil || request.MessageID == 0 {
		return remoteModel.Message{}, errors.New("任务消息 ID 不能为空")
	}
	message, err := s.repository.Resume(request.MessageID, s.now().UnixMilli())
	if err == nil {
		s.agentService.publishAgentByID(message.AgentID, remoteModel.AgentStatusOnline)
	}
	return message, err
}

func (s *ConversationService) Cancel(request *remoteReq.ConversationTaskRequest) (remoteModel.Message, error) {
	if request == nil || request.MessageID == 0 {
		return remoteModel.Message{}, errors.New("任务消息 ID 不能为空")
	}
	message, err := s.repository.Cancel(request.MessageID, s.now().UnixMilli())
	return message, err
}

func (s *ConversationService) send(conversationID snowflake.ID, content, source string) (remoteResp.SendMessageResult, error) {
	conversation, err := s.repository.Get(conversationID)
	if err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	if err := validateConversationExecution(conversation); err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	if conversation.ControlMode == "" {
		conversation.ControlMode = remoteModel.ConversationControlModeWeb
	}
	if source == "WEB" && conversation.ControlMode == remoteModel.ConversationControlModeWechat {
		return remoteResp.SendMessageResult{}, errors.New("该会话已由微信 Bot 接管，Web 端已暂停控制")
	}
	if source == "WECHAT" && conversation.ControlMode != remoteModel.ConversationControlModeWechat {
		return remoteResp.SendMessageResult{}, errors.New("该会话未开启微信接管")
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
		MODEL:          coreAPI.MODEL{ID: db.GetId(), CreatedBy: map[bool]string{true: "wechat-bot", false: "admin"}[source == "WECHAT"], CreatedAt: now, UpdatedAt: now},
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
	if err := s.repository.CreateTurn(conversation, &userMessage, &assistantMessage, &command, now, source); err != nil {
		return remoteResp.SendMessageResult{}, err
	}
	s.agentService.publishAgentByID(conversation.AgentID, remoteModel.AgentStatusOnline)
	return remoteResp.SendMessageResult{UserMessage: userMessage, AssistantMessage: assistantMessage}, nil
}

func (s *ConversationService) NextCommand(params *remoteReq.NextCommandParams) (*remoteResp.CommandDispatch, error) {
	if params == nil {
		return nil, errors.New("命令参数不能为空")
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
			conversation, err := s.repository.Get(command.ConversationID)
			if err != nil {
				return nil, err
			}
			if err := validateConversationExecution(conversation); err != nil {
				return nil, err
			}
			next, err := s.repository.NextChunkSequence(command.AssistantMessageID)
			if err != nil {
				return nil, err
			}
			return &remoteResp.CommandDispatch{
				CommandID: command.ID, ConversationID: command.ConversationID, UserMessageID: command.UserMessageID,
				AssistantMessageID: command.AssistantMessageID, Type: command.Type,
				CLIType: conversation.CLIType, PermissionMode: remoteModel.NormalizePermissionMode(conversation.PermissionMode), WorkingDirectory: conversation.WorkingDirectory,
				Prompt: command.Payload, NextChunkSequence: next,
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
		return false, errors.New("命令 ID 不能为空")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	return true, s.repository.Acknowledge(agent.ID, params.CommandID, s.now().UnixMilli())
}

func (s *ConversationService) CommandStatus(params *remoteReq.CommandStatusParams) (remoteResp.CommandControlStatus, error) {
	if params == nil || params.CommandID == 0 {
		return remoteResp.CommandControlStatus{}, errors.New("命令 ID 不能为空")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return remoteResp.CommandControlStatus{}, err
	}
	status, err := s.repository.CommandStatus(agent.ID, params.CommandID)
	return remoteResp.CommandControlStatus{Status: status}, err
}

func (s *ConversationService) AppendChunks(params *remoteReq.MessageChunkUploadParams) (bool, error) {
	if params == nil || params.Request.MessageID == 0 {
		return false, errors.New("消息 ID 不能为空")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	if len(params.Request.Chunks) == 0 || len(params.Request.Chunks) > 200 {
		return false, errors.New("消息分片数量必须在 1 到 200 之间")
	}
	now := s.now().UnixMilli()
	chunks := make([]remoteModel.MessageChunk, 0, len(params.Request.Chunks))
	for _, entry := range params.Request.Chunks {
		if entry.Sequence <= 0 || entry.Content == "" {
			return false, errors.New("消息分片无效")
		}
		if len([]byte(entry.Content)) > maxMessageChunkBytes {
			return false, errors.New("单个消息分片不能超过 32 KB")
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
		return false, errors.New("消息 ID 不能为空")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	if len([]byte(params.Request.Content)) > maxAssistantMessageBytes {
		return false, errors.New("助手消息不能超过 1 MB")
	}
	if len([]byte(params.Request.ErrorMessage)) > maxMessageErrorBytes {
		return false, errors.New("消息错误信息不能超过 64 KB")
	}
	conversation, err := s.repository.ConversationByMessage(params.Request.MessageID)
	if err != nil {
		return false, err
	}
	finalStatus, err := s.repository.Complete(agent.ID, params.Request, s.now().UnixMilli())
	if err != nil {
		return false, err
	}
	completed := finalStatus != ""
	if completed {
		s.agentService.publishAgentByID(agent.ID, remoteModel.AgentStatusBusy)
	}
	if completed && finalStatus != remoteModel.MessageStatusPaused && finalStatus != remoteModel.MessageStatusCancelled && conversation.ControlMode == remoteModel.ConversationControlModeWechat && conversation.WechatBotID != 0 && conversation.WechatUserID != "" && s.takeoverSender != nil {
		content := params.Request.Content
		if !params.Request.Success {
			content = "Agent 执行失败：" + params.Request.ErrorMessage
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if err := s.takeoverSender(ctx, conversation.WechatBotID, conversation.WechatUserID, content); err != nil && global.LOG != nil {
			global.LOG.Error("微信接管回复发送失败", zap.Error(err))
		}
		cancel()
	}
	return true, nil
}

func (s *ConversationService) Stream(params *remoteReq.MessageStreamParams) (remoteResp.MessageStreamEvent, error) {
	if params == nil || params.MessageID == 0 {
		return remoteResp.MessageStreamEvent{}, errors.New("消息 ID 不能为空")
	}
	if params.AfterSequence < 0 {
		params.AfterSequence = 0
	}
	return s.repository.StreamEvent(params.MessageID, params.AfterSequence)
}

func validateConversationExecution(conversation remoteModel.Conversation) error {
	switch strings.ToUpper(strings.TrimSpace(conversation.CLIType)) {
	case remoteModel.CLITypeCodex, remoteModel.CLITypeClaude:
	default:
		return errors.New("会话 CLI 类型必须是 CODEX 或 CLAUDE")
	}
	if strings.TrimSpace(conversation.WorkingDirectory) == "" {
		return errors.New("会话工作目录不能为空")
	}
	if conversation.PermissionMode != "" && remoteModel.NormalizePermissionMode(conversation.PermissionMode) != conversation.PermissionMode {
		return errors.New("会话权限模式必须是 AUTO_EDIT 或 FULL_ACCESS")
	}
	return nil
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
