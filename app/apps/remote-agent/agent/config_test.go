package agent

import (
	buildversion "apipig/version"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigNormalizeDefaults(t *testing.T) {
	config := Config{
		ControllerURL:     "https://controller.example.com/",
		RegistrationToken: "secret", AgentKey: "node", Name: "node",
		WorkspaceRoot: t.TempDir(),
	}
	require.NoError(t, config.normalize())
	assert.Equal(t, "https://controller.example.com", config.ControllerURL)
	assert.Equal(t, "codex", config.CodexCommand)
	assert.Equal(t, []string{
		"exec", "--json", "--skip-git-repo-check", "-",
	}, config.CodexArgs)
	assert.Equal(t, "claude", config.ClaudeCommand)
	assert.Equal(t, []string{"-p"}, config.ClaudeArgs)
	assert.Equal(t, 25, config.PollWaitSeconds)
	assert.Greater(t, config.RequestTimeoutSeconds, config.PollWaitSeconds)
}

func TestConfigPreservesPersistedCodexArgsForPerConversationPolicy(t *testing.T) {
	config := Config{
		ControllerURL:     "https://controller.example.com",
		RegistrationToken: "secret",
		AgentKey:          "node",
		Name:              "node",
		WorkspaceRoot:     t.TempDir(),
		CodexArgs:         []string{"exec", "--sandbox", "read-only", "-c", "sandbox_mode=read-only", "-"},
	}

	require.NoError(t, config.normalize())
	assert.Equal(t, []string{
		"exec", "--sandbox", "read-only", "-c", "sandbox_mode=read-only", "-",
	}, config.CodexArgs)
}

func TestConfigCreatesMissingWorkspaceRoot(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "remote-agent-workspaces", "nested")
	config := Config{
		ControllerURL: "https://controller.example.com", RegistrationToken: "secret",
		AgentKey: "node", Name: "node", WorkspaceRoot: workspaceRoot,
	}

	err := config.normalize()

	require.NoError(t, err)
	expected, err := filepath.Abs(workspaceRoot)
	require.NoError(t, err)
	assert.Equal(t, expected, config.WorkspaceRoot)
	info, err := os.Stat(expected)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestConfigRejectsWorkspaceRootFile(t *testing.T) {
	workspaceRoot := filepath.Join(t.TempDir(), "workspace-file")
	require.NoError(t, os.WriteFile(workspaceRoot, []byte("not a directory"), 0600))
	config := Config{
		ControllerURL: "https://controller.example.com", RegistrationToken: "secret",
		AgentKey: "node", Name: "node", WorkspaceRoot: workspaceRoot,
	}

	err := config.normalize()

	require.EqualError(t, err, "workspace-root must reference a directory")
}

func TestConfigRegistrationIncludesHostname(t *testing.T) {
	config := Config{AgentKey: "node", Name: "Node", hostname: "test-host"}

	registration := config.Registration()

	assert.Equal(t, "test-host", registration.Hostname)
	assert.Equal(t, buildversion.Version, registration.AgentVersion)
}

func TestWithinRoot(t *testing.T) {
	root := t.TempDir()
	assert.True(t, withinRoot(root, filepath.Join(root, "task", "repo")))
	assert.False(t, withinRoot(root, filepath.Join(root, "..", "outside")))
}

func TestBoundedBuffer(t *testing.T) {
	buffer := newBoundedBuffer(4)
	written, err := buffer.Write([]byte("123456"))
	require.NoError(t, err)
	assert.Equal(t, 6, written)
	assert.Equal(t, "1234", buffer.String())
}

func TestLogChunkEndPreservesUTF8Boundary(t *testing.T) {
	content := append(make([]byte, maxMessageChunkBytes-1), []byte("中文")...)
	end := messageChunkEnd(content)
	assert.Equal(t, maxMessageChunkBytes-1, end)
	assert.True(t, utf8.Valid(content[:end]))
}
