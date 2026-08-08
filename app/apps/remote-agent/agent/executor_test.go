package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	remoteModel "apipig/app/apps/remote-agent/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundedEmitterWriterKeepsStreamAndResultConsistent(t *testing.T) {
	buffer := newBoundedBuffer(maxCommandOutputBytes)
	var streamed bytes.Buffer
	writer := &boundedEmitterWriter{buffer: buffer, emit: func(content []byte) {
		_, _ = streamed.Write(content)
	}}
	input := bytes.Repeat([]byte("x"), maxCommandOutputBytes+1024)

	written, err := writer.Write(input)

	assert.NoError(t, err)
	assert.Equal(t, len(input), written)
	assert.Len(t, buffer.String(), maxCommandOutputBytes)
	assert.Equal(t, buffer.String(), streamed.String())
}

func TestExecutorResolvesExistingProjectWithinWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "projects", "api")
	require.NoError(t, os.MkdirAll(project, 0750))
	executor := NewExecutor(Config{WorkspaceRoot: root})

	resolved, err := executor.resolveProjectWorkspace(filepath.Join("projects", "api"))

	require.NoError(t, err)
	expected, err := filepath.EvalSymlinks(project)
	require.NoError(t, err)
	assert.Equal(t, expected, resolved)
	entries, err := os.ReadDir(project)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestExecutorRejectsMissingAndEscapingProjectDirectories(t *testing.T) {
	root := t.TempDir()
	executor := NewExecutor(Config{WorkspaceRoot: root})

	_, err := executor.resolveProjectWorkspace("missing")
	require.ErrorContains(t, err, "project directory must already exist")
	_, err = executor.resolveProjectWorkspace(filepath.Join("..", "outside"))
	require.Error(t, err)
}

func TestExecutorSelectsConfiguredCLI(t *testing.T) {
	executor := NewExecutor(Config{
		CodexCommand: "codex-custom", CodexArgs: []string{"exec"},
		ClaudeCommand: "claude-custom", ClaudeArgs: []string{"-p"},
	})

	command, args, err := executor.cliCommand(remoteModel.CLITypeClaude)
	require.NoError(t, err)
	assert.Equal(t, "claude-custom", command)
	assert.Equal(t, []string{"-p", "--permission-mode", "acceptEdits"}, args)
	_, _, err = executor.cliCommand("shell")
	require.EqualError(t, err, "unsupported CLI type: shell")
}

func TestCodexCommandEnablesStructuredStreaming(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{"exec", "--skip-git-repo-check", "-"}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--ask-for-approval", "never", "exec", "--json", "--sandbox", "workspace-write",
		"-c", "sandbox_workspace_write.network_access=true",
		"--skip-git-repo-check", "-",
	}, args)
	assert.Equal(t, []string{"exec", "--skip-git-repo-check", "-"}, executor.config.CodexArgs)
}

func TestCodexCommandOverridesReadOnlySandbox(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{
		"exec", "--json", "--full-auto", "--sandbox", "read-only", "--skip-git-repo-check", "-",
	}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--ask-for-approval", "never", "exec", "--json", "--sandbox", "workspace-write",
		"-c", "sandbox_workspace_write.network_access=true", "--skip-git-repo-check", "-",
	}, args)
}

func TestCodexCommandOverridesReadOnlyGlobalAndConfigArguments(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{
		"--profile", "automation", "-a", "untrusted", "--sandbox", "read-only", "-c", `sandbox_mode="read-only"`,
		"exec", "--config=sandbox_mode='read-only'",
		"--config=sandbox_workspace_write.network_access=false", "--json", "-",
	}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--ask-for-approval", "never", "--profile", "automation", "exec", "--json", "--sandbox", "workspace-write",
		"-c", "sandbox_workspace_write.network_access=true", "-",
	}, args)
	assert.Equal(t, "workspace-write", codexSandboxMode(args))
	assert.Equal(t, "true", codexNetworkAccess(args))
}

func TestCodexCommandAddsExecAndOverridesReadOnlyWithoutSubcommand(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{
		"--sandbox", "read-only", "--skip-git-repo-check", "-",
	}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--ask-for-approval", "never", "exec", "--json", "--sandbox", "workspace-write",
		"-c", "sandbox_workspace_write.network_access=true", "--skip-git-repo-check", "-",
	}, args)
}

