package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	remoteResp "apipig/app/apps/remote-agent/model/response"
	coreAPI "apipig/core/api"
	coreResp "apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

var (
	// errAgentDisabled 用于并发更新时统一表示 Agent 已被管理员禁用。
	errAgentDisabled = errors.New("Agent 已禁用")
	// defaultCodexArgs 通过标准输入向 Codex CLI 传递会话提示词。
	defaultCodexArgs = []string{
		"--ask-for-approval", "never", "exec", "--json", "--sandbox", "workspace-write",
		"-c", "sandbox_workspace_write.network_access=true",
		"--skip-git-repo-check", "-",
	}
	defaultClaudeArgs = []string{"-p", "--permission-mode", "acceptEdits"}
	// agentKeyPattern 限制 Agent Key 仅包含适合作为稳定标识的安全字符。
	agentKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// AgentService 负责 Agent 配置、注册鉴权、心跳和在线状态管理。
type AgentService struct {
	repository      agentRepository
	now             func() time.Time
	livenessTimeout func() time.Duration
	events          *agentEventBroker
	livenessMu      sync.Mutex
	livenessTimers  map[snowflake.ID]*agentLivenessTimer
}

type agentLivenessTimer struct {
	timer      *time.Timer
	lastSeenAt int64
}

// NewAgentService 创建使用 GORM 仓储和系统时间的 Agent 服务。
func NewAgentService() *AgentService {
	return &AgentService{
		repository: gormAgentRepository{}, now: time.Now, livenessTimeout: configuredAgentLivenessTimeout,
		events: newAgentEventBroker(), livenessTimers: make(map[snowflake.ID]*agentLivenessTimer),
	}
}

func configuredAgentLivenessTimeout() time.Duration {
	timeout := global.CONFIG.RemoteAgent.HeartbeatTimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}
	return time.Duration(timeout) * time.Second
}

// StartLivenessTracking restores one deadline per connected Agent after a controller restart.
func (s *AgentService) StartLivenessTracking() error {
	agents, err := s.repository.ActiveAgents()
	if err != nil {
		return err
	}
	for _, agent := range agents {
		s.scheduleLivenessTimeout(agent.ID, agent.LastSeenAt)
	}
	return nil
}

func (s *AgentService) StopLivenessTracking() {
	s.livenessMu.Lock()
	defer s.livenessMu.Unlock()
	for id, entry := range s.livenessTimers {
		entry.timer.Stop()
		delete(s.livenessTimers, id)
	}
}

func (s *AgentService) scheduleLivenessTimeout(id snowflake.ID, lastSeenAt int64) {
	if id == 0 {
		return
	}
	delay := s.livenessTimeout() - s.now().Sub(time.UnixMilli(lastSeenAt))
	if delay < 0 {
		delay = 0
	}
	entry := &agentLivenessTimer{lastSeenAt: lastSeenAt}
	s.livenessMu.Lock()
	if previous := s.livenessTimers[id]; previous != nil && previous.lastSeenAt > lastSeenAt {
		s.livenessMu.Unlock()
		return
	}
	entry.timer = time.AfterFunc(delay, func() { s.expireAgent(id, lastSeenAt, entry) })
	if previous := s.livenessTimers[id]; previous != nil {
		previous.timer.Stop()
	}
	s.livenessTimers[id] = entry
	s.livenessMu.Unlock()
}

func (s *AgentService) cancelLivenessTimeout(id snowflake.ID) {
	s.livenessMu.Lock()
	if entry := s.livenessTimers[id]; entry != nil {
		entry.timer.Stop()
		delete(s.livenessTimers, id)
	}
	s.livenessMu.Unlock()
}

func (s *AgentService) expireAgent(id snowflake.ID, lastSeenAt int64, entry *agentLivenessTimer) {
	s.livenessMu.Lock()
	if s.livenessTimers[id] != entry {
		s.livenessMu.Unlock()
		return
	}
	delete(s.livenessTimers, id)
	s.livenessMu.Unlock()

	agent, previousStatus, changed, err := s.repository.MarkAgentOffline(id, lastSeenAt, s.now().UnixMilli())
	if err != nil {
		if global.LOG != nil {
			global.LOG.Error("mark remote agent offline after heartbeat timeout", zap.String("agentId", id.String()), zap.Error(err))
		}
		return
	}
	if changed {
		s.publishAgent(agent, previousStatus)
	}
}

