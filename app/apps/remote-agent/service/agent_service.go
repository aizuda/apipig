package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

var errAgentDisabled = errors.New("agent is disabled")

type AgentService struct {
	repository        agentRepository
	registrationToken func() string
	now               func() time.Time
}

func NewAgentService() *AgentService {
	return &AgentService{
		repository:        gormAgentRepository{},
		registrationToken: func() string { return global.CONFIG.RemoteAgent.RegistrationToken },
		now:               time.Now,
	}
}

func (s *AgentService) Register(params *remoteReq.RegisterParams) (remoteResp.RegisterResult, error) {
	if params == nil {
		return remoteResp.RegisterResult{}, errors.New("registration parameters are required")
	}
	request := params.Request
	request.AgentKey = strings.TrimSpace(request.AgentKey)
	request.Name = strings.TrimSpace(request.Name)
	request.Hostname = strings.TrimSpace(request.Hostname)
	if err := validateRegistration(request); err != nil {
		return remoteResp.RegisterResult{}, err
	}
	if !secureTokenEqual(params.BootstrapToken, s.registrationToken()) {
		return remoteResp.RegisterResult{}, errors.New("invalid registration token")
	}

	rawToken, err := generateAgentToken()
	if err != nil {
		return remoteResp.RegisterResult{}, err
	}
	now := s.now().UnixMilli()
	agent, err := s.repository.FindByKey(request.AgentKey)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return remoteResp.RegisterResult{}, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		agent = remoteModel.Agent{
			MODEL:    coreAPI.MODEL{ID: db.GetId(), CreatedBy: "remote-agent", CreatedAt: now},
			AgentKey: request.AgentKey,
		}
	}
	agent.Name = request.Name
	agent.TokenHash = hashAgentToken(rawToken)
	agent.IPAddress = strings.TrimSpace(params.IPAddress)
	agent.Hostname = request.Hostname
	agent.OperatingSystem = strings.TrimSpace(request.OperatingSystem)
	agent.Architecture = strings.TrimSpace(request.Architecture)
	agent.CPUInfo = strings.TrimSpace(request.CPUInfo)
	agent.MemoryTotal = request.MemoryTotal
	agent.CodexVersion = strings.TrimSpace(request.CodexVersion)
	agent.AgentVersion = strings.TrimSpace(request.AgentVersion)
	agent.Status = remoteModel.AgentStatusOnline
	if agent.CurrentTaskID > 0 {
		agent.Status = remoteModel.AgentStatusBusy
	}
	agent.LastSeenAt = now
	agent.UpdatedAt = now
	existingTaskID := agent.CurrentTaskID
	if err == nil {
		err = s.repository.UpdateRegistration(&agent)
	} else {
		err = s.repository.Create(&agent)
	}
	if err != nil {
		return remoteResp.RegisterResult{}, err
	}
	if existingTaskID > 0 && request.CurrentTaskID == 0 {
		if err := s.repository.RecoverTask(agent.ID, existingTaskID, now); err != nil {
			return remoteResp.RegisterResult{}, err
		}
	}
	return remoteResp.RegisterResult{
		AgentID: agent.ID, AgentToken: rawToken, Status: agent.Status,
		HeartbeatIntervalSeconds: s.heartbeatIntervalSeconds(),
	}, nil
}

func (s *AgentService) Heartbeat(params *remoteReq.HeartbeatParams) (remoteResp.HeartbeatResult, error) {
	if params == nil || strings.TrimSpace(params.AgentToken) == "" {
		return remoteResp.HeartbeatResult{}, errors.New("agent token is required")
	}
	if params.Request.CPUUsage < 0 || params.Request.CPUUsage > 100 {
		return remoteResp.HeartbeatResult{}, errors.New("cpuUsage must be between 0 and 100")
	}
	if params.Request.MemoryUsed < 0 {
		return remoteResp.HeartbeatResult{}, errors.New("memoryUsed cannot be negative")
	}
	agent, err := s.Authenticate(params.AgentToken)
	if err != nil {
		return remoteResp.HeartbeatResult{}, err
	}
	now := s.now().UnixMilli()
	status := remoteModel.AgentStatusOnline
	if params.Request.Busy || agent.CurrentTaskID > 0 {
		status = remoteModel.AgentStatusBusy
	}
	agent.IPAddress = strings.TrimSpace(params.IPAddress)
	agent.CPUUsage = params.Request.CPUUsage
	agent.MemoryUsed = params.Request.MemoryUsed
	if version := strings.TrimSpace(params.Request.CodexVersion); version != "" {
		agent.CodexVersion = version
	}
	agent.Status = status
	agent.LastSeenAt = now
	agent.UpdatedAt = now
	heartbeat := remoteModel.Heartbeat{
		MODEL:   coreAPI.MODEL{ID: db.GetId(), CreatedBy: "remote-agent", CreatedAt: now},
		AgentID: agent.ID, Status: status, CPUUsage: agent.CPUUsage,
		MemoryUsed: agent.MemoryUsed, OccurredAt: now,
	}
	if err := s.repository.RecordHeartbeat(agent, heartbeat); err != nil {
		return remoteResp.HeartbeatResult{}, err
	}
	return remoteResp.HeartbeatResult{AgentID: agent.ID, Status: status, ServerTime: now}, nil
}

