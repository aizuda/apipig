package agent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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
	commandName, environment := resolveCLIExecutable(e.config.CodexCommand, remoteModel.CLITypeCodex)
	command := exec.CommandContext(ctx, commandName, "--version")
	command.Env = mergedEnvironment(environment)
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (e *Executor) ExecuteTurn(ctx context.Context, cliType, permissionMode, workingDirectory, prompt string, emit ChunkEmitter) ExecutionResult {
	workingDirectory, err := e.prepareWorkingDirectory(workingDirectory)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	workspace, err := e.resolveProjectWorkspace(workingDirectory)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	commandName, args, err := e.cliCommand(cliType, permissionMode)
	if err != nil {
		return ExecutionResult{ErrorMessage: err.Error()}
	}
	stdout := newBoundedBuffer(maxCommandOutputBytes)
	stderr := newBoundedBuffer(maxErrorOutputBytes)
	commandName, environment := resolveCLIExecutable(commandName, cliType)
	cli := exec.CommandContext(ctx, commandName, args...)
	cli.Dir = workspace
	cli.Env = mergedEnvironment(environment)
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

func (e *Executor) cliCommand(cliType string, permissionModes ...string) (string, []string, error) {
	permissionMode := remoteModel.PermissionModeAutoEdit
	if len(permissionModes) > 0 {
		permissionMode = remoteModel.NormalizePermissionMode(permissionModes[0])
	}
	switch strings.ToUpper(strings.TrimSpace(cliType)) {
	case remoteModel.CLITypeCodex:
		return e.config.CodexCommand, codexStreamingArgs(e.config.CodexArgs, permissionMode), nil
	case remoteModel.CLITypeClaude:
		return e.config.ClaudeCommand, claudeStreamingArgs(e.config.ClaudeArgs, permissionMode), nil
	default:
		return "", nil, errors.New("unsupported CLI type: " + cliType)
	}
}

func codexStreamingArgs(args []string, permissionModes ...string) []string {
	permissionMode := remoteModel.PermissionModeAutoEdit
	if len(permissionModes) > 0 {
		permissionMode = remoteModel.NormalizePermissionMode(permissionModes[0])
	} else {
		for _, arg := range args {
			if arg == "--dangerously-bypass-approvals-and-sandbox" {
				permissionMode = remoteModel.PermissionModeFullAccess
				break
			}
		}
	}
	result := append([]string(nil), args...)
	execIndex := -1
	for index, arg := range result {
		if strings.EqualFold(arg, "exec") || strings.EqualFold(arg, "e") {
			execIndex = index
			break
		}
	}
	if execIndex < 0 {
		// Older generated configurations omitted the subcommand and could leave
		// a read-only sandbox flag in place. Conversation turns always use
		// `exec`, so add it before applying the managed sandbox policy.
		result = append([]string{"exec"}, result...)
		execIndex = 0
	}
	normalized := make([]string, 0, len(result)+4)
	hasBypass := false
	normalizedExecIndex := -1
	for index := 0; index < len(result); index++ {
		arg := result[index]
		switch {
		case arg == "--json":
		case arg == "--full-auto":
		case arg == "--ask-for-approval" || arg == "-a":
			if index+1 < len(result) {
				index++
			}
		case strings.HasPrefix(arg, "--ask-for-approval=") || strings.HasPrefix(arg, "-a="):
		case arg == "--dangerously-bypass-approvals-and-sandbox":
			if permissionMode == remoteModel.PermissionModeFullAccess && !hasBypass {
				hasBypass = true
				normalized = append(normalized, arg)
			}
		case arg == "--sandbox" || arg == "-s":
			if index+1 < len(result) {
				index++
			}
		case strings.HasPrefix(arg, "--sandbox=") || strings.HasPrefix(arg, "-s="):
		case (arg == "--config" || arg == "-c") && index+1 < len(result) && isManagedSandboxOverride(result[index+1]):
			index++
		case strings.HasPrefix(arg, "--config=") && isManagedSandboxOverride(strings.TrimPrefix(arg, "--config=")):
		case strings.HasPrefix(arg, "-c=") && isManagedSandboxOverride(strings.TrimPrefix(arg, "-c=")):
		default:
			if index == execIndex {
				normalizedExecIndex = len(normalized)
			}
			normalized = append(normalized, arg)
		}
	}
	required := []string{"--json"}
	if permissionMode == remoteModel.PermissionModeAutoEdit {
		required = append(required,
			"--sandbox", "workspace-write",
			"-c", "sandbox_workspace_write.network_access=true",
		)
	}
	insertAt := normalizedExecIndex + 1
	globalRequired := make([]string, 0, 2)
	if permissionMode == remoteModel.PermissionModeAutoEdit {
		globalRequired = append(globalRequired, "--ask-for-approval", "never")
	} else if !hasBypass {
		globalRequired = append(globalRequired, "--dangerously-bypass-approvals-and-sandbox")
	}
	withRequired := make([]string, 0, len(normalized)+len(required)+len(globalRequired))
	withRequired = append(withRequired, globalRequired...)
	withRequired = append(withRequired, normalized[:insertAt]...)
	withRequired = append(withRequired, required...)
	return append(withRequired, normalized[insertAt:]...)
}

func resolveCLIExecutable(command, cliType string) (string, []string) {
	resolved, err := exec.LookPath(command)
	if err != nil {
		resolved = command
	}
	if runtime.GOOS != "windows" {
		return resolved, nil
	}
	overrides := make([]string, 0, 3)
	if os.Getenv("HOME") == "" && os.Getenv("USERPROFILE") != "" {
		overrides = append(overrides, "HOME="+os.Getenv("USERPROFILE"))
	}
	if strings.EqualFold(cliType, remoteModel.CLITypeClaude) {
		if native := findNativeClaudeExecutable(resolved); native != "" {
			return native, overrides
		}
		return resolved, overrides
	}
	if !strings.EqualFold(cliType, remoteModel.CLITypeCodex) {
		return resolved, overrides
	}
	native, pathDirectory := findNativeCodexExecutable(resolved)
	if native == "" {
		return resolved, overrides
	}
	overrides = append(overrides, "CODEX_MANAGED_BY_NPM=1")
	if pathDirectory != "" {
		pathValue := pathDirectory
		if existing := os.Getenv("PATH"); existing != "" {
			pathValue += string(os.PathListSeparator) + existing
		}
		overrides = append(overrides, "PATH="+pathValue)
	}
	return native, overrides
}

func findNativeClaudeExecutable(resolved string) string {
	packageRoots := make([]string, 0, 2)
	for current := filepath.Dir(resolved); current != filepath.Dir(current); current = filepath.Dir(current) {
		if strings.EqualFold(filepath.Base(current), "claude-code") && strings.EqualFold(filepath.Base(filepath.Dir(current)), "@anthropic-ai") {
			packageRoots = append(packageRoots, current)
			break
		}
		if strings.EqualFold(filepath.Base(current), ".nvmd") {
			version, err := os.ReadFile(filepath.Join(current, "default"))
			if err == nil && strings.TrimSpace(string(version)) != "" {
				packageRoots = append(packageRoots, filepath.Join(current, "versions", strings.TrimSpace(string(version)), "node_modules", "@anthropic-ai", "claude-code"))
			}
			break
		}
	}
	for _, root := range packageRoots {
		candidate := filepath.Join(root, "bin", "claude.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func findNativeCodexExecutable(resolved string) (string, string) {
	packageRoots := make([]string, 0, 2)
	for current := filepath.Dir(resolved); current != filepath.Dir(current); current = filepath.Dir(current) {
		if strings.EqualFold(filepath.Base(current), "codex") && strings.EqualFold(filepath.Base(filepath.Dir(current)), "@openai") {
			packageRoots = append(packageRoots, current)
			break
		}
		if strings.EqualFold(filepath.Base(current), ".nvmd") {
			version, err := os.ReadFile(filepath.Join(current, "default"))
			if err == nil && strings.TrimSpace(string(version)) != "" {
				packageRoots = append(packageRoots, filepath.Join(current, "versions", strings.TrimSpace(string(version)), "node_modules", "@openai", "codex"))
			}
			break
		}
	}
	patterns := []string{
		filepath.Join("vendor", "*", "bin", "codex.exe"),
		filepath.Join("vendor", "*", "codex", "codex.exe"),
		filepath.Join("node_modules", "@openai", "codex-win32-*", "vendor", "*", "bin", "codex.exe"),
		filepath.Join("node_modules", "@openai", "codex-win32-*", "vendor", "*", "codex", "codex.exe"),
	}
	for _, root := range packageRoots {
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(filepath.Join(root, pattern))
			sort.Strings(matches)
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil && !info.IsDir() {
					archRoot := filepath.Dir(filepath.Dir(match))
					for _, name := range []string{"codex-path", "path"} {
						pathDirectory := filepath.Join(archRoot, name)
						if info, err := os.Stat(pathDirectory); err == nil && info.IsDir() {
							return match, pathDirectory
						}
					}
					return match, ""
				}
			}
		}
	}
	return "", ""
}

func mergedEnvironment(overrides []string) []string {
	if len(overrides) == 0 {
		return nil
	}
	environment := os.Environ()
	for _, override := range overrides {
		key, _, found := strings.Cut(override, "=")
		if !found {
			continue
		}
		filtered := environment[:0]
		for _, entry := range environment {
			entryKey, _, _ := strings.Cut(entry, "=")
			if !strings.EqualFold(entryKey, key) {
				filtered = append(filtered, entry)
			}
		}
		environment = append(filtered, override)
	}
	return environment
}

func claudeStreamingArgs(args []string, permissionMode string) []string {
	result := make([]string, 0, len(args)+2)
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--dangerously-skip-permissions":
		case arg == "--permission-mode":
			if index+1 < len(args) {
				index++
			}
		case strings.HasPrefix(arg, "--permission-mode="):
		default:
			result = append(result, arg)
		}
	}
	if remoteModel.NormalizePermissionMode(permissionMode) == remoteModel.PermissionModeFullAccess {
		return append(result, "--dangerously-skip-permissions")
	}
	return append(result, "--permission-mode", "acceptEdits")
}

