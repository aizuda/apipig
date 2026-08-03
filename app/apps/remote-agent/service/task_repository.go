package service

import (
	"errors"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	remoteResp "apipig/app/apps/remote-agent/model/response"
	coreReq "apipig/core/api/request"
	coreResp "apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errAgentUnavailable = errors.New("agent is not online or already has a task")
	errCommandClaimed   = errors.New("command was already claimed")
)

type taskRepository interface {
	Create(remoteModel.Task, remoteModel.Workspace, remoteModel.Command) error
	Page(*remoteReq.TaskPageParams) (coreResp.PageResult, error)
	Get(snowflake.ID) (remoteResp.TaskDetail, error)
	ClaimNext(snowflake.ID, int64, int64) (*remoteReq.ClaimedCommand, error)
	Acknowledge(snowflake.ID, snowflake.ID, int64) error
	Complete(snowflake.ID, remoteReq.TaskResultRequest, string, int64) error
	Cancel(snowflake.ID, remoteModel.Command, int64) error
	AppendLogs(snowflake.ID, snowflake.ID, []remoteModel.TaskLog) error
	LogsAfter(snowflake.ID, int64, int) ([]remoteModel.TaskLog, error)
	TaskStatus(snowflake.ID) (string, error)
}

type gormTaskRepository struct{}

func (gormTaskRepository) db() *gorm.DB { return global.DB }

func (r gormTaskRepository) Create(task remoteModel.Task, workspace remoteModel.Workspace, command remoteModel.Command) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		reservation := tx.Model(&remoteModel.Agent{}).
			Where("id = ? AND status = ? AND current_task_id = ?", task.AgentID, remoteModel.AgentStatusOnline, 0).
			Updates(map[string]any{"status": remoteModel.AgentStatusBusy, "current_task_id": task.ID, "updated_at": task.CreatedAt})
		if reservation.Error != nil {
			return reservation.Error
		}
		if reservation.RowsAffected == 0 {
			return errAgentUnavailable
		}
		if err := tx.Create(&workspace).Error; err != nil {
			return err
		}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		return tx.Create(&command).Error
	})
}

func (r gormTaskRepository) Page(params *remoteReq.TaskPageParams) (coreResp.PageResult, error) {
	query := r.db().Model(&remoteModel.Task{})
	pageInfo := coreReq.PageInfo{Page: 1, PageSize: 10}
	if params != nil {
		pageInfo = params.GetPageInfo()
		if params.AgentID > 0 {
			query = query.Where("agent_id = ?", params.AgentID)
		}
		if params.Status != "" {
			query = query.Where("status = ?", params.Status)
		}
		if params.Keyword != "" {
			like := "%" + params.Keyword + "%"
			query = query.Where("name LIKE ? OR repository_url LIKE ?", like, like)
		}
	}
	var tasks []remoteModel.Task
	return db.Page(query.Order("created_at DESC"), pageInfo, tasks)
}

func (r gormTaskRepository) Get(id snowflake.ID) (remoteResp.TaskDetail, error) {
	var result remoteResp.TaskDetail
	if err := r.db().First(&result.Task, id).Error; err != nil {
		return result, err
	}
	if result.Task.WorkspaceID > 0 {
		if err := r.db().First(&result.Workspace, result.Task.WorkspaceID).Error; err != nil {
			return result, err
		}
	}
	err := r.db().Where("task_id = ?", id).Order("created_at ASC").Find(&result.Commands).Error
	if err != nil {
		return result, err
	}
	var reversed []remoteModel.TaskLog
	if err := r.db().Where("task_id = ?", id).Order("sequence DESC").Limit(500).Find(&reversed).Error; err != nil {
		return result, err
	}
	result.Logs = make([]remoteModel.TaskLog, len(reversed))
	for index := range reversed {
		result.Logs[len(reversed)-1-index] = reversed[index]
	}
	return result, nil
}

func (r gormTaskRepository) ClaimNext(agentID snowflake.ID, now, staleBefore int64) (*remoteReq.ClaimedCommand, error) {
	var claimed remoteReq.ClaimedCommand
	err := r.db().Transaction(func(tx *gorm.DB) error {
		query := tx.Where("agent_id = ? AND (status = ? OR (status = ? AND dispatched_at < ?))",
			agentID, remoteModel.CommandStatusPending, remoteModel.CommandStatusDispatched, staleBefore).
			Order("created_at ASC")
		if err := query.First(&claimed.Command).Error; err != nil {
			return err
		}
		condition := tx.Model(&remoteModel.Command{}).Where("id = ?", claimed.Command.ID)
		if claimed.Command.Status == remoteModel.CommandStatusPending {
			condition = condition.Where("status = ?", remoteModel.CommandStatusPending)
		} else {
			condition = condition.Where("status = ? AND dispatched_at < ?", remoteModel.CommandStatusDispatched, staleBefore)
		}
		update := condition.Updates(map[string]any{
			"status": remoteModel.CommandStatusDispatched, "dispatched_at": now,
			"attempt": gorm.Expr("attempt + 1"), "updated_at": now,
		})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return errCommandClaimed
		}
		claimed.Command.Status = remoteModel.CommandStatusDispatched
		claimed.Command.DispatchedAt = now
		claimed.Command.Attempt++
		if err := tx.First(&claimed.Task, claimed.Command.TaskID).Error; err != nil {
			return err
		}
		if claimed.Task.WorkspaceID > 0 {
			if err := tx.First(&claimed.Workspace, claimed.Task.WorkspaceID).Error; err != nil {
				return err
			}
		}
		return tx.Model(&remoteModel.TaskLog{}).Where("task_id = ?", claimed.Task.ID).
			Select("COALESCE(MAX(sequence), 0) + 1").Scan(&claimed.NextLogSequence).Error
	})
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (r gormTaskRepository) AppendLogs(agentID, taskID snowflake.ID, logs []remoteModel.TaskLog) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&remoteModel.Task{}).Where("id = ? AND agent_id = ?", taskID, agentID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("task does not belong to agent")
		}
		if len(logs) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_id"}, {Name: "sequence"}},
			DoNothing: true,
		}).Create(&logs).Error
	})
}

