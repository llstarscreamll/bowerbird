package commands

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/bowerbird/internal/connections/domain"
)

type memConnRepo struct {
	saved *domain.Connection
}

func (r *memConnRepo) ListActive(ctx context.Context) ([]*domain.Connection, error) {
	return nil, nil
}

func (r *memConnRepo) GetByID(ctx context.Context, id string) (*domain.Connection, error) {
	return nil, nil
}

func (r *memConnRepo) Upsert(ctx context.Context, conn *domain.Connection) error {
	r.saved = conn
	return nil
}

type passthroughCredentials struct{}

func (passthroughCredentials) Encrypt(plaintext []byte) ([]byte, error) {
	out := make([]byte, len(plaintext))
	copy(out, plaintext)
	for i := range out {
		out[i] ^= 0xff
	}
	return out, nil
}

func (passthroughCredentials) Decrypt(ciphertext []byte) ([]byte, error) {
	return nil, nil
}

func TestUpsertMailboxConnectionCommandSealsAndPersists(t *testing.T) {
	repo := &memConnRepo{}
	cmd := NewUpsertMailboxConnectionCommand(repo, passthroughCredentials{})
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	cmd.now = func() time.Time { return now }
	cmd.newID = func() string { return "conn-1" }

	plaintext := []byte(`{"refresh_token":"rt"}`)
	conn, err := cmd.Execute(context.Background(), UpsertMailboxConnectionInput{
		OwnerUserID:          "user-1",
		Provider:             "gmail",
		ProviderAccountEmail: "a@b.com",
		GrantedScopes:        []string{"mail.read"},
		TokenJSON:            plaintext,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if conn.ID != "conn-1" {
		t.Fatalf("id=%s", conn.ID)
	}
	if len(conn.EncryptedCredentials) == 0 {
		t.Fatal("expected sealed credentials")
	}
	if bytes.Equal(conn.EncryptedCredentials, plaintext) {
		t.Fatal("expected ciphertext to differ from plaintext")
	}
	if repo.saved == nil || repo.saved.ID != conn.ID {
		t.Fatal("expected connection to be persisted")
	}
}
