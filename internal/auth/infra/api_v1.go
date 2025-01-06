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

func RegisterRoutes(mux *http.ServeMux, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, googleAuthService domain.AuthServer, userRepo domain.UserRepository, sessionRepo domain.SessionRepository) {
	mux.HandleFunc("GET /v1/auth/google/login", googleLoginHandler(googleAuthService))
	mux.HandleFunc("GET /v1/auth/google/callback", googleLoginCallbackHandler(config, ulid, googleAuthService, userRepo, sessionRepo))
}

// redirects the user to the Google login page
func googleLoginHandler(authServer domain.AuthServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, authServer.GetLoginUrl(), http.StatusFound)
	}
}

// handles the Google login callback by upsert the user in database, starting session and redirecting to frontend app
func googleLoginCallbackHandler(config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServer, repo domain.UserRepository, sessionRepo domain.SessionRepository) http.HandlerFunc {
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

		err = repo.Upsert(r.Context(), user)
		if err != nil {
			log.Printf("Error creating user in database: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error writing user in storage","detail":%q}]}`, err.Error())
			return
		}

		sessionID, err := ulid.NewFromDate(time.Now())
		if err != nil {
			log.Printf("Error generating session ID: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error generating session ID","detail":%q}]}`, err.Error())
			return
		}

		err = sessionRepo.Save(r.Context(), user.ID, sessionID)
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
