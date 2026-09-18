package documentunlock

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	alexzip "github.com/alexmullins/zip"
	"github.com/stretchr/testify/require"
)

func TestExtractZIPMembersToDir_RejectsOversizedMember(t *testing.T) {
	zipPath := writeTempZIP(t, map[string][]byte{
		"large.xml": bytes.Repeat([]byte("x"), int(maxZIPMemberUncompressedBytes)+1),
	})
	destDir := t.TempDir()

	_, _, err := ExtractZIPMembersToDir(zipPath, destDir, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds uncompressed size limit")
}

func TestExtractZIPMembersToDir_RejectsTotalUncompressedBudget(t *testing.T) {
	chunk := bytes.Repeat([]byte("a"), 32<<20)
	zipPath := writeTempZIP(t, map[string][]byte{
		"a.xml": chunk,
		"b.xml": chunk,
		"c.xml": chunk,
		"d.xml": chunk,
		"e.xml": chunk,
	})
	destDir := t.TempDir()

	_, _, err := ExtractZIPMembersToDir(zipPath, destDir, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "total uncompressed size limit")
}

func TestExtractZIPMembersToDir_WritesKeptMembersToDisk(t *testing.T) {
	zipPath := writeTempZIP(t, map[string][]byte{
		"invoice.xml": []byte("<Invoice></Invoice>"),
		"readme.txt":  []byte("skip"),
	})
	destDir := t.TempDir()

	members, _, err := ExtractZIPMembersToDir(zipPath, destDir, nil, func(name, memberPath string) bool {
		return filepath.Ext(name) == ".xml"
	})
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.FileExists(t, members[0].Path)
	content, err := os.ReadFile(members[0].Path)
	require.NoError(t, err)
	require.Equal(t, "<Invoice></Invoice>", string(content))
}

func writeTempZIP(t *testing.T, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.zip")
	buf := new(bytes.Buffer)
	w := alexzip.NewWriter(buf)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))
	return path
}