func (s *AgentService) AgentEvents() (<-chan remoteResp.AgentEvent, func()) {
	return s.events.subscribe()
}

func (s *AgentService) publishAgent(agent remoteModel.Agent, previousStatus string) {
	s.publishAgentEvent("upsert", agent, previousStatus)
}

func (s *AgentService) publishAgentEvent(action string, agent remoteModel.Agent, previousStatus string) {
	s.events.publish(remoteResp.AgentEvent{Action: action, Agent: agent, PreviousStatus: previousStatus})
}

func (s *AgentService) publishAgentByID(id snowflake.ID, previousStatus string) {
	agent, err := s.repository.Get(id)
	if err != nil {
		if global.LOG != nil {
			global.LOG.Warn("failed to load remote agent for status event", zap.String("agentId", id.String()), zap.Error(err))
		}
		return
	}
	s.publishAgent(agent, previousStatus)
}

// Create 创建离线 Agent 配置，并生成控制端仅在本次响应中返回的接入注册令牌明文。
func (s *AgentService) Create(params *remoteReq.AgentCreateParams) (remoteResp.AgentCredential, error) {
	if params == nil {
		return remoteResp.AgentCredential{}, errors.New("Agent 配置不能为空")
	}
	agent, err := normalizeAgent(params.Request)
	if err != nil {
		return remoteResp.AgentCredential{}, err
	}
	if _, err := s.repository.FindByKey(agent.AgentKey); err == nil {
		return remoteResp.AgentCredential{}, errors.New("Agent Key 已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return remoteResp.AgentCredential{}, err
	}
	token, err := generateRegistrationToken()
	if err != nil {
		return remoteResp.AgentCredential{}, err
	}
	now := s.now().UnixMilli()
	agent.MODEL = coreAPI.MODEL{ID: db.GetId(), CreatedBy: "admin", CreatedAt: now, UpdatedAt: now}
	// 数据库只保存摘要，避免数据库泄露后可直接使用注册令牌接入 Agent。
	agent.RegistrationTokenHash = hashAgentToken(token)
	agent.Status = remoteModel.AgentStatusOffline
	if err := s.repository.Create(&agent); err != nil {
		return remoteResp.AgentCredential{}, err
	}
	s.publishAgent(agent, "")
	return credentialResult(agent, token, params.ControllerURL)
}

// Update 更新 Agent 的可配置项，已注册节点不允许变更唯一 Agent Key。
func (s *AgentService) Update(request *remoteReq.AgentSaveRequest) (remoteModel.Agent, error) {
	if request == nil || request.ID == 0 {
		return remoteModel.Agent{}, errors.New("Agent ID 不能为空")
	}
	existing, err := s.repository.Get(request.ID)
	if err != nil {
		return remoteModel.Agent{}, err
	}
	agent, err := normalizeAgent(*request)
	if err != nil {
		return remoteModel.Agent{}, err
	}
	if existing.Hostname != "" && agent.AgentKey != existing.AgentKey {
		return remoteModel.Agent{}, errors.New("注册后不能修改 Agent Key")
	}
	agent.MODEL = existing.MODEL
	agent.UpdatedAt = s.now().UnixMilli()
	if err := s.repository.UpdateConfiguration(&agent); err != nil {
		return remoteModel.Agent{}, err
	}
	updated, err := s.repository.Get(agent.ID)
	if err == nil {
		s.publishAgent(updated, existing.Status)
	}
	return updated, err
}

// RotateToken 重置注册令牌，并使旧注册令牌和当前运行令牌同时失效。
func (s *AgentService) RotateToken(params *remoteReq.AgentRotateTokenParams) (remoteResp.AgentCredential, error) {
	if params == nil || params.ID == 0 {
		return remoteResp.AgentCredential{}, errors.New("Agent ID 不能为空")
	}
	agent, err := s.repository.Get(params.ID)
	if err != nil {
		return remoteResp.AgentCredential{}, err
	}
	if agent.Status != remoteModel.AgentStatusOffline && agent.Status != remoteModel.AgentStatusDisabled {
		return remoteResp.AgentCredential{}, errors.New("只有离线或已禁用的 Agent 才能重置注册令牌")
	}
	token, err := generateRegistrationToken()
	if err != nil {
		return remoteResp.AgentCredential{}, err
	}
	agent.RegistrationTokenHash = hashAgentToken(token)
	agent.UpdatedAt = s.now().UnixMilli()
	if err := s.repository.UpdateRegistrationToken(agent.ID, agent.RegistrationTokenHash, agent.UpdatedAt); err != nil {
		return remoteResp.AgentCredential{}, err
	}
	return credentialResult(agent, token, params.ControllerURL)
}

// SetStatus 启用或禁用 Agent；正在响应的 Agent 不允许被禁用。
func (s *AgentService) SetStatus(request *remoteReq.AgentStatusRequest) (bool, error) {
	if request == nil || request.ID == 0 {
		return false, errors.New("Agent ID 不能为空")
	}
	existing, err := s.repository.Get(request.ID)
	if err != nil {
		return false, err
	}
	status := remoteModel.AgentStatusDisabled
	if request.Enabled {
		status = remoteModel.AgentStatusOffline
	} else {
		if existing.CurrentMessageID != 0 {
			return false, errors.New("Agent 正在响应时不能禁用")
		}
	}
	if err := s.repository.SetStatus(request.ID, status, s.now().UnixMilli()); err != nil {
		return false, err
	}
	if status == remoteModel.AgentStatusOffline || status == remoteModel.AgentStatusDisabled {
		s.cancelLivenessTimeout(request.ID)
	}
	updated, err := s.repository.Get(request.ID)
	if err == nil {
		s.publishAgent(updated, existing.Status)
	}
	return true, nil
}

// Delete 删除 Agent 及其心跳、会话、消息和命令等全部关联数据。
func (s *AgentService) Delete(request *remoteReq.AgentDeleteRequest) (bool, error) {
	if request == nil || request.ID == 0 {
		return false, errors.New("Agent ID 不能为空")
	}
	agent, err := s.repository.Get(request.ID)
	if err != nil {
		return false, err
	}
	if agent.CurrentMessageID != 0 {
		return false, errors.New("Agent 正在响应时不能删除")
	}
	if err := s.repository.Delete(agent.ID); err != nil {
		return false, err
	}
	s.cancelLivenessTimeout(agent.ID)
	s.publishAgentEvent("delete", agent, agent.Status)
	return true, nil
}

// Register 校验接入注册令牌，登记客户端信息并签发新的运行令牌。
func (s *AgentService) Register(params *remoteReq.RegisterParams) (remoteResp.RegisterResult, error) {
	if params == nil {
		return remoteResp.RegisterResult{}, errors.New("注册参数不能为空")
	}
	request := params.Request
	request.AgentKey = strings.TrimSpace(request.AgentKey)
	request.Hostname = strings.TrimSpace(request.Hostname)
	if err := validateRegistration(request); err != nil {
		return remoteResp.RegisterResult{}, err
	}
	if strings.TrimSpace(params.BootstrapToken) == "" {
		return remoteResp.RegisterResult{}, errors.New("注册令牌不能为空")
	}
	agent, err := s.repository.FindByRegistrationTokenHash(hashAgentToken(params.BootstrapToken))
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return remoteResp.RegisterResult{}, err
		}
		if configured, keyErr := s.repository.FindByKey(request.AgentKey); keyErr == nil {
			if configured.Status == remoteModel.AgentStatusDisabled {
				return remoteResp.RegisterResult{}, fmt.Errorf("远程 Agent %q 已禁用", request.AgentKey)
			}
			return remoteResp.RegisterResult{}, fmt.Errorf("注册令牌与 Agent %q 不匹配，请重置令牌并替换配置文件", request.AgentKey)
		}
		return remoteResp.RegisterResult{}, fmt.Errorf("控制端尚未添加远程 Agent %q", request.AgentKey)
	}
	if agent.Status == remoteModel.AgentStatusDisabled {
		return remoteResp.RegisterResult{}, fmt.Errorf("远程 Agent %q 已禁用", agent.AgentKey)
	}
	if agent.AgentKey != request.AgentKey {
		return remoteResp.RegisterResult{}, fmt.Errorf("注册令牌属于 Agent %q", agent.AgentKey)
	}
	now := s.now().UnixMilli()
	// 客户端重启后未报告正在执行的消息时，将服务端遗留的执行状态恢复为可重试。
	if request.CurrentMessageID == 0 && agent.CurrentMessageID != 0 {
		if err := s.repository.RecoverInterruptedTurn(agent.ID, agent.CurrentMessageID, now); err != nil {
			return remoteResp.RegisterResult{}, fmt.Errorf("恢复中断的会话轮次失败：%w", err)
		}
		agent.CurrentMessageID = 0
	} else if request.CurrentMessageID != 0 {
		// 客户端仍在执行时，必须与服务端记录一致，防止错误关联其他会话消息。
		if agent.CurrentMessageID != 0 && agent.CurrentMessageID != request.CurrentMessageID {
			return remoteResp.RegisterResult{}, errors.New("Agent 报告的当前消息不一致")
		}
		agent.CurrentMessageID = request.CurrentMessageID
	}
	previousStatus := agent.Status
	rawToken, err := generateAgentToken()
	if err != nil {
		return remoteResp.RegisterResult{}, err
	}
	// 每次注册都轮换运行令牌，旧客户端持有的令牌会立即失效。
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
	if agent.CurrentMessageID > 0 {
		agent.Status = remoteModel.AgentStatusBusy
	}
	agent.LastSeenAt, agent.UpdatedAt = now, now
	if err := s.repository.UpdateRegistration(&agent); err != nil {
		return remoteResp.RegisterResult{}, err
	}
	s.scheduleLivenessTimeout(agent.ID, agent.LastSeenAt)
	s.publishAgent(agent, previousStatus)
	return remoteResp.RegisterResult{
		AgentID: agent.ID, AgentToken: rawToken, Status: agent.Status,
		HeartbeatIntervalSeconds: s.heartbeatIntervalSeconds(),
	}, nil
}

