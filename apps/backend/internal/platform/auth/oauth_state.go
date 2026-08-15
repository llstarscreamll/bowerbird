package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"
)

const (
	oauthStateCookieName = "oauth_login_state"
	oauthStateTTL        = 10 * time.Minute
	oauthStateBytes      = 32
)

// NewOAuthState creates a random state and sets it in an HttpOnly cookie (SameSite=Lax for IdP redirects).
func NewOAuthState(w http.ResponseWriter) (string, error) {
	buf := make([]byte, oauthStateBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := base64.RawURLEncoding.EncodeToString(buf)
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/api/v1/auth",
		MaxAge:   int(oauthStateTTL.Seconds()),
		Expires:  time.Now().Add(oauthStateTTL),
	})
	return state, nil
}

// ConsumeOAuthState validates the callback state against the cookie and clears the cookie.
func ConsumeOAuthState(w http.ResponseWriter, r *http.Request, state string) bool {
	defer clearOAuthStateCookie(w)

	if state == "" {
		return false
	}
	cookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(state)) != 1 {
		return false
	}
	return true
}

func clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
