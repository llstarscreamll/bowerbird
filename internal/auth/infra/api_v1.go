package infra

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
)

type contextKey string

const userContextKey contextKey = "user"

func RegisterRoutes(mux *http.ServeMux, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, googleAuth domain.AuthServerGateway, userRepo domain.UserRepository, sessionRepo domain.SessionRepository, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) {
	mux.HandleFunc("GET /v1/auth/google/login", googleLoginHandler(googleAuth))
	mux.HandleFunc("GET /v1/auth/google/callback", googleLoginCallbackHandler(config, ulid, googleAuth, userRepo, sessionRepo))

	mux.HandleFunc("GET /v1/auth/google-mail/login", authMiddleware(mailLoginHandler("google", googleAuth), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/google-mail/callback", authMiddleware(gMailLoginCallbackHandler("google", config, ulid, googleAuth, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /v1/auth/outlook/login", authMiddleware(mailLoginHandler("microsoft", googleAuth), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/outlook/callback", authMiddleware(gMailLoginCallbackHandler("microsoft", config, ulid, googleAuth, crypt, mailSecretRepo), sessionRepo, userRepo))
}

// redirects the user to the Google login page
func googleLoginHandler(authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url, err := authServer.GetLoginUrl("google", []string{})
		if err != nil {
			log.Printf("Error getting auth server login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting auth server login url -> "+err.Error())
			return
		}

		http.Redirect(w, r, url, http.StatusFound)
	}
}

// handles the Google login callback by upsert the user in database, starting session and redirecting to frontend app
func googleLoginCallbackHandler(config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServerGateway, userRepo domain.UserRepository, sessionRepo domain.SessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.Trim(r.URL.Query().Get("code"), " ")
		if code == "" {
			log.Printf("Error getting code from query params, code is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"code is empty"}]}`)
			return
		}

		user, err := authServer.GetUserProfile(r.Context(), "google", code)
		if err != nil {
			log.Printf("Error getting user profile form OAuth server: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error getting user profiles from OAuth Server","detail":%q}]}`, err.Error())
			return
		}

		user.ID = ulid.New()

		err = userRepo.Upsert(r.Context(), user)
		if err != nil {
			log.Printf("Error creating user in database: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error writing user in storage","detail":%q}]}`, err.Error())
			return
		}

		sessionExpirationDate := time.Now().Add(time.Hour * 2)
		sessionID, err := ulid.NewFromDate(sessionExpirationDate)
		if err != nil {
			log.Printf("Error generating session ID: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error generating session ID","detail":%q}]}`, err.Error())
			return
		}

		err = sessionRepo.Save(r.Context(), sessionID, user.ID, sessionExpirationDate)
		if err != nil {
			log.Printf("Error storing session: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error storing session","detail":%q}]}`, err.Error())
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
		})

		http.Redirect(w, r, config.FrontendUrl+"/dashboard", http.StatusFound)
	}
}

// redirects user to Google login page and request access to *read* Gmail
func mailLoginHandler(provider string, authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url, err := authServer.GetLoginUrl(provider, []string{"https://www.googleapis.com/auth/gmail.readonly"})
		if err != nil {
			log.Printf("Error getting auth server login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting auth server login url -> "+err.Error())
			return
		}

		http.Redirect(w, r, url, http.StatusFound)
	}
}

func gMailLoginCallbackHandler(provider string, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServerGateway, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.Trim(r.URL.Query().Get("code"), " ")
		tokens, err := authServer.GetTokens(r.Context(), provider, code)
		if err != nil {
			log.Printf("Error getting tokens from auth server: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting tokens from auth server -> "+err.Error())
			return
		}

		encryptedAccessToken, err := crypt.EncryptString(tokens.AccessToken)
		if err != nil {
			log.Printf("Error securing access token: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error securing access token -> "+err.Error())
			return
		}

		encryptedRefreshToken, err := crypt.EncryptString(tokens.RefreshToken)
		if err != nil {
			log.Printf("Error securing refresh token: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error securing refresh token -> "+err.Error())
			return
		}

		user := r.Context().Value(userContextKey).(domain.User)
		err = mailSecretRepo.Save(r.Context(), ulid.New(), user.ID, provider, encryptedAccessToken, encryptedRefreshToken, tokens.ExpiresAt)
		if err != nil {
			log.Printf("Error writing tokens in storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error writing tokens in storage -> "+err.Error())
			return
		}

		http.Redirect(w, r, config.FrontendUrl+"/dashboard", http.StatusFound)
	}
}
