package documentunlock

import (
	"bytes"
	"fmt"
	"io"

	alexzip "github.com/alexmullins/zip"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

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

// ExtractZIPMembers opens a ZIP (optionally password-protected) and returns member bytes for names that pass keep.
func ExtractZIPMembers(data []byte, passwords []string, keep func(name string, content []byte) bool) ([]struct {
	Name string
	Data []byte
}, int, error) {
	reader, err := alexzip.NewReader(bytes.NewReader(data), int64(len(data)))
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
		reader, err := alexzip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			lastErr = err
			continue
		}

		members := make([]struct {
			Name string
			Data []byte
		}, 0)
		ok := true
		for _, f := range reader.File {
			if f.FileInfo().IsDir() {
				continue
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
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, rc)
			_ = rc.Close()
			if err != nil {
				ok = false
				lastErr = err
				break
			}
			content := buf.Bytes()
			name := f.Name
			if keep != nil && !keep(name, content) {
				continue
			}
			members = append(members, struct {
				Name string
				Data []byte
			}{Name: name, Data: content})
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
