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

func RegisterRoutes(mux *http.ServeMux, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, googleAuth domain.AuthServer, userRepo domain.UserRepository, sessionRepo domain.SessionRepository, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) {
	mux.HandleFunc("GET /v1/auth/google/login", googleLoginHandler(googleAuth))
	mux.HandleFunc("GET /v1/auth/google/callback", googleLoginCallbackHandler(config, ulid, googleAuth, userRepo, sessionRepo))

	mux.HandleFunc("GET /v1/auth/google-mail/login", authMiddleware(googleMailLoginHandler(googleAuth), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/google-mail/callback", authMiddleware(googleMailLoginCallbackHandler(config, googleAuth, crypt, mailSecretRepo), sessionRepo, userRepo))
}

// redirects the user to the Google login page
func googleLoginHandler(authServer domain.AuthServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, authServer.GetLoginUrl([]string{}), http.StatusFound)
	}
}

// handles the Google login callback by upsert the user in database, starting session and redirecting to frontend app
func googleLoginCallbackHandler(config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServer, userRepo domain.UserRepository, sessionRepo domain.SessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.Trim(r.URL.Query().Get("code"), " ")
		if code == "" {
			log.Printf("Error getting code from query params, code is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"code is empty"}]}`)
			return
		}

		user, err := authServer.GetUserInfo(r.Context(), code)
		if err != nil {
			log.Printf("Error getting user info form auth token: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error getting user info from OAuth Server","detail":%q}]}`, err.Error())
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
func googleMailLoginHandler(authServer domain.AuthServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, authServer.GetLoginUrl([]string{"https://www.googleapis.com/auth/gmail.readonly"}), http.StatusFound)
	}
}

func googleMailLoginCallbackHandler(config commonDomain.AppConfig, authServer domain.AuthServer, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.Trim(r.URL.Query().Get("code"), " ")
		accessToken, refreshToken, expirationTime, err := authServer.GetTokens(r.Context(), code)
		if err != nil {
			log.Printf("Error getting tokens from auth server: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting tokens from auth server -> "+err.Error())
			return
		}

		user := r.Context().Value(userContextKey).(domain.User)
		err = mailSecretRepo.Save(r.Context(), user.ID, "google", crypt.EncryptString(accessToken), crypt.EncryptString(refreshToken), expirationTime)
		if err != nil {
			log.Printf("Error writing tokens in storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error writing tokens in storage -> "+err.Error())
			return
		}

		http.Redirect(w, r, config.FrontendUrl+"/dashboard", http.StatusFound)
	}
}
