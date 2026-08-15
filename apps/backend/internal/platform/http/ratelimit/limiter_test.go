package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimiterAllow(t *testing.T) {
	l := New(2, time.Minute)
	require.True(t, l.Allow("a"))
	require.True(t, l.Allow("a"))
	require.False(t, l.Allow("a"))
	require.True(t, l.Allow("b"))
}

func TestProtect(t *testing.T) {
	l := New(1, time.Minute)
	called := 0
	h := l.Protect(func(w http.ResponseWriter, r *http.Request) error {
		called++
		return nil
	})

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	require.NoError(t, h(httptest.NewRecorder(), req))
	require.Error(t, h(httptest.NewRecorder(), req))
	require.Equal(t, 1, called)
}
