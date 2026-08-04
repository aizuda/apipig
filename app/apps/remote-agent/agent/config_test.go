package agent

import (
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
	assert.Equal(t, []string{"exec", "--skip-git-repo-check", "-"}, config.CodexArgs)
	assert.Equal(t, 25, config.PollWaitSeconds)
	assert.Greater(t, config.RequestTimeoutSeconds, config.PollWaitSeconds)
}

func TestConfigRegistrationIncludesHostname(t *testing.T) {
	config := Config{AgentKey: "node", Name: "Node", hostname: "test-host"}

	registration := config.Registration()

	assert.Equal(t, "test-host", registration.Hostname)
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
