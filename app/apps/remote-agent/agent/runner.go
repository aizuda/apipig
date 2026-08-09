package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/toolkit/snowflake"
)

const Version = "0.5.0"

type Runner struct {
	config            Config
	client            *Client
	executor          *Executor
	logger            *log.Logger
	busy              atomic.Bool
	codexVersionValue atomic.Value
	mu                sync.Mutex
	currentMessage    snowflake.ID
	cancelCurrent     context.CancelFunc
}

func NewRunner(config Config, logger *log.Logger) *Runner {
	runner := &Runner{config: config, client: NewClient(config), executor: NewExecutor(config), logger: logger}
	runner.codexVersionValue.Store("")
	return runner
}

func EnsureNonRoot() error {
	current, err := user.Current()
	if err != nil {
		return err
	}
	if current.Uid == "0" {
		return errors.New("remote agent must not run as root")
	}
	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	r.logger.Printf("starting with config=%q controller=%q agent-key=%q registration-token-fingerprint=%s", r.config.sourcePath, r.config.ControllerURL, r.config.AgentKey, tokenFingerprint(r.config.RegistrationToken))
	r.logCodexExecutionPolicy()
	registration, err := r.registerUntilReady(ctx)
	if err != nil {
		return err
	}
	defer r.disconnect()
	heartbeatInterval := time.Duration(registration.HeartbeatIntervalSeconds) * time.Second
	if heartbeatInterval <= 0 {
		heartbeatInterval = 30 * time.Second
	}
	go r.heartbeatLoop(ctx, heartbeatInterval)
	for {
		select {
		case <-ctx.Done():
			r.stopCurrent()
			return ctx.Err()
		default:
		}
		command, err := r.client.NextCommand(ctx, r.config.PollWaitSeconds)
		if err != nil {
			r.logger.Printf("command poll failed: %v", err)
			if isAuthenticationError(err) {
				_, _ = r.registerUntilReady(ctx)
			}
			if !sleepContext(ctx, 3*time.Second) {
				return ctx.Err()
			}
			continue
		}
		if command != nil {
			r.handleCommand(ctx, *command)
		}
	}
}

func (r *Runner) logCodexExecutionPolicy() {
	command, args, err := r.executor.cliCommand(remoteModel.CLITypeCodex, remoteModel.PermissionModeFullAccess)
	if err != nil {
		r.logger.Printf("codex execution policy unavailable: %v", err)
		return
	}
	resolved, _ := resolveCLIExecutable(command, remoteModel.CLITypeCodex)
	if _, lookErr := exec.LookPath(command); lookErr != nil {
		r.logger.Printf("codex executable lookup failed command=%q: %v", command, lookErr)
	}
	r.logger.Printf("codex execution policy agent-version=%s executable=%q args=%q sandbox=%s network-access=%s", Version, resolved, sanitizeCLIArgs(args), codexSandboxMode(args), codexNetworkAccess(args))
}

func (r *Runner) disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.client.Disconnect(ctx); err != nil {
		r.logger.Printf("report agent disconnect failed: %v", err)
	}
}

func (r *Runner) registerUntilReady(ctx context.Context) (result remoteRespRegisterResult, err error) {
	for {
		info := r.config.Registration()
		r.mu.Lock()
		info.CurrentMessageID = r.currentMessage
		r.mu.Unlock()
		versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		info.CodexVersion = r.executor.CodexVersion(versionCtx)
		cancel()
		registration, registerErr := r.client.Register(ctx, info)
		if registerErr == nil {
			r.codexVersionValue.Store(info.CodexVersion)
			r.logger.Printf("registered agent %s as %s", r.config.Name, registration.AgentID.String())
			return remoteRespRegisterResult{HeartbeatIntervalSeconds: registration.HeartbeatIntervalSeconds}, nil
		}
		r.logger.Printf("registration failed for agent-key %q at %s using config %q: %v", r.config.AgentKey, r.config.ControllerURL, r.config.sourcePath, registerErr)
		if !sleepContext(ctx, 5*time.Second) {
			return result, ctx.Err()
		}
	}
}

func tokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:6])
}

type remoteRespRegisterResult struct{ HeartbeatIntervalSeconds int }

func (r *Runner) heartbeatLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			err := r.client.Heartbeat(ctx, remoteReq.HeartbeatRequest{MemoryUsed: int64(memory.Sys), Busy: r.busy.Load(), CodexVersion: r.codexVersionValue.Load().(string)})
			if err != nil {
				r.logger.Printf("heartbeat failed: %v", err)
			}
		}
	}
}

