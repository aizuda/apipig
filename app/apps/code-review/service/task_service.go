package service

import (
	"context"
	"errors"
	"strings"
	"time"

	reviewModel "apipig/app/apps/code-review/model"
	reviewReq "apipig/app/apps/code-review/model/request"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"
)

type TaskService struct{ runner *taskRunner }

func (s *TaskService) Start() {
	if s.runner != nil {
		s.runner.Start()
	}
}
func (s *TaskService) Shutdown(ctx context.Context) error {
	if s.runner == nil {
		return nil
	}
	return s.runner.Shutdown(ctx)
}

func (s *TaskService) Page(params *reviewReq.TaskPageParams) (response.PageResult, error) {
	query := global.DB.Model(&reviewModel.Task{})
	if params != nil {
		if params.ProjectID > 0 {
			query = query.Where("project_id = ?", params.ProjectID)
		}
		if params.EventType != "" {
			query = query.Where("event_type = ?", params.EventType)
		}
		if params.Status != "" {
			query = query.Where("status = ?", params.Status)
		}
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			like := "%" + keyword + "%"
			query = query.Where("repository_name LIKE ? OR title LIKE ? OR author LIKE ? OR head_sha LIKE ?", like, like, like, like)
		}
	}
	var tasks []reviewModel.Task
	return db.Page(query.Order("created_at DESC"), pageInfo(params), tasks)
}

func (s *TaskService) Get(id snowflake.ID) (reviewModel.Task, error) {
	var task reviewModel.Task
	err := global.DB.First(&task, id).Error
	return task, err
}

func (s *TaskService) Retry(params *reviewReq.RetryTaskParams) (bool, error) {
	if params == nil || params.ID == 0 {
		return false, errors.New("任务 ID 不能为空")
	}
	result := global.DB.Model(&reviewModel.Task{}).Where("id = ? AND status IN ?", params.ID, []string{reviewModel.TaskStatusFailed, reviewModel.TaskStatusIgnored}).Updates(map[string]any{"status": reviewModel.TaskStatusQueued, "error_message": "", "finished_at": 0})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, errors.New("仅失败或忽略的任务可以重试")
	}
	s.runner.Enqueue(params.ID)
	return true, nil
}

func (r *taskRunner) Start() {
	r.startOnce.Do(func() {
		queueSize := global.CONFIG.CodeReview.QueueSize
		if queueSize <= 0 {
			queueSize = 128
		}
		workerCount := global.CONFIG.CodeReview.WorkerCount
		if workerCount <= 0 {
			workerCount = 2
		}
		r.queue = make(chan snowflake.ID, queueSize)
		r.stop = make(chan struct{})
		r.done = make(chan struct{})
		var workersDone = make(chan struct{}, workerCount)
		for index := 0; index < workerCount; index++ {
			go func() {
				defer func() { workersDone <- struct{}{} }()
				for {
					select {
					case id := <-r.queue:
						if r.startExecutor != nil {
							r.startExecutor(id)
						}
					case <-r.stop:
						return
					}
				}
			}()
		}
		go func() {
			for index := 0; index < workerCount; index++ {
				<-workersDone
			}
			close(r.done)
		}()
		var tasks []reviewModel.Task
		if err := global.DB.Where("status IN ?", []string{reviewModel.TaskStatusQueued, reviewModel.TaskStatusRunning}).Find(&tasks).Error; err != nil {
			logReviewError("恢复代码评审任务失败", err)
			return
		}
		for _, task := range tasks {
			_ = global.DB.Model(&task).Update("status", reviewModel.TaskStatusQueued).Error
			select {
			case r.queue <- task.ID:
			case <-r.stop:
				return
			}
		}
	})
}

func (r *taskRunner) Enqueue(id snowflake.ID) {
	r.Start()
	select {
	case r.queue <- id:
	case <-r.stop:
	}
}

func (r *taskRunner) Shutdown(ctx context.Context) error {
	if r.stop == nil {
		return nil
	}
	r.stopOnce.Do(func() { close(r.stop) })
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return errors.New("代码评审任务停止超时")
	}
}
