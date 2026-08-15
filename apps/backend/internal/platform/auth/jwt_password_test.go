package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidatePassword(t *testing.T) {
	require.ErrorIs(t, ValidatePassword("short"), ErrPasswordTooShort)
	require.NoError(t, ValidatePassword("longenough"))
	require.ErrorIs(t, ValidatePassword(string(make([]byte, 73))), ErrPasswordTooLong)
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	require.NoError(t, CheckPassword(hash, "correct-horse"))
	require.Error(t, CheckPassword(hash, "wrong-password"))
}

func TestGenerateAndValidateTokens(t *testing.T) {
	gen := NewTokenGenerator("access-secret", "refresh-secret", time.Minute, time.Hour)
	pair, err := gen.GenerateTokens("user-1", "a@b.co", "A", "B", "")
	require.NoError(t, err)
	require.NotEmpty(t, pair.RefreshJTI)

	claims, err := gen.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, "user-1", claims.UserID)

	userID, jti, err := gen.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	require.Equal(t, "user-1", userID)
	require.Equal(t, pair.RefreshJTI, jti)

	_, err = gen.ValidateAccessToken(pair.RefreshToken)
	require.Error(t, err)
}

func TestOAuthStateRoundTrip(t *testing.T) {
	rec := &responseRecorder{header: http.Header{}}
	state, err := NewOAuthState(rec)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	req := &http.Request{Header: http.Header{"Cookie": rec.header["Set-Cookie"]}}
	// Build request with cookie from Set-Cookie
	cookies := rec.header.Values("Set-Cookie")
	require.NotEmpty(t, cookies)
	req = &http.Request{Header: http.Header{}}
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: state})

	out := &responseRecorder{header: http.Header{}}
	require.True(t, ConsumeOAuthState(out, req, state))
	require.False(t, ConsumeOAuthState(out, req, "other"))
}

func TestOAuthStateRejectsMismatch(t *testing.T) {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	_ = base64.RawURLEncoding.EncodeToString(buf)

	req := &http.Request{Header: http.Header{}}
	req.AddCookie(&http.Cookie{Name: oauthStateCookieName, Value: "abc"})
	out := &responseRecorder{header: http.Header{}}
	require.False(t, ConsumeOAuthState(out, req, "xyz"))
}

type responseRecorder struct {
	header http.Header
	code   int
	body   []byte
}

func (r *responseRecorder) Header() http.Header { return r.header }
func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}
func (r *responseRecorder) WriteHeader(statusCode int) { r.code = statusCode }