func (r *Runner) handleCommand(ctx context.Context, command Command) {
	if command.Type != remoteModel.CommandTypeConversationTurn {
		r.logger.Printf("ignored unsupported command type %q", command.Type)
		return
	}
	if !r.busy.CompareAndSwap(false, true) {
		r.logger.Printf("ignored conversation command %s while another turn is running", command.CommandID.String())
		return
	}
	if err := r.client.Acknowledge(ctx, command.CommandID); err != nil {
		r.busy.Store(false)
		r.logger.Printf("acknowledge conversation command failed: %v", err)
		return
	}
	executionCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.currentMessage = command.AssistantMessageID
	r.cancelCurrent = cancel
	r.mu.Unlock()
	go r.execute(executionCtx, cancel, ctx, command)
}

func (r *Runner) execute(executionCtx context.Context, cancelExecution context.CancelFunc, lifecycleCtx context.Context, command Command) {
	defer func() {
		r.mu.Lock()
		r.currentMessage = 0
		r.cancelCurrent = nil
		r.mu.Unlock()
		r.busy.Store(false)
	}()
	sandboxMode := "n/a"
	networkAccess := "n/a"
	executable := ""
	var effectiveArgs []string
	commandName, args, commandErr := r.executor.cliCommand(command.CLIType, command.PermissionMode)
	if commandErr == nil {
		executable = commandName
		effectiveArgs = args
		executable, _ = resolveCLIExecutable(commandName, command.CLIType)
	}
	if strings.EqualFold(command.CLIType, remoteModel.CLITypeCodex) {
		if commandErr == nil {
			sandboxMode = codexSandboxMode(effectiveArgs)
			networkAccess = codexNetworkAccess(effectiveArgs)
		}
	}
	r.logger.Printf(
		"starting conversation turn %s cli=%s permission-mode=%s executable=%q args=%q workspace-root=%q working-directory=%q sandbox=%s network-access=%s agent-version=%s",
		command.AssistantMessageID.String(), command.CLIType, remoteModel.NormalizePermissionMode(command.PermissionMode), executable, sanitizeCLIArgs(effectiveArgs), r.config.WorkspaceRoot,
		command.WorkingDirectory, sandboxMode, networkAccess, Version,
	)
	uploader := newMessageUploader(lifecycleCtx, r.client, command.AssistantMessageID, command.NextChunkSequence)
	controlCtx, stopControl := context.WithCancel(lifecycleCtx)
	var paused atomic.Bool
	var cancelled atomic.Bool
	go r.watchCommandControl(controlCtx, command.CommandID, &paused, &cancelled, cancelExecution)
	result := r.executor.ExecuteTurn(executionCtx, command.CLIType, command.PermissionMode, command.WorkingDirectory, command.Prompt, uploader.Append)
	stopControl()
	if err := uploader.Close(lifecycleCtx); err != nil {
		result.Success = false
		if result.ErrorMessage == "" {
			result.ErrorMessage = "upload assistant response: " + err.Error()
		}
	}
	report := MessageResult{MessageID: command.AssistantMessageID, Success: result.Success, Content: result.Content, ErrorMessage: result.ErrorMessage}
	if paused.Load() {
		report.Success = false
		report.Paused = true
		report.ErrorMessage = ""
	}
	if cancelled.Load() {
		report.Success = false
		report.Cancelled = true
		report.ErrorMessage = ""
	}
	for {
		reportCtx, cancel := context.WithTimeout(lifecycleCtx, time.Duration(r.config.RequestTimeoutSeconds)*time.Second)
		err := r.client.Complete(reportCtx, report)
		cancel()
		if err == nil {
			r.logger.Printf("finished conversation turn %s success=%t", command.AssistantMessageID.String(), result.Success)
			return
		}
		r.logger.Printf("report conversation result failed: %v", err)
		if !sleepContext(lifecycleCtx, 5*time.Second) {
			return
		}
	}
}

func (r *Runner) watchCommandControl(ctx context.Context, commandID snowflake.ID, paused, cancelled *atomic.Bool, cancel context.CancelFunc) {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, err := r.client.CommandStatus(ctx, commandID)
		if err == nil && (status == remoteModel.CommandStatusPauseRequested || status == remoteModel.CommandStatusPaused) {
			paused.Store(true)
			cancel()
			return
		}
		if err == nil && status == remoteModel.CommandStatusCancelled {
			cancelled.Store(true)
			cancel()
			return
		}
		if err != nil && ctx.Err() == nil {
			r.logger.Printf("command control poll failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Runner) stopCurrent() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancelCurrent != nil {
		r.cancelCurrent()
	}
}

func isAuthenticationError(err error) bool {
	var apiError *APIError
	if !errors.As(err, &apiError) {
		return false
	}
	message := strings.ToLower(apiError.Message)
	return strings.Contains(message, "agent token") || strings.Contains(message, "not registered")
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func logicalCPUInfo() string { return fmt.Sprintf("%d logical CPUs", runtime.NumCPU()) }
