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

func RegisterRoutes(mux *http.ServeMux, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authGateway domain.AuthServerGateway, userRepo domain.UserRepository, sessionRepo domain.SessionRepository, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository, mailGateway domain.MailGateway, mailMessageRepo domain.MailMessageRepository) {
	mux.HandleFunc("GET /v1/auth/google/login", googleLoginHandler(config, authGateway))
	mux.HandleFunc("GET /v1/auth/google/callback", googleLoginCallbackHandler(config, ulid, authGateway, userRepo, sessionRepo))

	mux.HandleFunc("GET /v1/auth/google-mail/login", authMiddleware(gMailLoginHandler("google", config, authGateway), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/google-mail/callback", authMiddleware(mailLoginCallbackHandler("google", config, ulid, authGateway, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /v1/auth/microsoft/login", authMiddleware(outlookLoginHandler("microsoft", config, authGateway), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/microsoft/callback", authMiddleware(mailLoginCallbackHandler("microsoft", config, ulid, authGateway, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("POST /v1/transactions/sync-from-mail", authMiddleware(syncTransactionsFromEmailHandler(crypt, mailSecretRepo, mailGateway, mailMessageRepo), sessionRepo, userRepo))
}

// redirects the user to the Google login page
func googleLoginHandler(config commonDomain.AppConfig, authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url, err := authServer.GetLoginUrl("google", config.ServerHost+"/v1/auth/google/callback", []string{})
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
		authCode := strings.Trim(r.URL.Query().Get("code"), " ")
		if authCode == "" {
			log.Printf("Error getting code from query params, code is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"code is empty"}]}`)
			return
		}

		tokens, err := authServer.GetTokens(r.Context(), "google", authCode)
		if err != nil {
			log.Printf("Error getting auth tokens from OAuth server: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error getting auth tokens from OAuth Server","detail":%q}]}`, err.Error())
			return
		}

		user, err := authServer.GetUserProfile(r.Context(), "google", tokens.AccessToken)
		if err != nil {
			log.Printf("Error getting user profile from OAuth server: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error getting user profile from OAuth Server","detail":%q}]}`, err.Error())
			return
		}

		user.ID = ulid.New()
		id, err := userRepo.Upsert(r.Context(), user)
		if err != nil {
			log.Printf("Error creating user in database: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error writing user in storage","detail":%q}]}`, err.Error())
			return
		}

		fmt.Printf("User.ID %s and returned user id %s", user.ID, id)
		user.ID = id
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

// redirects user to Google login page and request access to *read* mail
func gMailLoginHandler(provider string, config commonDomain.AppConfig, authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url, err := authServer.GetLoginUrl(provider, config.ServerHost+"/v1/auth/google-mail/callback", []string{"https://www.googleapis.com/auth/gmail.readonly"})
		if err != nil {
			log.Printf("Error getting auth server login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting auth server login url -> "+err.Error())
			return
		}

		http.Redirect(w, r, url, http.StatusFound)
	}
}

// redirects user to Microsoft login page and request access to *read* mail
func outlookLoginHandler(provider string, config commonDomain.AppConfig, authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url, err := authServer.GetLoginUrl(provider, config.ServerHost+"/v1/auth/microsoft/callback", []string{"https://graph.microsoft.com/Mail.Read", "https://graph.microsoft.com/User.Read"})
		if err != nil {
			log.Printf("Error getting auth server login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting auth server login url -> "+err.Error())
			return
		}

		http.Redirect(w, r, url, http.StatusFound)
	}
}

func mailLoginCallbackHandler(provider string, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServerGateway, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) http.HandlerFunc {
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

		fmt.Println("Callback refresh token: ", tokens.RefreshToken)

		encryptedRefreshToken, err := crypt.EncryptString(tokens.RefreshToken)
		if err != nil {
			log.Printf("Error securing refresh token: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error securing refresh token -> "+err.Error())
			return
		}

		authUser := r.Context().Value(userContextKey).(domain.User)
		userMailProfile, err := authServer.GetUserProfile(r.Context(), provider, tokens.AccessToken)
		if err != nil {
			log.Printf("Error getting user mail profile: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting user mail profile -> "+err.Error())
			return
		}

		err = mailSecretRepo.Save(r.Context(), ulid.New(), authUser.ID, provider, userMailProfile.Email, encryptedAccessToken, encryptedRefreshToken, tokens.ExpiresAt)
		if err != nil {
			log.Printf("Error writing tokens in storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error writing tokens in storage -> "+err.Error())
			return
		}

		http.Redirect(w, r, config.FrontendUrl+"/dashboard", http.StatusFound)
	}
}

func syncTransactionsFromEmailHandler(crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository, mailGateway domain.MailGateway, mailMessageRepo domain.MailMessageRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser := r.Context().Value(userContextKey).(domain.User)
		mailCredentials, err := mailSecretRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting mail credentials from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting mail credentials from storage -> "+err.Error())
			return
		}

		for _, c := range mailCredentials {
			if c.MailProvider != "google" {
				continue
			}

			decryptedAccessToken, err := crypt.DecryptString(c.AccessToken)
			if err != nil {
				log.Printf("Error decoding mail access token: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error decoding mail access token -> "+err.Error())
				return
			}

			decryptedRefreshToken, err := crypt.DecryptString(c.RefreshToken)
			if err != nil {
				log.Printf("Error decoding mail refresh token: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error decoding mail refresh token -> "+err.Error())
				return
			}

			mailMessages, err := mailGateway.SearchFromDateAndSenders(
				r.Context(),
				c.MailProvider,
				domain.Tokens{AccessToken: decryptedAccessToken, RefreshToken: decryptedRefreshToken, ExpiresAt: c.ExpiresAt},
				time.Now().Add(-time.Hour*24),
				[]string{"nu@nu.com.co", "colpatriaInforma@scotiabankcolpatria.com", "bancodavivienda@davivienda.com"},
			)
			if err != nil {
				log.Printf("Error getting mails from provider "+c.MailProvider+": %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting mails from provider "+c.MailProvider+" -> "+err.Error())
				return
			}

			for i, m := range mailMessages {
				mailMessages[i].UserID = authUser.ID

				fmt.Println("-------------------------------------------------")
				fmt.Printf("mail ID: %s\n", m.ExternalID)
				fmt.Printf("mail From: %s\n", m.From)
				fmt.Printf("mail To: %s\n", m.To)
				fmt.Printf("mail Subject: %s\n", m.Subject)
				fmt.Printf("mail ReceivedAt: %s\n", m.ReceivedAt)
				fmt.Printf("mail Body length: %d\n", len(m.Body))
			}

			err = mailMessageRepo.UpsertMany(r.Context(), mailMessages)
			if err != nil {
				log.Printf("Error persisting mails on storage: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error persisting mails on storage -> "+err.Error())
				return
			}
		}
	}
}
