package infra

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type contextKey string

const userContextKey contextKey = "user"

func RegisterRoutes(mux *http.ServeMux, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authGateway domain.AuthServerGateway, userRepo domain.UserRepository, sessionRepo domain.SessionRepository, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) {
	mux.HandleFunc("GET /v1/auth/google/login", googleLoginHandler(config, authGateway))
	mux.HandleFunc("GET /v1/auth/google/callback", googleLoginCallbackHandler(config, ulid, authGateway, userRepo, sessionRepo))

	mux.HandleFunc("GET /v1/auth/google-mail/login", authMiddleware(gMailLoginHandler("google", config, authGateway), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/google-mail/callback", authMiddleware(mailLoginCallbackHandler("google", config, ulid, authGateway, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /v1/auth/microsoft/login", authMiddleware(outlookLoginHandler("microsoft", config, authGateway), sessionRepo, userRepo))
	mux.HandleFunc("GET /v1/auth/microsoft/callback", authMiddleware(mailLoginCallbackHandler("microsoft", config, ulid, authGateway, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /v1/transactions/sync-from-mail", authMiddleware(syncTransactionsFromEmailHandler(ulid, crypt, authGateway, mailSecretRepo), sessionRepo, userRepo))
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

func syncTransactionsFromEmailHandler(ulid commonDomain.ULIDGenerator, crypt commonDomain.Crypt, authGateway domain.AuthServerGateway, mailSecretRepo domain.MailCredentialRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser := r.Context().Value(userContextKey).(domain.User)
		mailCredentials, err := mailSecretRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting mail credentials from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting mail credentials from storage -> "+err.Error())
			return
		}

		fmt.Println("Credentials from storage:", len(mailCredentials))

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

			// this is for microsoft outlook
			// r, err := http.NewRequest(http.MethodGet, "https://graph.microsoft.com/v1.0/me/messages?$filter=from in ('colpatriaInforma@scotiabankcolpatria.com','nu@nu.com.co')", nil)
			// r.Header.Add("Authorization", "Bearer "+decryptedAccessToken)
			// if err != nil {
			// 	log.Printf("Error building email provider request: %s", err.Error())
			// 	w.WriteHeader(http.StatusInternalServerError)
			// 	fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error building email provider request -> "+err.Error())
			// 	return
			// }

			// response, err := http.DefaultClient.Do(r)
			// if err != nil {
			// 	log.Printf("Error sending request to mail provider: %s", err.Error())
			// 	w.WriteHeader(http.StatusInternalServerError)
			// 	fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error sending request to mail provider -> "+err.Error())
			// 	return
			// }

			config := &oauth2.Config{
				ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
				ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
				RedirectURL:  os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
				Endpoint:     google.Endpoint,
				Scopes:       []string{gmail.GmailReadonlyScope},
			}

			fmt.Println("Encrypted Refresh token: ", c.RefreshToken)
			fmt.Println("Decrypted Refresh token: ", decryptedRefreshToken)

			t := &oauth2.Token{AccessToken: decryptedAccessToken, RefreshToken: decryptedRefreshToken, Expiry: c.ExpiresAt, TokenType: "Bearer"}
			ts := config.TokenSource(r.Context(), t)
			tt, err := ts.Token()
			if err != nil {
				log.Printf("Error building token source: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error building token source -> "+err.Error())
				return
			}

			client := config.Client(r.Context(), tt)
			service, err := gmail.NewService(r.Context(), option.WithHTTPClient(client))
			if err != nil {
				log.Printf("Error building Gmail client: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error building Gmail client -> "+err.Error())
				return
			}

			messagesList, err := service.Users.Messages.List("me").IncludeSpamTrash(true).Q("from:nu@nu.com.co OR from:colpatriaInforma@scotiabankcolpatria.com OR from:bancodavivienda@davivienda.com").Do()
			if err != nil {
				log.Printf("Error building Gmail client: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error building Gmail client -> "+err.Error())
				return
			}

			fmt.Println("Messages count: ", len(messagesList.Messages))
			for i, v := range messagesList.Messages {
				fmt.Printf("Message %s index: %v \n", v.Id, i)

				msg, err := service.Users.Messages.Get("me", v.Id).Format("full").Do()
				if err != nil {
					log.Printf("Error retrieving Gmail message %s: %s", v.Id, err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error retrieving Gmail message "+v.Id+" -> "+err.Error())
					return
				}

				fmt.Println("-------")
				fmt.Println("message ID -> ", msg.Id)

				for _, v := range msg.Payload.Headers {
					if !slices.Contains([]string{"from", "subject", "to", "date"}, strings.ToLower(v.Name)) {
						continue
					}

					fmt.Println(v.Name + " -> " + v.Value)
				}
				fmt.Println("parts -> ", len(msg.Payload.Parts))

				for _, p := range msg.Payload.Parts {
					fmt.Println(p.MimeType + " -> " + p.PartId)
				}
				fmt.Println("-------")

				if i == 0 {
					return
				}

				// fmt.Println("-------")
				// fmt.Println("message data: ", msg.Payload.MimeType)
				// fmt.Println("-------")
				// decodedBody, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
				// if err != nil {
				// 	log.Printf("Error decoding mail body: %s", err.Error())
				// 	w.WriteHeader(http.StatusInternalServerError)
				// 	fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error decoding mail body -> "+err.Error())
				// 	return
				// }
				// fmt.Println(string(decodedBody))
			}
		}
	}
}
