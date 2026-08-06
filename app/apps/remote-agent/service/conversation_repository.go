package service

import (
	"errors"
	"strings"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	remoteResp "apipig/app/apps/remote-agent/model/response"
	coreReq "apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type conversationRepository struct{}

func (conversationRepository) db() *gorm.DB { return global.DB }

func (r conversationRepository) Create(conversation *remoteModel.Conversation) error {
	return r.db().Create(conversation).Error
}

func (r conversationRepository) Page(params *remoteReq.ConversationPageParams) (response.PageResult, error) {
	query := r.db().Model(&remoteModel.Conversation{}).Where("agent_id = ?", params.AgentID)
	var conversations []remoteModel.Conversation
	pageInfo := coreReq.PageInfo{Page: 1, PageSize: 20}
	if params != nil {
		pageInfo = params.GetPageInfo()
	}
	return db.Page(query.Order("pinned DESC, pinned_at DESC, last_message_at DESC, created_at DESC"), pageInfo, conversations)
}

func (r conversationRepository) Get(id snowflake.ID) (remoteModel.Conversation, error) {
	var conversation remoteModel.Conversation
	err := r.db().First(&conversation, id).Error
	return conversation, err
}

func (r conversationRepository) Messages(conversationID snowflake.ID) ([]remoteModel.Message, error) {
	var messages []remoteModel.Message
	err := r.db().Where("conversation_id = ?", conversationID).Order("sequence ASC").Find(&messages).Error
	return messages, err
}

func (r conversationRepository) Pin(id snowflake.ID, pinned bool, pinnedAt, now int64) error {
	return r.db().Model(&remoteModel.Conversation{}).Where("id = ?", id).
		Updates(map[string]any{"pinned": pinned, "pinned_at": pinnedAt, "updated_at": now}).Error
}

func (r conversationRepository) Rename(id snowflake.ID, title string, now int64) error {
	return r.db().Model(&remoteModel.Conversation{}).Where("id = ?", id).
		Updates(map[string]any{"title": title, "updated_at": now}).Error
}

func (r conversationRepository) Delete(id snowflake.ID) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var conversation remoteModel.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, id).Error; err != nil {
			return err
		}

		var agent remoteModel.Agent
		if err := tx.First(&agent, conversation.AgentID).Error; err != nil {
			return err
		}
		if agent.CurrentMessageID != 0 {
			var activeMessageCount int64
			if err := tx.Model(&remoteModel.Message{}).
				Where("id = ? AND conversation_id = ?", agent.CurrentMessageID, id).
				Count(&activeMessageCount).Error; err != nil {
				return err
			}
			if activeMessageCount > 0 {
				return errors.New("Agent 正在响应时不能删除会话")
			}
		}

		var commands []remoteModel.Command
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("conversation_id = ?", id).Find(&commands).Error; err != nil {
			return err
		}
		for _, command := range commands {
			if command.Status == remoteModel.CommandStatusDispatched || command.Status == remoteModel.CommandStatusAcknowledged {
				return errors.New("Agent 正在响应时不能删除会话")
			}
		}

		messageIDs := tx.Unscoped().Model(&remoteModel.Message{}).Select("id").Where("conversation_id = ?", id)
		if err := tx.Unscoped().Where("message_id IN (?)", messageIDs).Delete(&remoteModel.MessageChunk{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("conversation_id = ?", id).Delete(&remoteModel.Command{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("conversation_id = ?", id).Delete(&remoteModel.Message{}).Error; err != nil {
			return err
		}
		return tx.Unscoped().Delete(&conversation).Error
	})
}

func (r conversationRepository) CreateTurn(
	conversation remoteModel.Conversation,
	userMessage, assistantMessage remoteModel.Message,
	command remoteModel.Command,
	now int64, source string,
) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var lockedConversation remoteModel.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedConversation, conversation.ID).Error; err != nil {
			return err
		}
		if lockedConversation.AgentID != conversation.AgentID {
			return errors.New("会话不属于当前 Agent")
		}
		conversation = lockedConversation
		if conversation.ControlMode == "" {
			conversation.ControlMode = remoteModel.ConversationControlModeWeb
		}
		if source == "WEB" && conversation.ControlMode == remoteModel.ConversationControlModeWechat {
			return errors.New("该会话已由微信 Bot 接管，Web 端已暂停控制")
		}
		if source == "WECHAT" && conversation.ControlMode != remoteModel.ConversationControlModeWechat {
			return errors.New("该会话未开启微信接管")
		}
		reservation := tx.Model(&remoteModel.Agent{}).
			Where("id = ? AND status = ? AND current_message_id = 0", conversation.AgentID, remoteModel.AgentStatusOnline).
			Updates(map[string]any{"status": remoteModel.AgentStatusBusy, "current_message_id": assistantMessage.ID, "updated_at": now})
		if reservation.Error != nil {
			return reservation.Error
		}
		if reservation.RowsAffected == 0 {
			return errors.New("Agent 不在线或正在处理其他消息")
		}
		if err := tx.Create(&userMessage).Error; err != nil {
			return err
		}
		if err := tx.Create(&assistantMessage).Error; err != nil {
			return err
		}
		if err := tx.Create(&command).Error; err != nil {
			return err
		}
		updates := map[string]any{"last_message_at": now, "updated_at": now}
		if conversation.Title == "新会话" {
			title := []rune(strings.TrimSpace(userMessage.Content))
			if len(title) > 40 {
				title = title[:40]
			}
			updates["title"] = string(title)
		}
		return tx.Model(&remoteModel.Conversation{}).Where("id = ?", conversation.ID).Updates(updates).Error
	})
}