func (s *AgentService) Authenticate(rawToken string) (remoteModel.Agent, error) {
	if strings.TrimSpace(rawToken) == "" {
		return remoteModel.Agent{}, errors.New("agent token is required")
	}
	agent, err := s.repository.FindByTokenHash(hashAgentToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return remoteModel.Agent{}, errors.New("invalid agent token")
		}
		return remoteModel.Agent{}, err
	}
	if agent.Status == remoteModel.AgentStatusDisabled {
		return remoteModel.Agent{}, errAgentDisabled
	}
	return agent, nil
}

func (s *AgentService) Page(params *remoteReq.AgentPageParams) (coreResp.PageResult, error) {
	if err := s.MarkOffline(); err != nil {
		return coreResp.PageResult{}, err
	}
	if params != nil {
		params.Keyword = strings.TrimSpace(params.Keyword)
		params.Status = strings.ToUpper(strings.TrimSpace(params.Status))
		if params.Status != "" && !validAgentStatus(params.Status) {
			return coreResp.PageResult{}, errors.New("invalid agent status")
		}
	}
	return s.repository.Page(params)
}

func (s *AgentService) Get(id snowflake.ID) (remoteResp.AgentDetail, error) {
	if id == 0 {
		return remoteResp.AgentDetail{}, errors.New("agent ID is required")
	}
	if err := s.MarkOffline(); err != nil {
		return remoteResp.AgentDetail{}, err
	}
	agent, err := s.repository.Get(id)
	if err != nil {
		return remoteResp.AgentDetail{}, err
	}
	heartbeats, err := s.repository.RecentHeartbeats(id, 100)
	if err != nil {
		return remoteResp.AgentDetail{}, err
	}
	tasks, err := s.repository.RecentTasks(id, 20)
	if err != nil {
		return remoteResp.AgentDetail{}, err
	}
	return remoteResp.AgentDetail{Agent: agent, Heartbeats: heartbeats, Tasks: tasks}, nil
}

func (s *AgentService) MarkOffline() error {
	timeout := global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds
	if timeout <= 0 {
		timeout = 90
	}
	now := s.now()
	return s.repository.MarkOffline(now.Add(-time.Duration(timeout)*time.Second).UnixMilli(), now.UnixMilli())
}

func (s *AgentService) heartbeatIntervalSeconds() int {
	timeout := global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds
	if timeout <= 0 {
		timeout = 90
	}
	interval := timeout / 3
	if interval < 10 {
		return 10
	}
	if interval > 30 {
		return 30
	}
	return interval
}

func validateRegistration(request remoteReq.RegisterRequest) error {
	if request.AgentKey == "" || len(request.AgentKey) > 100 {
		return errors.New("agentKey is required and cannot exceed 100 characters")
	}
	if request.Name == "" || len(request.Name) > 100 {
		return errors.New("name is required and cannot exceed 100 characters")
	}
	if request.Hostname == "" || len(request.Hostname) > 255 {
		return errors.New("hostname is required and cannot exceed 255 characters")
	}
	if request.MemoryTotal < 0 {
		return errors.New("memoryTotal cannot be negative")
	}
	return nil
}

func validAgentStatus(status string) bool {
	switch status {
	case remoteModel.AgentStatusOnline, remoteModel.AgentStatusOffline, remoteModel.AgentStatusBusy, remoteModel.AgentStatusDisabled:
		return true
	default:
		return false
	}
}

func generateAgentToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "ra_" + base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashAgentToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func secureTokenEqual(actual, expected string) bool {
	actualHash := sha256.Sum256([]byte(strings.TrimSpace(actual)))
	expectedHash := sha256.Sum256([]byte(strings.TrimSpace(expected)))
	return strings.TrimSpace(expected) != "" && subtle.ConstantTimeCompare(actualHash[:], expectedHash[:]) == 1
}
