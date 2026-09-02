package temp

import (
	"fmt"
	"os"
	"strings"
)

const EnvWorkTempDir = "BOWERBIRD_WORK_TEMP_DIR"

// WorkDir returns the base directory for short-lived worker files (zip staging, extraction).
// Precedence: BOWERBIRD_WORK_TEMP_DIR → TMPDIR → os.TempDir().
func WorkDir() string {
	if dir := strings.TrimSpace(os.Getenv(EnvWorkTempDir)); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(os.Getenv("TMPDIR")); dir != "" {
		return dir
	}
	return os.TempDir()
}

func EnsureWorkDir() error {
	dir := WorkDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("ensure work temp dir %q: %w", dir, err)
	}
	return nil
}

func CreateFile(pattern string) (*os.File, error) {
	if err := EnsureWorkDir(); err != nil {
		return nil, err
	}
	return os.CreateTemp(WorkDir(), pattern)
}

func MkdirDir(pattern string) (string, error) {
	if err := EnsureWorkDir(); err != nil {
		return "", err
	}
	return os.MkdirTemp(WorkDir(), pattern)
}
