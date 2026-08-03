package toolkit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileMd5(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0600))
	assert.Equal(t, "b1946ac92492d2347c6235b4d2611184", GetFileMd5(path))
}
