package service

import (
	"encoding/json"
	"errors"
	"net/url"
	"path"
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
	maxTaskLogBatchSize    = 100
	maxTaskLogContentBytes = 32 * 1024
)

type TaskService struct {
	repository   taskRepository
	agentService *AgentService
	now          func() time.Time
	sleep        func(time.Duration)
}

func NewTaskService(agentService *AgentService) *TaskService {
	return &TaskService{repository: gormTaskRepository{}, agentService: agentService, now: time.Now, sleep: time.Sleep}
}

func (s *TaskService) Create(params *remoteReq.TaskCreateParams) (remoteModel.Task, error) {
	if params == nil || params.Ctx == nil {
		return remoteModel.Task{}, errors.New("task parameters are required")
	}
	request := params.Request
	request.Name = strings.TrimSpace(request.Name)
	request.RepositoryURL = strings.TrimSpace(request.RepositoryURL)
	request.WorkingDir = strings.TrimSpace(request.WorkingDir)
	request.Prompt = strings.TrimSpace(request.Prompt)
	if err := validateTaskCreate(request); err != nil {
		return remoteModel.Task{}, err
	}
	now := s.now().UnixMilli()
	base := db.NewModel(params.Ctx)
	base.CreatedAt = now
	task := remoteModel.Task{
		MODEL: base, Name: request.Name, AgentID: request.AgentID,
		RepositoryURL: request.RepositoryURL, WorkingDir: request.WorkingDir,
		Prompt: request.Prompt, Status: remoteModel.TaskStatusPending,
	}
	workspace := remoteModel.Workspace{
		MODEL:   coreAPI.MODEL{ID: db.GetId(), CreatedId: base.CreatedId, CreatedBy: base.CreatedBy, CreatedAt: now},
		AgentID: task.AgentID, Name: task.Name, Path: task.ID.String(), RepositoryURL: task.RepositoryURL, Status: 1,
	}
	task.WorkspaceID = workspace.ID
	command := remoteModel.Command{
		MODEL:  coreAPI.MODEL{ID: db.GetId(), CreatedId: base.CreatedId, CreatedBy: base.CreatedBy, CreatedAt: now},
		TaskID: task.ID, AgentID: task.AgentID, Type: remoteModel.CommandTypeExecuteTask,
		Status: remoteModel.CommandStatusPending,
	}
	if err := s.repository.Create(task, workspace, command); err != nil {
		return remoteModel.Task{}, err
	}
	return task, nil
}

func (s *TaskService) Page(params *remoteReq.TaskPageParams) (coreResp.PageResult, error) {
	if params != nil {
		params.Keyword = strings.TrimSpace(params.Keyword)
		params.Status = strings.ToUpper(strings.TrimSpace(params.Status))
		if params.Status != "" && !validTaskStatus(params.Status) {
			return coreResp.PageResult{}, errors.New("invalid task status")
		}
	}
	return s.repository.Page(params)
}

func (s *TaskService) Get(id snowflake.ID) (remoteResp.TaskDetail, error) {
	if id == 0 {
		return remoteResp.TaskDetail{}, errors.New("task ID is required")
	}
	return s.repository.Get(id)
}

func (s *TaskService) NextCommand(params *remoteReq.NextCommandParams) (*remoteResp.CommandDispatch, error) {
	if params == nil {
		return nil, errors.New("command parameters are required")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return nil, err
	}
	waitSeconds := params.WaitSeconds
	maxWait := global.CONFIG.RemoteAgent.CommandPollTimeoutSeconds
	if maxWait <= 0 {
		maxWait = 25
	}
	if waitSeconds <= 0 || waitSeconds > maxWait {
		waitSeconds = maxWait
	}
	leaseSeconds := global.CONFIG.RemoteAgent.DispatchLeaseSeconds
	if leaseSeconds <= 0 {
		leaseSeconds = 30
	}
	deadline := s.now().Add(time.Duration(waitSeconds) * time.Second)
	for {
		now := s.now()
		claimed, claimErr := s.repository.ClaimNext(agent.ID, now.UnixMilli(), now.Add(-time.Duration(leaseSeconds)*time.Second).UnixMilli())
		if claimErr == nil {
			return commandDispatch(claimed), nil
		}
		if !errors.Is(claimErr, gorm.ErrRecordNotFound) && !errors.Is(claimErr, errCommandClaimed) {
			return nil, claimErr
		}
		if !now.Before(deadline) {
			return nil, nil
		}
		s.sleep(300 * time.Millisecond)
	}
}