func (r conversationRepository) SetTakeover(id, botID snowflake.ID, userID string, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var conversation remoteModel.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, id).Error; err != nil {
			return err
		}
		var agent remoteModel.Agent
		if err := tx.First(&agent, conversation.AgentID).Error; err != nil {
			return err
		}
		if agent.CurrentMessageID != 0 {
			return errors.New("Agent 正在响应时不能切换接管模式")
		}
		var existing remoteModel.Conversation
		if err := tx.Where("control_mode = ? AND wechat_bot_id = ? AND wechat_user_id = ? AND id <> ?", remoteModel.ConversationControlModeWechat, botID, userID, id).First(&existing).Error; err == nil {
			return errors.New("该微信联系人已被其他会话接管")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Model(&conversation).Updates(map[string]any{"control_mode": remoteModel.ConversationControlModeWechat, "wechat_bot_id": botID, "wechat_user_id": userID, "wechat_takeover_at": now, "updated_at": now}).Error
	})
}

func (r conversationRepository) StopTakeover(id snowflake.ID, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var conversation remoteModel.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, id).Error; err != nil {
			return err
		}
		var agent remoteModel.Agent
		if err := tx.First(&agent, conversation.AgentID).Error; err != nil {
			return err
		}
		if agent.CurrentMessageID != 0 {
			return errors.New("Agent 正在响应时不能结束微信接管")
		}
		return tx.Model(&conversation).Updates(map[string]any{"control_mode": remoteModel.ConversationControlModeWeb, "wechat_bot_id": 0, "wechat_user_id": "", "wechat_takeover_at": 0, "updated_at": now}).Error
	})
}

func (r conversationRepository) ConversationByMessage(messageID snowflake.ID) (remoteModel.Conversation, error) {
	var message remoteModel.Message
	if err := r.db().First(&message, messageID).Error; err != nil {
		return remoteModel.Conversation{}, err
	}
	return r.Get(message.ConversationID)
}