// Heartbeat 更新 Agent 的资源快照和在线状态，并追加一条心跳历史记录。
func (s *AgentService) Heartbeat(params *remoteReq.HeartbeatParams) (remoteResp.HeartbeatResult, error) {
	if params == nil || strings.TrimSpace(params.AgentToken) == "" {
		return remoteResp.HeartbeatResult{}, errors.New("Agent 令牌不能为空")
	}
	if params.Request.CPUUsage < 0 || params.Request.CPUUsage > 100 || params.Request.MemoryUsed < 0 {
		return remoteResp.HeartbeatResult{}, errors.New("资源使用数据无效")
	}
	agent, err := s.Authenticate(params.AgentToken)
	if err != nil {
		return remoteResp.HeartbeatResult{}, err
	}
	now := s.now().UnixMilli()
	status := remoteModel.AgentStatusOnline
	previousStatus := agent.Status
	if params.Request.Busy || agent.CurrentMessageID > 0 {
		status = remoteModel.AgentStatusBusy
	}
	agent.IPAddress = strings.TrimSpace(params.IPAddress)
	agent.CPUUsage, agent.MemoryUsed = params.Request.CPUUsage, params.Request.MemoryUsed
	if version := strings.TrimSpace(params.Request.CodexVersion); version != "" {
		agent.CodexVersion = version
	}
	agent.Status, agent.LastSeenAt, agent.UpdatedAt = status, now, now
	heartbeat := remoteModel.Heartbeat{
		MODEL:   coreAPI.MODEL{ID: db.GetId(), CreatedBy: "remote-agent", CreatedAt: now},
		AgentID: agent.ID, Status: status, CPUUsage: agent.CPUUsage, MemoryUsed: agent.MemoryUsed, OccurredAt: now,
	}
	if err := s.repository.RecordHeartbeat(agent, heartbeat); err != nil {
		return remoteResp.HeartbeatResult{}, err
	}
	s.scheduleLivenessTimeout(agent.ID, agent.LastSeenAt)
	s.publishAgent(agent, previousStatus)
	return remoteResp.HeartbeatResult{AgentID: agent.ID, Status: status, ServerTime: now}, nil
}

