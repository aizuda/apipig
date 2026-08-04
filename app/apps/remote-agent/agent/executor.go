package agent

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"apipig/toolkit/snowflake"
)

const maxCommandOutputBytes = 1024 * 1024
const maxErrorOutputBytes = 64 * 1024

type ExecutionResult struct {
	Success      bool
	Content      string
	ErrorMessage string
}

type Executor struct{ config Config }
type ChunkEmitter func(content []byte)

func NewExecutor(config Config) *Executor { return &Executor{config: config} }

func (e *Executor) CodexVersion(ctx context.Context) string {
	command := exec.CommandContext(ctx, e.config.CodexCommand, "--version")
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (e *Executor) ExecuteTurn(ctx context.Context, conversationID snowflake.ID, prompt string, emit ChunkEmitter) ExecutionResult {
	workspace, err := e.prepareConversationWorkspace(conversationID)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	stdout := newBoundedBuffer(maxCommandOutputBytes)
	stderr := newBoundedBuffer(maxErrorOutputBytes)
	codex := exec.CommandContext(ctx, e.config.CodexCommand, e.config.CodexArgs...)
	codex.Dir = workspace
	codex.Stdin = strings.NewReader(prompt)
	codex.Stdout = &boundedEmitterWriter{buffer: stdout, emit: emit}
	codex.Stderr = stderr
	err = codex.Run()
	result := ExecutionResult{Success: err == nil, Content: stdout.String()}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		result.ErrorMessage = message
	}
	return result
}

func (e *Executor) prepareConversationWorkspace(conversationID snowflake.ID) (string, error) {
	if conversationID == 0 {
		return "", errors.New("conversation ID is required")
	}
	if err := os.MkdirAll(e.config.WorkspaceRoot, 0750); err != nil {
		return "", err
	}
	workspace := filepath.Join(e.config.WorkspaceRoot, "conversations", conversationID.String())
	if !withinRoot(e.config.WorkspaceRoot, workspace) {
		return "", errors.New("conversation workspace escapes workspace root")
	}
	if err := os.MkdirAll(workspace, 0750); err != nil {
		return "", err
	}
	return workspace, nil
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

type boundedEmitterWriter struct {
	buffer *boundedBuffer
	emit   ChunkEmitter
}

func (w *boundedEmitterWriter) Write(data []byte) (int, error) {
	original := len(data)
	remaining := w.buffer.limit - w.buffer.buffer.Len()
	if remaining <= 0 {
		return original, nil
	}
	if len(data) > remaining {
		data = data[:remaining]
	}
	if _, err := w.buffer.buffer.Write(data); err != nil {
		return 0, err
	}
	if w.emit != nil {
		w.emit(data)
	}
	return original, nil
}