func (s *TaskService) Acknowledge(params *remoteReq.AcknowledgeCommandParams) (bool, error) {
	if params == nil || params.CommandID == 0 {
		return false, errors.New("command ID is required")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	err = s.repository.Acknowledge(agent.ID, params.CommandID, s.now().UnixMilli())
	return err == nil, err
}

func (s *TaskService) Complete(params *remoteReq.TaskResultParams) (bool, error) {
	if params == nil || params.Request.TaskID == 0 {
		return false, errors.New("task ID is required")
	}
	if len(params.Request.Result) > 1024*1024 || len(params.Request.ErrorMessage) > 64*1024 || len(params.Request.ChangedFiles) > 5000 {
		return false, errors.New("task result is too large")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	changedFiles, err := json.Marshal(params.Request.ChangedFiles)
	if err != nil {
		return false, err
	}
	err = s.repository.Complete(agent.ID, params.Request, string(changedFiles), s.now().UnixMilli())
	return err == nil, err
}

func (s *TaskService) AppendLogs(params *remoteReq.TaskLogUploadParams) (bool, error) {
	if params == nil || params.Request.TaskID == 0 {
		return false, errors.New("task ID is required")
	}
	if len(params.Request.Logs) == 0 || len(params.Request.Logs) > maxTaskLogBatchSize {
		return false, errors.New("log batch must contain between 1 and 100 entries")
	}
	agent, err := s.agentService.Authenticate(params.AgentToken)
	if err != nil {
		return false, err
	}
	now := s.now().UnixMilli()
	logs := make([]remoteModel.TaskLog, 0, len(params.Request.Logs))
	previousSequence := int64(0)
	for _, entry := range params.Request.Logs {
		if entry.Sequence <= 0 || entry.Sequence <= previousSequence {
			return false, errors.New("log sequences must be positive and strictly increasing")
		}
		if entry.Stream != "stdout" && entry.Stream != "stderr" {
			return false, errors.New("log stream must be stdout or stderr")
		}
		if entry.Content == "" || len(entry.Content) > maxTaskLogContentBytes {
			return false, errors.New("log content is empty or too large")
		}
		logs = append(logs, remoteModel.TaskLog{
			MODEL:  coreAPI.MODEL{ID: db.GetId(), CreatedBy: "remote-agent", CreatedAt: now},
			TaskID: params.Request.TaskID, AgentID: agent.ID, Sequence: entry.Sequence,
			Stream: entry.Stream, Content: entry.Content,
		})
		previousSequence = entry.Sequence
	}
	err = s.repository.AppendLogs(agent.ID, params.Request.TaskID, logs)
	return err == nil, err
}

func (s *TaskService) LogsAfter(taskID snowflake.ID, after int64) ([]remoteModel.TaskLog, error) {
	if taskID == 0 || after < 0 {
		return nil, errors.New("invalid log cursor")
	}
	return s.repository.LogsAfter(taskID, after, 200)
}

func (s *TaskService) Status(taskID snowflake.ID) (string, error) {
	return s.repository.TaskStatus(taskID)
}

func (s *TaskService) Cancel(params *remoteReq.CancelTaskParams) (bool, error) {
	if params == nil || params.Ctx == nil || params.ID == 0 {
		return false, errors.New("task ID is required")
	}
	detail, err := s.repository.Get(params.ID)
	if err != nil {
		return false, err
	}
	now := s.now().UnixMilli()
	base := db.NewModel(params.Ctx)
	base.CreatedAt = now
	stop := remoteModel.Command{
		MODEL: base, TaskID: detail.Task.ID, AgentID: detail.Task.AgentID,
		Type: remoteModel.CommandTypeStopTask, Status: remoteModel.CommandStatusPending,
	}
	err = s.repository.Cancel(params.ID, stop, now)
	return err == nil, err
}

func commandDispatch(claimed *remoteReq.ClaimedCommand) *remoteResp.CommandDispatch {
	if claimed == nil {
		return nil
	}
	return &remoteResp.CommandDispatch{
		CommandID: claimed.Command.ID, Type: claimed.Command.Type, TaskID: claimed.Task.ID,
		TaskName: claimed.Task.Name, RepositoryURL: claimed.Task.RepositoryURL,
		WorkingDir: claimed.Task.WorkingDir, Prompt: claimed.Task.Prompt,
		NextLogSequence: claimed.NextLogSequence,
	}
}

func validateTaskCreate(request remoteReq.TaskCreateRequest) error {
	if request.AgentID == 0 {
		return errors.New("agent ID is required")
	}
	if request.Name == "" || len(request.Name) > 200 {
		return errors.New("task name is required and cannot exceed 200 characters")
	}
	if request.Prompt == "" || len(request.Prompt) > 128*1024 {
		return errors.New("prompt is required and cannot exceed 128 KiB")
	}
	if len(request.RepositoryURL) > 1000 {
		return errors.New("repository URL cannot exceed 1000 characters")
	}
	repositoryURL, err := url.ParseRequestURI(request.RepositoryURL)
	if err != nil || repositoryURL.Scheme != "https" || repositoryURL.Host == "" || repositoryURL.User != nil {
		return errors.New("repository URL must be an HTTPS URL without embedded credentials")
	}
	if request.WorkingDir != "" {
		cleaned := path.Clean(strings.ReplaceAll(request.WorkingDir, "\\", "/"))
		if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") || cleaned != strings.ReplaceAll(request.WorkingDir, "\\", "/") {
			return errors.New("working directory must be a clean relative path inside the repository")
		}
	}
	return nil
}

func validTaskStatus(status string) bool {
	switch status {
	case remoteModel.TaskStatusPending, remoteModel.TaskStatusRunning, remoteModel.TaskStatusSuccess, remoteModel.TaskStatusFailed, remoteModel.TaskStatusCancelled:
		return true
	default:
		return false
	}
}