func isManagedSandboxOverride(value string) bool {
	key, _, found := strings.Cut(strings.TrimSpace(value), "=")
	if !found {
		return false
	}
	key = strings.TrimSpace(key)
	return strings.EqualFold(key, "sandbox_mode") ||
		strings.EqualFold(key, "sandbox_workspace_write.network_access")
}

func codexSandboxMode(args []string) string {
	for index, arg := range args {
		if arg == "--dangerously-bypass-approvals-and-sandbox" {
			return "danger-full-access"
		}
		if (arg == "--sandbox" || arg == "-s") && index+1 < len(args) {
			return args[index+1]
		}
		if value, found := strings.CutPrefix(arg, "--sandbox="); found {
			return value
		}
		if value, found := strings.CutPrefix(arg, "-s="); found {
			return value
		}
	}
	return "default"
}

func codexNetworkAccess(args []string) string {
	for index := 0; index < len(args); index++ {
		configValue := ""
		switch {
		case (args[index] == "--config" || args[index] == "-c") && index+1 < len(args):
			configValue = args[index+1]
			index++
		case strings.HasPrefix(args[index], "--config="):
			configValue = strings.TrimPrefix(args[index], "--config=")
		case strings.HasPrefix(args[index], "-c="):
			configValue = strings.TrimPrefix(args[index], "-c=")
		}
		key, value, found := strings.Cut(strings.TrimSpace(configValue), "=")
		if found && strings.EqualFold(strings.TrimSpace(key), "sandbox_workspace_write.network_access") {
			return strings.TrimSpace(value)
		}
	}
	return "default"
}

func sanitizeCLIArgs(args []string) []string {
	result := append([]string(nil), args...)
	for index := 0; index < len(result); index++ {
		arg := result[index]
		if key, _, found := strings.Cut(arg, "="); found && isSensitiveArgKey(key) {
			result[index] = key + "=<redacted>"
			continue
		}
		if isSensitiveArgKey(arg) && index+1 < len(result) {
			result[index+1] = "<redacted>"
			index++
		}
	}
	return result
}

func isSensitiveArgKey(value string) bool {
	value = strings.ToLower(strings.TrimLeft(strings.TrimSpace(value), "-"))
	return strings.Contains(value, "token") || strings.Contains(value, "key") ||
		strings.Contains(value, "secret") || strings.Contains(value, "password") ||
		strings.Contains(value, "credential")
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
