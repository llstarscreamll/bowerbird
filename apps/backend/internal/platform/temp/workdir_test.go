package temp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkDirPrefersBowerbirdEnv(t *testing.T) {
	t.Setenv(EnvWorkTempDir, t.TempDir())
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "ignored"))

	require.Equal(t, os.Getenv(EnvWorkTempDir), WorkDir())
}

func TestWorkDirFallsBackToTmpDir(t *testing.T) {
	t.Setenv(EnvWorkTempDir, "")
	custom := t.TempDir()
	t.Setenv("TMPDIR", custom)

	require.Equal(t, custom, WorkDir())
}

func TestMkdirDirUsesWorkDir(t *testing.T) {
	base := t.TempDir()
	t.Setenv(EnvWorkTempDir, base)

	dir, err := MkdirDir("bowerbird-test-*")
	require.NoError(t, err)
	require.DirExists(t, dir)
	require.Contains(t, dir, base)
}
