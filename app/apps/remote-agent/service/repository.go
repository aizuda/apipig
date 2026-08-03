package service

import (
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
	FindByTokenHash(string) (remoteModel.Agent, error)
	Create(*remoteModel.Agent) error
	UpdateRegistration(*remoteModel.Agent) error
	RecoverTask(snowflake.ID, snowflake.ID, int64) error
	RecordHeartbeat(remoteModel.Agent, remoteModel.Heartbeat) error
	MarkOffline(int64, int64) error
	Page(*remoteReq.AgentPageParams) (response.PageResult, error)
	Get(snowflake.ID) (remoteModel.Agent, error)
	RecentHeartbeats(snowflake.ID, int) ([]remoteModel.Heartbeat, error)
	RecentTasks(snowflake.ID, int) ([]remoteModel.Task, error)
}

type gormAgentRepository struct{}

func (gormAgentRepository) db() *gorm.DB { return global.DB }

func (r gormAgentRepository) FindByKey(key string) (remoteModel.Agent, error) {
	var agent remoteModel.Agent
	err := r.db().Where("agent_key = ?", key).First(&agent).Error
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

func (r gormAgentRepository) UpdateRegistration(agent *remoteModel.Agent) error {
	result := r.db().Model(&remoteModel.Agent{}).
		Where("id = ? AND status <> ?", agent.ID, remoteModel.AgentStatusDisabled).
		Updates(map[string]any{
			"name": agent.Name, "token_hash": agent.TokenHash, "ip_address": agent.IPAddress,
			"hostname": agent.Hostname, "operating_system": agent.OperatingSystem,
			"architecture": agent.Architecture, "cpu_info": agent.CPUInfo,
			"memory_total": agent.MemoryTotal, "codex_version": agent.CodexVersion,
			"agent_version": agent.AgentVersion, "status": agent.Status,
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

func (r gormAgentRepository) RecoverTask(agentID, taskID snowflake.ID, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&remoteModel.Task{}).
			Where("id = ? AND agent_id = ? AND status = ?", taskID, agentID, remoteModel.TaskStatusRunning).
			Updates(map[string]any{
				"status": remoteModel.TaskStatusPending, "started_at": 0,
				"error_message": "", "updated_at": now,
			})
		if update.Error != nil || update.RowsAffected == 0 {
			return update.Error
		}
		return tx.Model(&remoteModel.Command{}).
			Where("task_id = ? AND agent_id = ? AND type = ?", taskID, agentID, remoteModel.CommandTypeExecuteTask).
			Updates(map[string]any{
				"status": remoteModel.CommandStatusPending, "dispatched_at": 0,
				"acknowledged_at": 0, "updated_at": now,
			}).Error
	})
}

func (r gormAgentRepository) RecordHeartbeat(agent remoteModel.Agent, heartbeat remoteModel.Heartbeat) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
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

func (r gormAgentRepository) MarkOffline(cutoff, updatedAt int64) error {
	return r.db().Model(&remoteModel.Agent{}).
		Where("status IN ? AND last_seen_at < ?", []string{remoteModel.AgentStatusOnline, remoteModel.AgentStatusBusy}, cutoff).
		Updates(map[string]any{"status": remoteModel.AgentStatusOffline, "updated_at": updatedAt}).Error
}

func (r gormAgentRepository) Page(params *remoteReq.AgentPageParams) (response.PageResult, error) {
	query := r.db().Model(&remoteModel.Agent{})
	if params != nil {
		if params.Status != "" {
			query = query.Where("status = ?", params.Status)
		}
		if params.Keyword != "" {
			like := "%" + params.Keyword + "%"
			query = query.Where("name LIKE ? OR hostname LIKE ? OR ip_address LIKE ?", like, like, like)
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

func (r gormAgentRepository) RecentTasks(agentID snowflake.ID, limit int) ([]remoteModel.Task, error) {
	var tasks []remoteModel.Task
	err := r.db().Where("agent_id = ?", agentID).Order("created_at DESC").Limit(limit).Find(&tasks).Error
	return tasks, err
}