// Authenticate 使用运行令牌摘要查找 Agent，并拒绝已禁用节点。
func (s *AgentService) Authenticate(rawToken string) (remoteModel.Agent, error) {
	if strings.TrimSpace(rawToken) == "" {
		return remoteModel.Agent{}, errors.New("Agent 令牌不能为空")
	}
	agent, err := s.repository.FindByTokenHash(hashAgentToken(rawToken))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return remoteModel.Agent{}, errors.New("Agent 令牌无效")
		}
		return remoteModel.Agent{}, err
	}
	if agent.Status == remoteModel.AgentStatusDisabled {
		return remoteModel.Agent{}, errAgentDisabled
	}
	return agent, nil
}

// Page 查询 Agent 分页列表。在线状态由 Agent 生命周期事件维护，无需查询时扫描。
func (s *AgentService) Page(params *remoteReq.AgentPageParams) (coreResp.PageResult, error) {
	if params != nil {
		params.Keyword = strings.TrimSpace(params.Keyword)
		params.Status = strings.ToUpper(strings.TrimSpace(params.Status))
		if params.Status != "" && !validAgentStatus(params.Status) {
			return coreResp.PageResult{}, errors.New("Agent 状态无效")
		}
	}
	return s.repository.Page(params)
}

// Get 返回 Agent 当前快照、最近心跳和最近会话。
func (s *AgentService) Get(id snowflake.ID) (remoteResp.AgentDetail, error) {
	if id == 0 {
		return remoteResp.AgentDetail{}, errors.New("Agent ID 不能为空")
	}
	agent, err := s.repository.Get(id)
	if err != nil {
		return remoteResp.AgentDetail{}, err
	}
	heartbeats, err := s.repository.RecentHeartbeats(id, 100)
	if err != nil {
		return remoteResp.AgentDetail{}, err
	}
	conversations, err := s.repository.RecentConversations(id, 50)
	if err != nil {
		return remoteResp.AgentDetail{}, err
	}
	return remoteResp.AgentDetail{Agent: agent, Heartbeats: heartbeats, Conversations: conversations}, nil
}

