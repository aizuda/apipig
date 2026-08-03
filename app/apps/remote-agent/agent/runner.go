package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
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

const Version = "0.1.0"

type Runner struct {
	config            Config
	client            *Client
	executor          *Executor
	logger            *log.Logger
	busy              atomic.Bool
	codexVersionValue atomic.Value
	mu                sync.Mutex
	currentTask       snowflake.ID
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
	registration, err := r.registerUntilReady(ctx)
	if err != nil {
		return err
	}
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
		if command == nil {
			continue
		}
		r.handleCommand(ctx, *command)
	}
}

func (r *Runner) registerUntilReady(ctx context.Context) (result remoteRespRegisterResult, err error) {
	for {
		info := r.config.Registration()
		r.mu.Lock()
		info.CurrentTaskID = r.currentTask
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
		r.logger.Printf("registration failed: %v", registerErr)
		if !sleepContext(ctx, 5*time.Second) {
			return result, ctx.Err()
		}
	}
}

type remoteRespRegisterResult struct {
	HeartbeatIntervalSeconds int
}

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
			err := r.client.Heartbeat(ctx, remoteReq.HeartbeatRequest{
				MemoryUsed: int64(memory.Sys), Busy: r.busy.Load(), CodexVersion: r.codexVersionValue.Load().(string),
			})
			if err != nil {
				r.logger.Printf("heartbeat failed: %v", err)
			}
		}
	}
}

func (r *Runner) handleCommand(ctx context.Context, command Command) {
	switch command.Type {
	case remoteModel.CommandTypeExecuteTask:
		if !r.busy.CompareAndSwap(false, true) {
			r.logger.Printf("ignored execute command %s while another task is running", command.CommandID.String())
			return
		}
		if err := r.client.Acknowledge(ctx, command.CommandID); err != nil {
			r.busy.Store(false)
			r.logger.Printf("acknowledge execute command failed: %v", err)
			return
		}
		executionCtx, cancel := context.WithCancel(ctx)
		r.mu.Lock()
		r.currentTask = command.TaskID
		r.cancelCurrent = cancel
		r.mu.Unlock()
		go r.execute(executionCtx, ctx, command)
	case remoteModel.CommandTypeStopTask:
		if err := r.client.Acknowledge(ctx, command.CommandID); err != nil {
			r.logger.Printf("acknowledge stop command failed: %v", err)
			return
		}
		r.mu.Lock()
		stopped := false
		if r.currentTask == command.TaskID && r.cancelCurrent != nil {
			r.cancelCurrent()
			stopped = true
		}
		r.mu.Unlock()
		if !stopped {
			go r.reportStoppedTask(ctx, command.TaskID)
		}
	default:
		r.logger.Printf("ignored unsupported command type %q", command.Type)
	}
}

func (r *Runner) reportStoppedTask(ctx context.Context, taskID snowflake.ID) {
	result := TaskResult{TaskID: taskID, Success: false, ErrorMessage: "task cancelled"}
	for {
		reportCtx, cancel := context.WithTimeout(ctx, time.Duration(r.config.RequestTimeoutSeconds)*time.Second)
		err := r.client.Complete(reportCtx, result)
		cancel()
		if err == nil {
			return
		}
		r.logger.Printf("report stopped task failed: %v", err)
		if !sleepContext(ctx, 5*time.Second) {
			return
		}
	}
}

func (r *Runner) execute(executionCtx, lifecycleCtx context.Context, command Command) {
	defer func() {
		r.mu.Lock()
		r.currentTask = 0
		r.cancelCurrent = nil
		r.mu.Unlock()
		r.busy.Store(false)
	}()
	r.logger.Printf("starting task %s", command.TaskID.String())
	uploader := newLogUploader(lifecycleCtx, r.client, command.TaskID, command.NextLogSequence)
	result := r.executor.Execute(executionCtx, command, uploader.Append)
	if err := uploader.Close(lifecycleCtx); err != nil {
		r.logger.Printf("flush task logs failed: %v", err)
		return
	}
	report := TaskResult{
		TaskID: command.TaskID, Success: result.Success, Result: result.Result,
		ErrorMessage: result.ErrorMessage, ChangedFiles: result.ChangedFiles,
	}
	for {
		reportCtx, cancel := context.WithTimeout(lifecycleCtx, time.Duration(r.config.RequestTimeoutSeconds)*time.Second)
		err := r.client.Complete(reportCtx, report)
		cancel()
		if err == nil {
			r.logger.Printf("finished task %s success=%t", command.TaskID.String(), result.Success)
			return
		}
		r.logger.Printf("report task result failed: %v", err)
		if !sleepContext(lifecycleCtx, 5*time.Second) {
			return
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