func (r gormTaskRepository) LogsAfter(taskID snowflake.ID, after int64, limit int) ([]remoteModel.TaskLog, error) {
	var logs []remoteModel.TaskLog
	err := r.db().Where("task_id = ? AND sequence > ?", taskID, after).
		Order("sequence ASC").Limit(limit).Find(&logs).Error
	return logs, err
}

func (r gormTaskRepository) TaskStatus(taskID snowflake.ID) (string, error) {
	var task remoteModel.Task
	err := r.db().Select("status").First(&task, taskID).Error
	return task.Status, err
}

func (r gormTaskRepository) Acknowledge(agentID, commandID snowflake.ID, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var command remoteModel.Command
		if err := tx.Where("id = ? AND agent_id = ?", commandID, agentID).First(&command).Error; err != nil {
			return err
		}
		if command.Status == remoteModel.CommandStatusAcknowledged || command.Status == remoteModel.CommandStatusCompleted {
			return nil
		}
		if command.Status != remoteModel.CommandStatusDispatched {
			return errors.New("command is not dispatched")
		}
		if command.Type == remoteModel.CommandTypeExecuteTask {
			update := tx.Model(&remoteModel.Task{}).
				Where("id = ? AND agent_id = ? AND status = ?", command.TaskID, agentID, remoteModel.TaskStatusPending).
				Updates(map[string]any{"status": remoteModel.TaskStatusRunning, "started_at": now, "updated_at": now})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected == 0 {
				return errors.New("task is not pending")
			}
		}
		return tx.Model(&command).Updates(map[string]any{
			"status": remoteModel.CommandStatusAcknowledged, "acknowledged_at": now, "updated_at": now,
		}).Error
	})
}

func (r gormTaskRepository) Complete(agentID snowflake.ID, result remoteReq.TaskResultRequest, changedFiles string, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var task remoteModel.Task
		if err := tx.Where("id = ? AND agent_id = ?", result.TaskID, agentID).First(&task).Error; err != nil {
			return err
		}
		if task.Status != remoteModel.TaskStatusRunning && task.Status != remoteModel.TaskStatusCancelled {
			if task.Status == remoteModel.TaskStatusSuccess || task.Status == remoteModel.TaskStatusFailed {
				return nil
			}
			return errors.New("task is not running")
		}
		values := map[string]any{
			"result": result.Result, "error_message": result.ErrorMessage,
			"changed_files": changedFiles, "finished_at": now, "updated_at": now,
		}
		commandStatus := remoteModel.CommandStatusCompleted
		if task.Status != remoteModel.TaskStatusCancelled {
			if result.Success {
				values["status"] = remoteModel.TaskStatusSuccess
			} else {
				values["status"] = remoteModel.TaskStatusFailed
				commandStatus = remoteModel.CommandStatusFailed
			}
		}
		if err := tx.Model(&task).Updates(values).Error; err != nil {
			return err
		}
		if err := tx.Model(&remoteModel.Command{}).
			Where("task_id = ? AND type = ?", task.ID, remoteModel.CommandTypeExecuteTask).
			Updates(map[string]any{"status": commandStatus, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&remoteModel.Command{}).
			Where("task_id = ? AND type = ? AND status IN ?", task.ID, remoteModel.CommandTypeStopTask,
				[]string{remoteModel.CommandStatusPending, remoteModel.CommandStatusDispatched, remoteModel.CommandStatusAcknowledged}).
			Updates(map[string]any{"status": remoteModel.CommandStatusCompleted, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&remoteModel.Agent{}).
			Where("id = ? AND current_task_id = ?", agentID, task.ID).
			Updates(map[string]any{"status": remoteModel.AgentStatusOnline, "current_task_id": 0, "updated_at": now}).Error
	})
}

func (r gormTaskRepository) Cancel(taskID snowflake.ID, stopCommand remoteModel.Command, now int64) error {
	return r.db().Transaction(func(tx *gorm.DB) error {
		var task remoteModel.Task
		if err := tx.First(&task, taskID).Error; err != nil {
			return err
		}
		switch task.Status {
		case remoteModel.TaskStatusPending:
			if err := tx.Model(&task).Updates(map[string]any{
				"status": remoteModel.TaskStatusCancelled, "finished_at": now, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&remoteModel.Command{}).Where("task_id = ?", task.ID).
				Updates(map[string]any{"status": remoteModel.CommandStatusCancelled, "updated_at": now}).Error; err != nil {
				return err
			}
			return tx.Model(&remoteModel.Agent{}).Where("id = ? AND current_task_id = ?", task.AgentID, task.ID).
				Updates(map[string]any{"status": remoteModel.AgentStatusOnline, "current_task_id": 0, "updated_at": now}).Error
		case remoteModel.TaskStatusRunning:
			if err := tx.Model(&task).Updates(map[string]any{"status": remoteModel.TaskStatusCancelled, "updated_at": now}).Error; err != nil {
				return err
			}
			return tx.Create(&stopCommand).Error
		default:
			return errors.New("only pending or running tasks can be cancelled")
		}
	})
}