// Status returns the latest liveness snapshot without loading heartbeat or conversation history.
func (s *AgentService) Status(id snowflake.ID) (remoteModel.Agent, error) {
	if id == 0 {
		return remoteModel.Agent{}, errors.New("Agent ID 不能为空")
	}
	return s.repository.Get(id)
}

// Disconnect immediately marks a gracefully stopped client as offline.
func (s *AgentService) Disconnect(rawToken string) (bool, error) {
	agent, err := s.Authenticate(rawToken)
	if err != nil {
		return false, err
	}
	previousStatus := agent.Status
	if err := s.repository.Disconnect(agent.ID, s.now().UnixMilli()); err != nil {
		return false, err
	}
	s.cancelLivenessTimeout(agent.ID)
	agent.Status = remoteModel.AgentStatusOffline
	agent.TokenHash = ""
	s.publishAgent(agent, previousStatus)
	return true, nil
}

// heartbeatIntervalSeconds 根据服务端离线阈值计算客户端心跳间隔，并限制在 10 到 30 秒。
func (s *AgentService) heartbeatIntervalSeconds() int {
	timeout := int(s.livenessTimeout() / time.Second)
	interval := timeout / 3
	if interval < 10 {
		return 10
	}
	if interval > 30 {
		return 30
	}
	return interval
}

// normalizeAgent 清理并校验管理端配置，同时补齐客户端运行所需默认值。
func normalizeAgent(request remoteReq.AgentSaveRequest) (remoteModel.Agent, error) {
	agent := remoteModel.Agent{
		AgentKey: strings.TrimSpace(request.AgentKey), Name: strings.TrimSpace(request.Name),
		WorkspaceRoot: strings.TrimSpace(request.WorkspaceRoot), CodexCommand: strings.TrimSpace(request.CodexCommand),
		ClaudeCommand:   strings.TrimSpace(request.ClaudeCommand),
		PollWaitSeconds: request.PollWaitSeconds, RequestTimeoutSeconds: request.RequestTimeoutSeconds,
		LogFile: strings.TrimSpace(request.LogFile),
	}
	if err := validateAgentKey(agent.AgentKey); err != nil {
		return remoteModel.Agent{}, err
	}
	if agent.Name == "" || len(agent.Name) > 100 {
		return remoteModel.Agent{}, errors.New("名称不能为空且不能超过 100 个字符")
	}
	if agent.WorkspaceRoot == "" {
		agent.WorkspaceRoot = "."
	}
	if agent.CodexCommand == "" {
		agent.CodexCommand = "codex"
	}
	for _, arg := range request.CodexArgs {
		if trimmed := strings.TrimSpace(arg); trimmed != "" {
			agent.CodexArgs = append(agent.CodexArgs, trimmed)
		}
	}
	if len(agent.CodexArgs) == 0 {
		agent.CodexArgs = append([]string(nil), defaultCodexArgs...)
	}
	if agent.ClaudeCommand == "" {
		agent.ClaudeCommand = "claude"
	}
	for _, arg := range request.ClaudeArgs {
		if trimmed := strings.TrimSpace(arg); trimmed != "" {
			agent.ClaudeArgs = append(agent.ClaudeArgs, trimmed)
		}
	}
	if len(agent.ClaudeArgs) == 0 {
		agent.ClaudeArgs = append([]string(nil), defaultClaudeArgs...)
	}
	if agent.PollWaitSeconds <= 0 || agent.PollWaitSeconds > 25 {
		agent.PollWaitSeconds = 25
	}
	if agent.RequestTimeoutSeconds <= agent.PollWaitSeconds {
		agent.RequestTimeoutSeconds = agent.PollWaitSeconds + 15
	}
	if agent.LogFile == "" {
		agent.LogFile = "./logs/remote-agent.log"
	}
	return agent, nil
}

