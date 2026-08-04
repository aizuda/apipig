package agent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	remoteModel "apipig/app/apps/remote-agent/model"
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

func (e *Executor) ExecuteTurn(ctx context.Context, cliType, workingDirectory, prompt string, emit ChunkEmitter) ExecutionResult {
	workingDirectory, err := e.prepareWorkingDirectory(workingDirectory)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	workspace, err := e.resolveProjectWorkspace(workingDirectory)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	commandName, args, err := e.cliCommand(cliType)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	stdout := newBoundedBuffer(maxCommandOutputBytes)
	stderr := newBoundedBuffer(maxErrorOutputBytes)
	cli := exec.CommandContext(ctx, commandName, args...)
	cli.Dir = workspace
	cli.Stdin = strings.NewReader(prompt)
	var stdoutWriter io.Writer = &boundedEmitterWriter{buffer: stdout, emit: emit}
	var codexStream *codexJSONStreamWriter
	if isCodexJSONCommand(cliType, args) {
		codexStream = newCodexJSONStreamWriter(emit)
		stdoutWriter = codexStream
	}
	cli.Stdout = stdoutWriter
	cli.Stderr = &boundedEmitterWriter{buffer: stderr, emit: emit}
	err = cli.Run()
	content := stdout.String()
	if codexStream != nil {
		codexStream.Close()
		content = codexStream.Result()
	}
	result := ExecutionResult{Success: err == nil, Content: content}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		result.ErrorMessage = message
	}
	return result
}

func (e *Executor) prepareWorkingDirectory(workingDirectory string) (string, error) {
	workingDirectory = strings.TrimSpace(workingDirectory)
	if workingDirectory == "" {
		return "", errors.New("working directory is required")
	}
	if isAgentManagedWorkspace(workingDirectory) {
		if err := e.createAgentManagedWorkspace(workingDirectory); err != nil {
			return "", err
		}
	}
	return workingDirectory, nil
}

func isAgentManagedWorkspace(workingDirectory string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(workingDirectory)), "/")
	if len(parts) != 3 || parts[0] != ".apipig" || parts[1] != "conversations" {
		return false
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	return err == nil && id > 0
}

func (e *Executor) createAgentManagedWorkspace(workingDirectory string) error {
	root, err := filepath.EvalSymlinks(e.config.WorkspaceRoot)
	if err != nil {
		return errors.New("resolve workspace root: " + err.Error())
	}
	current := root
	for _, segment := range strings.Split(filepath.Clean(workingDirectory), string(filepath.Separator)) {
		if segment == "" || segment == "." {
			continue
		}
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			if err := os.Mkdir(current, 0750); err != nil && !errors.Is(err, os.ErrExist) {
				return errors.New("create agent-managed workspace: " + err.Error())
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return errors.New("inspect agent-managed workspace: " + statErr.Error())
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("agent-managed workspace path must contain directories only")
		}
	}
	return nil
}

func (e *Executor) cliCommand(cliType string) (string, []string, error) {
	switch strings.ToUpper(strings.TrimSpace(cliType)) {
	case remoteModel.CLITypeCodex:
		return e.config.CodexCommand, codexStreamingArgs(e.config.CodexArgs), nil
	case remoteModel.CLITypeClaude:
		return e.config.ClaudeCommand, e.config.ClaudeArgs, nil
	default:
		return "", nil, errors.New("unsupported CLI type: " + cliType)
	}
}

func codexStreamingArgs(args []string) []string {
	result := append([]string(nil), args...)
	if len(result) == 0 || !strings.EqualFold(result[0], "exec") {
		return result
	}
	normalized := make([]string, 0, len(result)+4)
	normalized = append(normalized, result[0])
	hasBypass := false
	for index := 1; index < len(result); index++ {
		arg := result[index]
		switch {
		case arg == "--json":
		case arg == "--full-auto":
		case arg == "--dangerously-bypass-approvals-and-sandbox":
			hasBypass = true
			normalized = append(normalized, arg)
		case arg == "--sandbox" || arg == "-s":
			if index+1 < len(result) {
				index++
			}
		case strings.HasPrefix(arg, "--sandbox=") || strings.HasPrefix(arg, "-s="):
		default:
			normalized = append(normalized, arg)
		}
	}
	required := []string{"--json"}
	if !hasBypass {
		required = append(required, "--full-auto", "--sandbox", "workspace-write")
	}
	return append(append(normalized[:1:1], required...), normalized[1:]...)
}

func isCodexJSONCommand(cliType string, args []string) bool {
	if !strings.EqualFold(strings.TrimSpace(cliType), remoteModel.CLITypeCodex) {
		return false
	}
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func (e *Executor) resolveProjectWorkspace(workingDirectory string) (string, error) {
	workingDirectory = strings.TrimSpace(workingDirectory)
	if workingDirectory == "" {
		return "", errors.New("working directory is required")
	}
	if filepath.IsAbs(workingDirectory) {
		return "", errors.New("working directory must be relative to workspace-root")
	}
	root, err := filepath.EvalSymlinks(e.config.WorkspaceRoot)
	if err != nil {
		return "", errors.New("resolve workspace root: " + err.Error())
	}
	workspace := filepath.Join(root, filepath.Clean(workingDirectory))
	if !withinRoot(root, workspace) {
		return "", errors.New("project directory escapes workspace-root")
	}
	workspace, err = filepath.EvalSymlinks(workspace)
	if err != nil {
		return "", errors.New("project directory must already exist: " + err.Error())
	}
	if !withinRoot(root, workspace) {
		return "", errors.New("project directory escapes workspace-root")
	}
	info, err := os.Stat(workspace)
	if err != nil || !info.IsDir() {
		return "", errors.New("project working directory is not a directory")
	}
	probe, err := os.CreateTemp(workspace, ".apipig-write-check-*")
	if err != nil {
		return "", errors.New("project working directory is not writable by the agent user: " + err.Error())
	}
	probePath := probe.Name()
	if closeErr := probe.Close(); closeErr != nil {
		_ = os.Remove(probePath)
		return "", errors.New("verify project working directory write access: " + closeErr.Error())
	}
	if err := os.Remove(probePath); err != nil {
		return "", errors.New("clean project write probe: " + err.Error())
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
