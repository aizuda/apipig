package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const maxCommandOutputBytes = 1024 * 1024
const maxErrorOutputBytes = 64 * 1024

type ExecutionResult struct {
	Success      bool
	Result       string
	ErrorMessage string
	ChangedFiles []string
}

type Executor struct {
	config Config
}

type LogEmitter func(stream string, content []byte)

func NewExecutor(config Config) *Executor { return &Executor{config: config} }

func (e *Executor) CodexVersion(ctx context.Context) string {
	command := exec.CommandContext(ctx, e.config.CodexCommand, "--version")
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (e *Executor) Execute(ctx context.Context, command Command, emit LogEmitter) ExecutionResult {
	workspace, err := e.prepareWorkspace(ctx, command, emit)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	stdout := newBoundedBuffer(maxCommandOutputBytes)
	stderr := newBoundedBuffer(maxErrorOutputBytes)
	codex := exec.CommandContext(ctx, e.config.CodexCommand, e.config.CodexArgs...)
	codex.Dir = workspace
	codex.Stdin = strings.NewReader(command.Prompt)
	codex.Stdout = io.MultiWriter(stdout, logEmitterWriter{stream: "stdout", emit: emit})
	codex.Stderr = io.MultiWriter(stderr, logEmitterWriter{stream: "stderr", emit: emit})
	err = codex.Run()
	changedFiles := e.changedFiles(ctx, workspace)
	result := ExecutionResult{Success: err == nil, Result: stdout.String(), ChangedFiles: changedFiles}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		result.ErrorMessage = message
	}
	return result
}

func (e *Executor) prepareWorkspace(ctx context.Context, command Command, emit LogEmitter) (string, error) {
	if command.TaskID == 0 {
		return "", errors.New("task ID is required")
	}
	if err := os.MkdirAll(e.config.WorkspaceRoot, 0750); err != nil {
		return "", err
	}
	taskRoot := filepath.Join(e.config.WorkspaceRoot, command.TaskID.String())
	if !withinRoot(e.config.WorkspaceRoot, taskRoot) {
		return "", errors.New("task workspace escapes workspace root")
	}
	if err := os.RemoveAll(taskRoot); err != nil {
		return "", err
	}
	cloneOutput := newBoundedBuffer(maxCommandOutputBytes)
	clone := exec.CommandContext(ctx, e.config.GitCommand, "clone", "--depth", "1", "--", command.RepositoryURL, taskRoot)
	clone.Stdout = io.MultiWriter(cloneOutput, logEmitterWriter{stream: "stdout", emit: emit})
	clone.Stderr = io.MultiWriter(cloneOutput, logEmitterWriter{stream: "stderr", emit: emit})
	if err := clone.Run(); err != nil {
		return "", fmt.Errorf("git clone failed: %s", strings.TrimSpace(cloneOutput.String()))
	}
	workspace := taskRoot
	if command.WorkingDir != "" {
		workspace = filepath.Join(taskRoot, filepath.FromSlash(command.WorkingDir))
		if !withinRoot(taskRoot, workspace) {
			return "", errors.New("working directory escapes task workspace")
		}
		info, err := os.Stat(workspace)
		if err != nil || !info.IsDir() {
			return "", errors.New("working directory does not exist in repository")
		}
	}
	return workspace, nil
}

func (e *Executor) changedFiles(ctx context.Context, workspace string) []string {
	command := exec.CommandContext(ctx, e.config.GitCommand, "status", "--porcelain")
	command.Dir = workspace
	output, err := command.Output()
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) < 4 {
			continue
		}
		name := strings.TrimSpace(line[3:])
		if index := strings.LastIndex(name, " -> "); index >= 0 {
			name = name[index+4:]
		}
		if name != "" {
			seen[name] = struct{}{}
		}
	}
	files := make([]string, 0, len(seen))
	for name := range seen {
		files = append(files, name)
	}
	sort.Strings(files)
	return files
}

func withinRoot(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

type boundedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func newBoundedBuffer(limit int) *boundedBuffer { return &boundedBuffer{limit: limit} }

func (b *boundedBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := b.limit - b.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = b.buffer.Write(data)
	}
	return original, nil
}

func (b *boundedBuffer) String() string { return b.buffer.String() }

type logEmitterWriter struct {
	stream string
	emit   LogEmitter
}

func (w logEmitterWriter) Write(data []byte) (int, error) {
	if w.emit != nil && len(data) > 0 {
		w.emit(w.stream, data)
	}
	return len(data), nil
}