// credentialResult 生成客户端配置文件；控制端仅在创建或重置响应中返回接入注册令牌明文。
func credentialResult(agent remoteModel.Agent, token, controllerURL string) (remoteResp.AgentCredential, error) {
	controllerURL = strings.TrimRight(strings.TrimSpace(controllerURL), "/")
	parsed, err := url.ParseRequestURI(controllerURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return remoteResp.AgentCredential{}, errors.New("控制端 URL 必须是 HTTP 或 HTTPS 地址")
	}
	config := struct {
		ControllerURL         string   `yaml:"controller-url"`
		RegistrationToken     string   `yaml:"registration-token"`
		AgentKey              string   `yaml:"agent-key"`
		Name                  string   `yaml:"name"`
		WorkspaceRoot         string   `yaml:"workspace-root"`
		CodexCommand          string   `yaml:"codex-command"`
		CodexArgs             []string `yaml:"codex-args"`
		ClaudeCommand         string   `yaml:"claude-command"`
		ClaudeArgs            []string `yaml:"claude-args"`
		PollWaitSeconds       int      `yaml:"poll-wait-seconds"`
		RequestTimeoutSeconds int      `yaml:"request-timeout-seconds"`
		LogFile               string   `yaml:"log-file"`
	}{controllerURL, token, agent.AgentKey, agent.Name, agent.WorkspaceRoot, agent.CodexCommand, agent.CodexArgs, agent.ClaudeCommand, agent.ClaudeArgs, agent.PollWaitSeconds, agent.RequestTimeoutSeconds, agent.LogFile}
	content, err := yaml.Marshal(config)
	if err != nil {
		return remoteResp.AgentCredential{}, err
	}
	return remoteResp.AgentCredential{Agent: agent, RegistrationToken: token, ConfigYAML: string(content)}, nil
}

// validateRegistration 校验客户端首次接入时上报的身份和硬件基础信息。
func validateRegistration(request remoteReq.RegisterRequest) error {
	if err := validateAgentKey(request.AgentKey); err != nil {
		return err
	}
	if request.Hostname == "" || len(request.Hostname) > 255 {
		return errors.New("主机名不能为空且不能超过 255 个字符")
	}
	if request.MemoryTotal < 0 {
		return errors.New("总内存不能为负数")
	}
	return nil
}

// validateAgentKey 保证 Agent Key 可安全用于配置标识和日志上下文。
func validateAgentKey(value string) error {
	if value == "" || len(value) > 100 {
		return errors.New("Agent Key 不能为空且不能超过 100 个字符")
	}
	if !agentKeyPattern.MatchString(value) {
		return errors.New("Agent Key 只能包含字母、数字、连字符和下划线")
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

// generateAgentToken 使用密码学安全随机数生成运行令牌。
func generateAgentToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "ra_" + base64.RawURLEncoding.EncodeToString(buffer), nil
}

// generateRegistrationToken 生成带独立前缀的客户端接入注册令牌。
func generateRegistrationToken() (string, error) {
	token, err := generateAgentToken()
	if err != nil {
		return "", err
	}
	return "rar_" + strings.TrimPrefix(token, "ra_"), nil
}

// hashAgentToken 生成不可逆令牌摘要，数据库不保存任何令牌明文。
func hashAgentToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
