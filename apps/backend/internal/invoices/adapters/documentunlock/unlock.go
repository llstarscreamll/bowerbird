package documentunlock

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	alexzip "github.com/alexmullins/zip"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	maxZIPMemberUncompressedBytes = 64 << 20  // 64MB per member
	maxZIPTotalUncompressedBytes  = 128 << 20 // 128MB per archive
)

type ExtractedMember struct {
	Name string
	Path string
}

func IsPDFEncrypted(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	lower := bytes.ToLower(data)
	return bytes.Contains(lower, []byte("/encrypt"))
}

func unlockPDF(data []byte, password string) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password

	var out bytes.Buffer
	if err := api.Decrypt(bytes.NewReader(data), &out, conf); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func TryUnlockPDF(data []byte, passwords []string) (unlocked []byte, passwordIndex int, err error) {
	if !IsPDFEncrypted(data) {
		return data, -1, nil
	}
	if len(passwords) == 0 {
		return nil, -1, fmt.Errorf("pdf is encrypted and no passwords configured")
	}
	var lastErr error
	for i, pw := range passwords {
		unlocked, err := unlockPDF(data, pw)
		if err == nil && len(unlocked) > 0 {
			return unlocked, i, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unable to unlock pdf with configured passwords")
	}
	return nil, -1, lastErr
}

// ExtractZIPMembersToDir opens a ZIP on disk and writes kept members under destDir.
func ExtractZIPMembersToDir(zipPath, destDir string, passwords []string, keep func(name, memberPath string) bool) ([]ExtractedMember, int, error) {
	info, err := os.Stat(zipPath)
	if err != nil {
		return nil, -1, err
	}

	archive, err := os.Open(zipPath)
	if err != nil {
		return nil, -1, err
	}
	defer archive.Close()

	reader, err := alexzip.NewReader(archive, info.Size())
	if err != nil {
		return nil, -1, err
	}

	needsPassword := false
	for _, f := range reader.File {
		if f.IsEncrypted() {
			needsPassword = true
			break
		}
	}

	tryPasswords := []string{""}
	if needsPassword {
		if len(passwords) == 0 {
			return nil, -1, fmt.Errorf("zip is encrypted and no passwords configured")
		}
		tryPasswords = passwords
	}

	var lastErr error
	for pwIdx, pw := range tryPasswords {
		if _, err := archive.Seek(0, io.SeekStart); err != nil {
			lastErr = err
			continue
		}
		reader, err := alexzip.NewReader(archive, info.Size())
		if err != nil {
			lastErr = err
			continue
		}

		members := make([]ExtractedMember, 0)
		ok := true
		var totalUncompressed int64
		for _, f := range reader.File {
			if f.FileInfo().IsDir() {
				continue
			}
			if f.UncompressedSize64 > maxZIPMemberUncompressedBytes {
				ok = false
				lastErr = fmt.Errorf("zip member %q exceeds uncompressed size limit", f.Name)
				break
			}
			memberPath, err := uniqueMemberPath(destDir, f.Name)
			if err != nil {
				ok = false
				lastErr = err
				break
			}
			if f.IsEncrypted() {
				f.SetPassword(pw)
			}
			rc, err := f.Open()
			if err != nil {
				ok = false
				lastErr = err
				break
			}
			written, err := writeLimitedMember(rc, memberPath)
			_ = rc.Close()
			if err != nil {
				ok = false
				lastErr = err
				break
			}
			totalUncompressed += written
			if totalUncompressed > maxZIPTotalUncompressedBytes {
				_ = os.Remove(memberPath)
				ok = false
				lastErr = fmt.Errorf("zip archive exceeds total uncompressed size limit")
				break
			}
			if keep != nil && !keep(f.Name, memberPath) {
				_ = os.Remove(memberPath)
				continue
			}
			members = append(members, ExtractedMember{Name: f.Name, Path: memberPath})
		}
		if ok {
			usedIdx := -1
			if needsPassword {
				usedIdx = pwIdx
			}
			return members, usedIdx, nil
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("unable to unlock zip with configured passwords")
	}
	return nil, -1, lastErr
}

func writeLimitedMember(rc io.Reader, destPath string) (int64, error) {
	file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, fmt.Errorf("open zip member dest: %w", err)
	}
	defer file.Close()

	limited := io.LimitReader(rc, maxZIPMemberUncompressedBytes+1)
	written, err := io.Copy(file, limited)
	if err != nil {
		_ = os.Remove(destPath)
		return 0, err
	}
	if written > maxZIPMemberUncompressedBytes {
		_ = os.Remove(destPath)
		return 0, fmt.Errorf("zip member exceeds uncompressed size limit")
	}
	return written, nil
}

func uniqueMemberPath(destDir, name string) (string, error) {
	base, err := safeMemberBase(name)
	if err != nil {
		return "", err
	}
	path := filepath.Join(destDir, base)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}
	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(destDir, fmt.Sprintf("%s-%d%s", strings.TrimSuffix(base, filepath.Ext(base)), i, filepath.Ext(base)))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("zip member name collision for %q", name)
}

func safeMemberBase(name string) (string, error) {
	clean := filepath.Base(filepath.FromSlash(name))
	if clean == "" || clean == "." || clean == ".." {
		return "", fmt.Errorf("invalid zip member name %q", name)
	}
	return clean, nil
}
