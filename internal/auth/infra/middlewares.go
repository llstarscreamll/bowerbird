package infra

import (
	"context"
	"fmt"
	"llstarscreamll/bowerbird/internal/auth/domain"
	"log"
	"net/http"
)

func authMiddleware(next http.Handler, sessionRepo domain.SessionRepository, userRepo domain.UserRepository) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionIDCookie, err := r.Cookie("session_token")
		if err != nil {
			log.Printf("Error retrieving session_token cookie: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"Error retrieving session_token cookie"}]}`)
			return
		}

		userID, err := sessionRepo.GetByID(r.Context(), sessionIDCookie.Value)
		if err != nil {
			log.Printf("Error getting session data from storage: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":"Error getting session data from storage"}]}`)
			return
		}
		if userID == "" {
			log.Printf("Session ID does not exists")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"errors":[{"status":"401","title":"Unauthorized","detail":"Session ID does not exists"}]}`)
			return
		}

		user, err := userRepo.GetByID(r.Context(), userID)
		if err != nil {
			log.Printf("Error getting auth user from storage: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":"Error getting auth user data from storage"}]}`)
			return
		}
		if user.ID == "" {
			log.Printf("Error getting auth user from storage, user %s not found", userID)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":"Error getting auth user from storage, user not found"}]}`)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