func TestCodexCommandPreservesExplicitSandboxBypass(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{
		"--dangerously-bypass-approvals-and-sandbox", "exec", "--sandbox", "read-only", "-",
	}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex, remoteModel.PermissionModeFullAccess)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--dangerously-bypass-approvals-and-sandbox", "exec", "--json", "-",
	}, args)
	assert.Equal(t, "danger-full-access", codexSandboxMode(args))
	assert.Equal(t, "default", codexNetworkAccess(args))
}

func TestCodexCommandUsesFullAccessPermissionMode(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{
		"exec", "--sandbox", "read-only", "--skip-git-repo-check", "-",
	}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex, remoteModel.PermissionModeFullAccess)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"--dangerously-bypass-approvals-and-sandbox", "exec", "--json", "--skip-git-repo-check", "-",
	}, args)
	assert.Equal(t, "danger-full-access", codexSandboxMode(args))
}

func TestClaudeCommandMapsPermissionModes(t *testing.T) {
	executor := NewExecutor(Config{ClaudeCommand: "claude", ClaudeArgs: []string{
		"-p", "--permission-mode", "default",
	}})

	_, autoEdit, err := executor.cliCommand(remoteModel.CLITypeClaude, remoteModel.PermissionModeAutoEdit)
	require.NoError(t, err)
	assert.Equal(t, []string{"-p", "--permission-mode", "acceptEdits"}, autoEdit)

	_, fullAccess, err := executor.cliCommand(remoteModel.CLITypeClaude, remoteModel.PermissionModeFullAccess)
	require.NoError(t, err)
	assert.Equal(t, []string{"-p", "--dangerously-skip-permissions"}, fullAccess)
}

func TestSanitizeCLIArgsRedactsCredentialValues(t *testing.T) {
	assert.Equal(t, []string{"--api-key=<redacted>", "-c", "token=<redacted>", "exec"}, sanitizeCLIArgs([]string{
		"--api-key=secret-value", "-c", "token=another-secret", "exec",
	}))
}

func TestExecutorRunsCodexWithWorkspaceWriteAndWritesInProject(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	require.NoError(t, os.Mkdir(project, 0750))
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	writtenFile := filepath.Join(project, "fake-codex-write.txt")
	t.Setenv("APIPIG_FAKE_CODEX_ARGS", argsFile)
	t.Setenv("APIPIG_FAKE_CODEX_WRITE", writtenFile)
	command := writeFakeCodex(t)
	executor := NewExecutor(Config{
		WorkspaceRoot: root,
		CodexCommand:  command,
		CodexArgs:     []string{"exec", "--skip-git-repo-check", "-"},
	})

	result := executor.ExecuteTurn(context.Background(), remoteModel.CLITypeCodex, remoteModel.PermissionModeAutoEdit, "project", "write a file", nil)

	require.True(t, result.Success, result.ErrorMessage)
	assert.Equal(t, "fake completed", result.Content)
	argsContent, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	effectiveArgs := string(argsContent)
	assert.Contains(t, effectiveArgs, "--ask-for-approval never")
	assert.Contains(t, effectiveArgs, "--sandbox workspace-write")
	assert.Contains(t, effectiveArgs, "sandbox_workspace_write.network_access=true")
	written, err := os.ReadFile(writtenFile)
	require.NoError(t, err)
	assert.Equal(t, "ok", strings.TrimSpace(string(written)))
}

func TestFindNativeCodexExecutableFromNVMDShim(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("NVMD executable resolution is Windows-specific")
	}
	nvmdRoot := filepath.Join(t.TempDir(), ".nvmd")
	shim := filepath.Join(nvmdRoot, "bin", "codex.exe")
	packageRoot := filepath.Join(nvmdRoot, "versions", "22.0.0", "node_modules", "@openai", "codex")
	archRoot := filepath.Join(packageRoot, "node_modules", "@openai", "codex-win32-x64", "vendor", "x86_64-pc-windows-msvc")
	native := filepath.Join(archRoot, "bin", "codex.exe")
	pathDirectory := filepath.Join(archRoot, "codex-path")
	require.NoError(t, os.MkdirAll(filepath.Dir(shim), 0750))
	require.NoError(t, os.MkdirAll(filepath.Dir(native), 0750))
	require.NoError(t, os.MkdirAll(pathDirectory, 0750))
	require.NoError(t, os.WriteFile(shim, []byte("shim"), 0750))
	require.NoError(t, os.WriteFile(native, []byte("native"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(nvmdRoot, "default"), []byte("22.0.0\n"), 0600))

	resolved, resolvedPath := findNativeCodexExecutable(shim)

	assert.Equal(t, native, resolved)
	assert.Equal(t, pathDirectory, resolvedPath)
}

