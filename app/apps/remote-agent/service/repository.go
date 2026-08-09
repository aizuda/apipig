package service

import (
	"encoding/json"
	"errors"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	coreReq "apipig/core/api/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

type agentRepository interface {
	FindByKey(string) (remoteModel.Agent, error)
	FindByRegistrationTokenHash(string) (remoteModel.Agent, error)
	FindByTokenHash(string) (remoteModel.Agent, error)
	Create(*remoteModel.Agent) error
	UpdateConfiguration(*remoteModel.Agent) error
	UpdateRegistration(*remoteModel.Agent) error
	RecoverInterruptedTurn(snowflake.ID, snowflake.ID, int64) error
	UpdateRegistrationToken(snowflake.ID, string, int64) error
	SetStatus(snowflake.ID, string, int64) error
	Disconnect(snowflake.ID, int64) error
	Delete(snowflake.ID) error
	RecordHeartbeat(remoteModel.Agent, remoteModel.Heartbeat) error
	ActiveAgents() ([]remoteModel.Agent, error)
	MarkAgentOffline(snowflake.ID, int64, int64) (remoteModel.Agent, string, bool, error)
	Page(*remoteReq.AgentPageParams) (response.PageResult, error)
	Get(snowflake.ID) (remoteModel.Agent, error)
	RecentHeartbeats(snowflake.ID, int) ([]remoteModel.Heartbeat, error)
	RecentConversations(snowflake.ID, int) ([]remoteModel.Conversation, error)
}

// gormAgentRepository 使用全局 GORM 连接实现 Agent 数据访问。
type gormAgentRepository struct{}

func (gormAgentRepository) db() *gorm.DB { return global.DB }

func (r gormAgentRepository) FindByKey(key string) (remoteModel.Agent, error) {
	var agent remoteModel.Agent
	err := r.db().Where("agent_key = ?", key).First(&agent).Error
	return agent, err
}

func (r gormAgentRepository) FindByRegistrationTokenHash(tokenHash string) (remoteModel.Agent, error) {
	var agent remoteModel.Agent
	err := r.db().Where("registration_token_hash = ?", tokenHash).First(&agent).Error
	return agent, err
}

func (r gormAgentRepository) FindByTokenHash(tokenHash string) (remoteModel.Agent, error) {
	var agent remoteModel.Agent
	err := r.db().Where("token_hash = ?", tokenHash).First(&agent).Error
	return agent, err
}

func (r gormAgentRepository) Create(agent *remoteModel.Agent) error {
	return r.db().Create(agent).Error
}

func (r gormAgentRepository) UpdateConfiguration(agent *remoteModel.Agent) error {
	// 显式序列化切片，确保不同数据库驱动下 Codex 参数的存储格式一致。
	codexArgs, err := json.Marshal(agent.CodexArgs)
	if err != nil {
		return err
	}
	claudeArgs, err := json.Marshal(agent.ClaudeArgs)
	if err != nil {
		return err
	}
	return r.db().Model(&remoteModel.Agent{}).Where("id = ?", agent.ID).Updates(map[string]any{
		"agent_key": agent.AgentKey, "name": agent.Name,
		"workspace_root": agent.WorkspaceRoot, "codex_command": agent.CodexCommand,
		"codex_args": string(codexArgs), "claude_command": agent.ClaudeCommand,
		"claude_args": string(claudeArgs), "poll_wait_seconds": agent.PollWaitSeconds,
		"request_timeout_seconds": agent.RequestTimeoutSeconds, "log_file": agent.LogFile,
		"updated_at": agent.UpdatedAt,
	}).Error
}

func (r gormAgentRepository) UpdateRegistration(agent *remoteModel.Agent) error {
	// 将“未禁用”放入更新条件，避免注册请求与管理员禁用操作并发时重新启用 Agent。
	result := r.db().Model(&remoteModel.Agent{}).
		Where("id = ? AND status <> ?", agent.ID, remoteModel.AgentStatusDisabled).
		Updates(map[string]any{
			"token_hash": agent.TokenHash, "ip_address": agent.IPAddress, "hostname": agent.Hostname,
			"operating_system": agent.OperatingSystem, "architecture": agent.Architecture,
			"cpu_info": agent.CPUInfo, "memory_total": agent.MemoryTotal,
			"codex_version": agent.CodexVersion, "agent_version": agent.AgentVersion,
			"status": agent.Status, "current_message_id": agent.CurrentMessageID,
			"last_seen_at": agent.LastSeenAt, "updated_at": agent.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errAgentDisabled
	}
	return nil
}

func (r gormAgentRepository) RecoverInterruptedTurn(agentID, messageID snowflake.ID, now int64) error {
	// 恢复消息、命令和 Agent 占用状态必须在同一事务内完成，避免命令重复或永久卡在忙碌状态。
	return r.db().Transaction(func(tx *gorm.DB) error {
		var message remoteModel.Message
		if err := tx.Where("id = ? AND agent_id = ? AND role = ?", messageID, agentID, remoteModel.MessageRoleAssistant).
			First(&message).Error; err != nil {
			return err
		}
		if message.Status == remoteModel.MessageStatusCompleted || message.Status == remoteModel.MessageStatusFailed || message.Status == remoteModel.MessageStatusCancelled {
			// 终态消息无需重试，仅清除客户端异常退出后残留的占用标记。
			return tx.Model(&remoteModel.Agent{}).Where("id = ? AND current_message_id = ?", agentID, messageID).
				Updates(map[string]any{"current_message_id": 0, "updated_at": now}).Error
		}
		var storedCommand remoteModel.Command
		if err := tx.Where("assistant_message_id = ? AND agent_id = ?", messageID, agentID).
			First(&storedCommand).Error; err != nil {
			return err
		}
		if storedCommand.Status == remoteModel.CommandStatusCancelled || message.Status == remoteModel.MessageStatusCancelling {
			if err := tx.Model(&remoteModel.Message{}).Where("id = ?", messageID).
				Updates(map[string]any{"status": remoteModel.MessageStatusCancelled, "error_message": "任务已取消", "updated_at": now}).Error; err != nil {
				return err
			}
			return tx.Model(&remoteModel.Agent{}).Where("id = ? AND current_message_id = ?", agentID, messageID).
				Updates(map[string]any{"current_message_id": 0, "updated_at": now}).Error
		}
		if storedCommand.Status == remoteModel.CommandStatusPauseRequested || storedCommand.Status == remoteModel.CommandStatusPaused {
			if err := tx.Model(&remoteModel.Message{}).Where("id = ?", messageID).
				Updates(map[string]any{"status": remoteModel.MessageStatusPaused, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&remoteModel.Command{}).Where("id = ?", storedCommand.ID).
				Updates(map[string]any{"status": remoteModel.CommandStatusPaused, "updated_at": now}).Error; err != nil {
				return err
			}
			return tx.Model(&remoteModel.Agent{}).Where("id = ? AND current_message_id = ?", agentID, messageID).
				Updates(map[string]any{"current_message_id": 0, "updated_at": now}).Error
		}
		// 未完成输出从头重试，因此先清除已有分片和助手消息中的部分内容。
		if err := tx.Where("message_id = ?", messageID).Delete(&remoteModel.MessageChunk{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&remoteModel.Message{}).Where("id = ?", messageID).Updates(map[string]any{
			"status": remoteModel.MessageStatusPending, "content": "", "error_message": "", "updated_at": now,
		}).Error; err != nil {
			return err
		}
		command := tx.Model(&remoteModel.Command{}).
			Where("assistant_message_id = ? AND agent_id = ? AND status IN ?", messageID, agentID, []string{
				remoteModel.CommandStatusPending,
				remoteModel.CommandStatusDispatched,
				remoteModel.CommandStatusAcknowledged,
			}).
			Updates(map[string]any{
				"status": remoteModel.CommandStatusPending, "dispatched_at": 0,
				"acknowledged_at": 0, "updated_at": now,
			})
		if command.Error != nil {
			return command.Error
		}
		if command.RowsAffected == 0 {
			return errors.New("中断的会话命令无法恢复")
		}
		return tx.Model(&remoteModel.Agent{}).Where("id = ? AND current_message_id = ?", agentID, messageID).
			Updates(map[string]any{"current_message_id": 0, "updated_at": now}).Error
	})
}

func (r gormAgentRepository) UpdateRegistrationToken(id snowflake.ID, tokenHash string, updatedAt int64) error {
	// 重置注册令牌时清空运行令牌，强制客户端使用新配置重新注册。
	return r.db().Model(&remoteModel.Agent{}).Where("id = ?", id).
		Updates(map[string]any{"registration_token_hash": tokenHash, "token_hash": "", "updated_at": updatedAt}).Error
}

func (r gormAgentRepository) SetStatus(id snowflake.ID, status string, updatedAt int64) error {
	updates := map[string]any{"status": status, "updated_at": updatedAt}
	if status == remoteModel.AgentStatusDisabled {
		// 禁用立即撤销运行令牌，阻止已启动客户端继续访问控制端。
		updates["token_hash"] = ""
	}
	return r.db().Model(&remoteModel.Agent{}).Where("id = ?", id).Updates(updates).Error
}

func (r gormAgentRepository) Disconnect(id snowflake.ID, updatedAt int64) error {
	return r.db().Model(&remoteModel.Agent{}).Where("id = ?", id).Updates(map[string]any{
		"status": remoteModel.AgentStatusOffline, "token_hash": "", "updated_at": updatedAt,
	}).Error
}

func (r gormAgentRepository) Delete(id snowflake.ID) error {
	// 使用物理删除释放 Agent Key，并在同一事务内清除所有关联存档数据。
	return r.db().Transaction(func(tx *gorm.DB) error {
		// current_message_id 条件是服务层检查之外的并发保护，防止响应刚开始时误删 Agent。
		deleted := tx.Unscoped().Where("id = ? AND current_message_id = 0", id).Delete(&remoteModel.Agent{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected == 0 {
			return errors.New("Agent 正在响应时不能删除")
		}

		// 消息分片只关联消息 ID，需要在删除消息前通过子查询先行清理。
		messageIDs := tx.Unscoped().Model(&remoteModel.Message{}).Select("id").Where("agent_id = ?", id)
		if err := tx.Unscoped().Where("message_id IN (?)", messageIDs).Delete(&remoteModel.MessageChunk{}).Error; err != nil {
			return err
		}
		for _, cleanup := range []struct {
			model any
			query string
		}{
			{model: &remoteModel.Command{}, query: "agent_id = ?"},
			{model: &remoteModel.Message{}, query: "agent_id = ?"},
			{model: &remoteModel.Conversation{}, query: "agent_id = ?"},
			{model: &remoteModel.Heartbeat{}, query: "agent_id = ?"},
		} {
			if err := tx.Unscoped().Where(cleanup.query, id).Delete(cleanup.model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r gormAgentRepository) RecordHeartbeat(agent remoteModel.Agent, heartbeat remoteModel.Heartbeat) error {
	// 最新快照与心跳历史原子写入，保证详情页当前状态和历史记录一致。
	return r.db().Transaction(func(tx *gorm.DB) error {
		// 条件更新避免心跳请求与管理员禁用并发时把 Agent 状态改回在线。
		result := tx.Model(&remoteModel.Agent{}).
			Where("id = ? AND status <> ?", agent.ID, remoteModel.AgentStatusDisabled).
			Updates(map[string]any{
				"ip_address": agent.IPAddress, "cpu_usage": agent.CPUUsage,
				"memory_used": agent.MemoryUsed, "codex_version": agent.CodexVersion,
				"status": agent.Status, "last_seen_at": agent.LastSeenAt, "updated_at": agent.UpdatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errAgentDisabled
		}
		return tx.Create(&heartbeat).Error
	})
}

func (r gormAgentRepository) ActiveAgents() ([]remoteModel.Agent, error) {
	var agents []remoteModel.Agent
	err := r.db().Where("status IN ?", []string{remoteModel.AgentStatusOnline, remoteModel.AgentStatusBusy}).Find(&agents).Error
	return agents, err
}

func (r gormAgentRepository) MarkAgentOffline(id snowflake.ID, observedLastSeenAt, updatedAt int64) (remoteModel.Agent, string, bool, error) {
	var agent remoteModel.Agent
	err := r.db().Where("id = ? AND status IN ? AND last_seen_at <= ?", id,
		[]string{remoteModel.AgentStatusOnline, remoteModel.AgentStatusBusy}, observedLastSeenAt).First(&agent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return remoteModel.Agent{}, "", false, nil
	}
	if err != nil {
		return remoteModel.Agent{}, "", false, err
	}
	previousStatus := agent.Status
	result := r.db().Model(&remoteModel.Agent{}).
		Where("id = ? AND status IN ? AND last_seen_at <= ?", id,
			[]string{remoteModel.AgentStatusOnline, remoteModel.AgentStatusBusy}, observedLastSeenAt).
		Updates(map[string]any{"status": remoteModel.AgentStatusOffline, "token_hash": "", "updated_at": updatedAt})
	if result.Error != nil {
		return remoteModel.Agent{}, "", false, result.Error
	}
	if result.RowsAffected == 0 {
		return remoteModel.Agent{}, "", false, nil
	}
	agent.Status = remoteModel.AgentStatusOffline
	agent.TokenHash = ""
	agent.UpdatedAt = updatedAt
	return agent, previousStatus, true, nil
}

func (r gormAgentRepository) Page(params *remoteReq.AgentPageParams) (response.PageResult, error) {
	query := r.db().Model(&remoteModel.Agent{})
	if params != nil {
		if params.Status != "" {
			query = query.Where("status = ?", params.Status)
		}
		if params.Keyword != "" {
			like := "%" + params.Keyword + "%"
			query = query.Where("name LIKE ? OR agent_key LIKE ? OR hostname LIKE ? OR ip_address LIKE ?", like, like, like, like)
		}
	}
	var agents []remoteModel.Agent
	pageInfo := coreReq.PageInfo{Page: 1, PageSize: 10}
	if params != nil {
		pageInfo = params.GetPageInfo()
	}
	return db.Page(query.Order("created_at DESC"), pageInfo, agents)
}

func (r gormAgentRepository) Get(id snowflake.ID) (remoteModel.Agent, error) {
	var agent remoteModel.Agent
	err := r.db().First(&agent, id).Error
	return agent, err
}

func (r gormAgentRepository) RecentHeartbeats(agentID snowflake.ID, limit int) ([]remoteModel.Heartbeat, error) {
	var heartbeats []remoteModel.Heartbeat
	err := r.db().Where("agent_id = ?", agentID).Order("occurred_at DESC").Limit(limit).Find(&heartbeats).Error
	return heartbeats, err
}

func (r gormAgentRepository) RecentConversations(agentID snowflake.ID, limit int) ([]remoteModel.Conversation, error) {
	var conversations []remoteModel.Conversation
	err := r.db().Where("agent_id = ?", agentID).
		Order("pinned DESC, pinned_at DESC, last_message_at DESC").Limit(limit).Find(&conversations).Error
	return conversations, err
}