func (r conversationRepository) Claim(agentID snowflake.ID, leaseCutoff, now int64) (*remoteModel.Command, error) {
	var claimed remoteModel.Command
	err := r.db().Transaction(func(tx *gorm.DB) error {
		var command remoteModel.Command
		err := tx.Where(
			"agent_id = ? AND ((status = ?) OR (status = ? AND dispatched_at < ?))",
			agentID, remoteModel.CommandStatusPending, remoteModel.CommandStatusDispatched, leaseCutoff,
		).Order("created_at ASC").First(&command).Error
		if err != nil {
			return err
		}
		result := tx.Model(&remoteModel.Command{}).Where(
			"id = ? AND ((status = ?) OR (status = ? AND dispatched_at < ?))",
			command.ID, remoteModel.CommandStatusPending, remoteModel.CommandStatusDispatched, leaseCutoff,
		).Updates(map[string]any{
			"status": remoteModel.CommandStatusDispatched, "attempt": gorm.Expr("attempt + 1"),
			"dispatched_at": now, "updated_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&remoteModel.Message{}).Where("id = ?", command.AssistantMessageID).
			Updates(map[string]any{"status": remoteModel.MessageStatusStreaming, "updated_at": now}).Error; err != nil {
			return err
		}
		command.Status = remoteModel.CommandStatusDispatched
		command.Attempt++
		command.DispatchedAt = now
		claimed = command
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (r conversationRepository) Acknowledge(agentID, commandID snowflake.ID, now int64) error {
	result := r.db().Model(&remoteModel.Command{}).
		Where("id = ? AND agent_id = ? AND status = ?", commandID, agentID, remoteModel.CommandStatusDispatched).
		Updates(map[string]any{"status": remoteModel.CommandStatusAcknowledged, "acknowledged_at": now, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("会话命令当前无法确认")
	}
	return nil
}

func (r conversationRepository) NextChunkSequence(messageID snowflake.ID) (int64, error) {
	var maximum int64
	err := r.db().Model(&remoteModel.MessageChunk{}).Where("message_id = ?", messageID).
		Select("COALESCE(MAX(sequence), 0)").Scan(&maximum).Error
	return maximum + 1, err
}

func (r conversationRepository) AppendChunks(agentID snowflake.ID, messageID snowflake.ID, chunks []remoteModel.MessageChunk, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var message remoteModel.Message
		if err := tx.Where("id = ? AND agent_id = ? AND role = ?", messageID, agentID, remoteModel.MessageRoleAssistant).First(&message).Error; err != nil {
			return err
		}
		if message.Status != remoteModel.MessageStatusStreaming && message.Status != remoteModel.MessageStatusPending {
			return errors.New("助手消息已结束流式处理")
		}
		if len(chunks) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&chunks).Error; err != nil {
				return err
			}
		}
		var stored []remoteModel.MessageChunk
		if err := tx.Where("message_id = ?", messageID).Order("sequence ASC").Find(&stored).Error; err != nil {
			return err
		}
		var content strings.Builder
		for _, chunk := range stored {
			content.WriteString(chunk.Content)
			if content.Len() > maxAssistantMessageBytes {
				return errors.New("助手消息不能超过 1 MB")
			}
		}
		return tx.Model(&remoteModel.Message{}).Where("id = ?", messageID).
			Updates(map[string]any{"content": content.String(), "status": remoteModel.MessageStatusStreaming, "updated_at": now}).Error
	})
}

func (r conversationRepository) Complete(agentID snowflake.ID, result remoteReq.MessageResultRequest, now int64) (bool, error) {
	completed := false
	err := r.db().Transaction(func(tx *gorm.DB) error {
		var message remoteModel.Message
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND agent_id = ? AND role = ?", result.MessageID, agentID, remoteModel.MessageRoleAssistant).First(&message).Error; err != nil {
			return err
		}
		if message.Status == remoteModel.MessageStatusCompleted || message.Status == remoteModel.MessageStatusFailed {
			return nil
		}
		status := remoteModel.MessageStatusCompleted
		commandStatus := remoteModel.CommandStatusCompleted
		if !result.Success {
			status, commandStatus = remoteModel.MessageStatusFailed, remoteModel.CommandStatusFailed
		}
		content := result.Content
		if content == "" {
			content = message.Content
		}
		if err := tx.Model(&remoteModel.Message{}).Where("id = ?", result.MessageID).Updates(map[string]any{
			"status": status, "content": content, "error_message": result.ErrorMessage, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&remoteModel.Command{}).Where("assistant_message_id = ? AND agent_id = ?", result.MessageID, agentID).
			Updates(map[string]any{"status": commandStatus, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&remoteModel.Agent{}).Where("id = ? AND current_message_id = ?", agentID, result.MessageID).
			Updates(map[string]any{"status": remoteModel.AgentStatusOnline, "current_message_id": 0, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&remoteModel.Conversation{}).Where("id = ?", message.ConversationID).
			Updates(map[string]any{"last_message_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		completed = true
		return nil
	})
	return completed, err
}

func (r conversationRepository) StreamEvent(messageID snowflake.ID, afterSequence int64) (remoteResp.MessageStreamEvent, error) {
	var event remoteResp.MessageStreamEvent
	if err := r.db().First(&event.Message, messageID).Error; err != nil {
		return event, err
	}
	err := r.db().Where("message_id = ? AND sequence > ?", messageID, afterSequence).
		Order("sequence ASC").Limit(200).Find(&event.Chunks).Error
	return event, err
}