func TestFindNativeClaudeExecutableFromNVMDShim(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("NVMD executable resolution is Windows-specific")
	}
	nvmdRoot := filepath.Join(t.TempDir(), ".nvmd")
	shim := filepath.Join(nvmdRoot, "bin", "claude.exe")
	native := filepath.Join(nvmdRoot, "versions", "22.0.0", "node_modules", "@anthropic-ai", "claude-code", "bin", "claude.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(shim), 0750))
	require.NoError(t, os.MkdirAll(filepath.Dir(native), 0750))
	require.NoError(t, os.WriteFile(shim, []byte("shim"), 0750))
	require.NoError(t, os.WriteFile(native, []byte("native"), 0750))
	require.NoError(t, os.WriteFile(filepath.Join(nvmdRoot, "default"), []byte("22.0.0\n"), 0600))

	assert.Equal(t, native, findNativeClaudeExecutable(shim))
}

func writeFakeCodex(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if runtime.GOOS == "windows" {
		path := filepath.Join(directory, "fake-codex.cmd")
		content := "@echo off\r\n" +
			"echo %* > \"%APIPIG_FAKE_CODEX_ARGS%\"\r\n" +
			"echo ok> \"%APIPIG_FAKE_CODEX_WRITE%\"\r\n" +
			"echo {\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"fake completed\"}}\r\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0750))
		return path
	}
	path := filepath.Join(directory, "fake-codex")
	content := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" > \"$APIPIG_FAKE_CODEX_ARGS\"\n" +
		"printf 'ok\\n' > \"$APIPIG_FAKE_CODEX_WRITE\"\n" +
		"printf '%s\\n' '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"fake completed\"}}'\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0750))
	return path
}

func TestCodexJSONStreamEmitsProgressAndKeepsFinalAnswer(t *testing.T) {
	var streamed bytes.Buffer
	writer := newCodexJSONStreamWriter(func(content []byte) {
		_, _ = streamed.Write(content)
	})
	lines := []string{
		`{"type":"thread.started","thread_id":"thread-1"}`,
		`{"type":"item.started","item":{"type":"command_execution","command":"go test ./..."}}`,
		`{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"ok apipig","exit_code":0}}`,
		`{"type":"item.completed","item":{"type":"reasoning","text":"Checking the implementation."}}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"All tests passed."}}`,
	}
	for _, line := range lines {
		_, err := writer.Write([]byte(line + "\n"))
		require.NoError(t, err)
	}
	writer.Close()

	assert.Contains(t, streamed.String(), "$ go test ./...")
	assert.Contains(t, streamed.String(), "ok apipig")
	assert.Contains(t, streamed.String(), "Checking the implementation.")
	assert.Contains(t, streamed.String(), "All tests passed.")
	assert.Equal(t, "All tests passed.", writer.Result())
}

func TestAgentManagedConversationWorkspaceIsCreated(t *testing.T) {
	root := t.TempDir()
	executor := NewExecutor(Config{WorkspaceRoot: root})
	workingDirectory := filepath.Join(".apipig", "conversations", "123")

	prepared, err := executor.prepareWorkingDirectory(workingDirectory)

	require.NoError(t, err)
	assert.Equal(t, workingDirectory, prepared)
	info, err := os.Stat(filepath.Join(root, workingDirectory))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExecutorRejectsMissingProtocolFields(t *testing.T) {
	executor := NewExecutor(Config{})

	_, err := executor.prepareWorkingDirectory("")
	require.EqualError(t, err, "working directory is required")
	_, _, err = executor.cliCommand("")
	require.EqualError(t, err, "unsupported CLI type: ")
}
