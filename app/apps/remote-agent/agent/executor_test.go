package agent

import (
	"bytes"
	"os"
	"path/filepath"
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
	assert.Equal(t, []string{"-p"}, args)
	_, _, err = executor.cliCommand("shell")
	require.EqualError(t, err, "unsupported CLI type: shell")
}

func TestCodexCommandEnablesStructuredStreaming(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{"exec", "--skip-git-repo-check", "-"}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex)

	require.NoError(t, err)
	assert.Equal(t, []string{"exec", "--json", "--full-auto", "--sandbox", "workspace-write", "--skip-git-repo-check", "-"}, args)
	assert.Equal(t, []string{"exec", "--skip-git-repo-check", "-"}, executor.config.CodexArgs)
}

func TestCodexCommandOverridesReadOnlySandbox(t *testing.T) {
	executor := NewExecutor(Config{CodexCommand: "codex", CodexArgs: []string{
		"exec", "--json", "--full-auto", "--sandbox", "read-only", "--skip-git-repo-check", "-",
	}})

	_, args, err := executor.cliCommand(remoteModel.CLITypeCodex)

	require.NoError(t, err)
	assert.Equal(t, []string{
		"exec", "--json", "--full-auto", "--sandbox", "workspace-write", "--skip-git-repo-check", "-",
	}, args)
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
