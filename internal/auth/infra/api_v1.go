package infra

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
)

type contextKey string

const userContextKey contextKey = "user"

func RegisterRoutes(
	mux *http.ServeMux,
	config commonDomain.AppConfig,
	ulid commonDomain.ULIDGenerator,
	authGateway domain.AuthServerGateway,
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	crypt commonDomain.Crypt,
	mailSecretRepo domain.MailCredentialRepository,
	mailGateway domain.MailGateway,
	mailMessageRepo domain.MailMessageRepository,
	walletRepo domain.WalletRepository,
	transactionRepo domain.TransactionRepository,
	categoryRepo domain.CategoryRepository,
) {
	mux.HandleFunc("GET /api/v1/auth/user", authMiddleware(getUserProfileHandler(), sessionRepo, userRepo))
	mux.HandleFunc("GET /api/v1/auth/google/login", googleLoginHandler(config, ulid, sessionRepo, authGateway))
	mux.HandleFunc("GET /api/v1/auth/google/callback", googleLoginCallbackHandler(config, ulid, authGateway, userRepo, sessionRepo, walletRepo))

	mux.HandleFunc("GET /api/v1/auth/google-mail/login", authMiddleware(gMailLoginHandler("google", config, ulid, sessionRepo, authGateway), sessionRepo, userRepo))
	mux.HandleFunc("GET /api/v1/auth/google-mail/callback", authMiddleware(mailLoginCallbackHandler("google", config, ulid, sessionRepo, authGateway, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /api/v1/auth/microsoft/login", authMiddleware(outlookLoginHandler("microsoft", config, ulid, authGateway, sessionRepo), sessionRepo, userRepo))
	mux.HandleFunc("GET /api/v1/auth/microsoft/callback", authMiddleware(mailLoginCallbackHandler("microsoft", config, ulid, sessionRepo, authGateway, crypt, mailSecretRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /api/v1/wallets", authMiddleware(searchWalletsHandler(walletRepo), sessionRepo, userRepo))
	mux.HandleFunc("GET /api/v1/wallets/{walletID}/transactions", authMiddleware(searchWalletTransactionsHandler(walletRepo, transactionRepo), sessionRepo, userRepo))
	mux.HandleFunc("POST /api/v1/wallets/{walletID}/transactions/sync-from-mail", authMiddleware(syncTransactionsFromEmailHandler(ulid, crypt, mailSecretRepo, mailGateway, mailMessageRepo, walletRepo, transactionRepo), sessionRepo, userRepo))
	mux.HandleFunc("GET /api/v1/wallets/{walletID}/transactions/{transactionID}", authMiddleware(getTransactionHandler(walletRepo, transactionRepo), sessionRepo, userRepo))

	mux.HandleFunc("GET /api/v1/wallets/{walletID}/categories", authMiddleware(searchWalletCategoriesHandler(walletRepo, categoryRepo), sessionRepo, userRepo))
}

func getUserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser := r.Context().Value(userContextKey).(domain.User)
		fmt.Fprintf(w, `{"data":{"id":%q,"name":%q,"email":%q,"pictureUrl":%q}}`, authUser.ID, authUser.Name, authUser.Email, authUser.PictureUrl)
	}
}

// redirects the user to the Google login page
func googleLoginHandler(config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, sessionRepo domain.SessionRepository, authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionExpirationDate := time.Now().Add(time.Minute * 10)
		stateID, err := ulid.NewFromDate(sessionExpirationDate)
		if err != nil {
			log.Printf("Error generating state: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error generating state","detail":%q}]}`, err.Error())
			return
		}

		stateID = "googleOAuth-" + stateID

		err = sessionRepo.Save(r.Context(), stateID, "ABC-123", sessionExpirationDate)
		if err != nil {
			log.Printf("Error storing state: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error storing state","detail":%q}]}`, err.Error())
			return
		}

		url, err := authServer.GetLoginUrl("google", config.ApiUrl+"/api/v1/auth/google/callback", []string{}, stateID)
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
func googleLoginCallbackHandler(config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServerGateway, userRepo domain.UserRepository, sessionRepo domain.SessionRepository, walletRepo domain.WalletRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := strings.Trim(r.URL.Query().Get("state"), " ")
		if state == "" {
			log.Printf("Error getting state from query params, state is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"state is empty"}]}`)
			return
		}

		userID, err := sessionRepo.GetByID(r.Context(), state)
		if err != nil {
			log.Printf("Error getting state from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error getting state from storage","detail":%q}]}`, err.Error())
			return
		}

		if userID == "" || userID != "ABC-123" {
			log.Printf("State was not found in session storage or is mismatched with auth user ID: " + state)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"State miss match"}]}`)
			return
		}

		err = sessionRepo.Delete(r.Context(), state)
		if err != nil {
			log.Printf("Error cleaning state from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error cleaning state from storage -> "+err.Error())
			return
		}

		authCode := strings.Trim(r.URL.Query().Get("code"), " ")
		if authCode == "" {
			log.Printf("Error getting code from query params, code is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"code is empty"}]}`)
			return
		}

		tokens, err := authServer.GetTokens(r.Context(), "google", authCode, state)
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

		// if the user does not exists, then create the default wallet to him
		if id == user.ID {
			err := walletRepo.Create(r.Context(), domain.UserWallet{ID: ulid.New(), UserID: id, Name: "My wallet", Role: "owner", JoinedAt: time.Now(), CreatedAt: time.Now()})

			if err != nil {
				log.Printf("Error creating default wallet for user: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error creating default wallet for user","detail":%q}]}`, err.Error())
				return
			}
		}

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
			Secure:   config.IsProduction,
		})

		http.Redirect(w, r, config.WebUrl+"/dashboard", http.StatusFound)
	}
}

// redirects user to Google login page and request access to *read* Gmail
func gMailLoginHandler(provider string, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, sessionRepo domain.SessionRepository, authServer domain.AuthServerGateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletID := strings.Trim(r.URL.Query().Get("wallet_id"), " ")
		if walletID == "" {
			log.Printf("Error getting walletID from query params, walletID is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"Wallet ID is empty"}]}`)
			return
		}

		sessionExpirationDate := time.Now().Add(time.Minute * 10)
		state, err := ulid.NewFromDate(sessionExpirationDate)
		if err != nil {
			log.Printf("Error generating state: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error generating state","detail":%q}]}`, err.Error())
			return
		}

		authUser := r.Context().Value(userContextKey).(domain.User)

		state = state + "-" + walletID
		err = sessionRepo.Save(r.Context(), state, authUser.ID, sessionExpirationDate)
		if err != nil {
			log.Printf("Error storing state: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error storing state","detail":%q}]}`, err.Error())
			return
		}

		redirectUrl, err := authServer.GetLoginUrl(provider, config.ApiUrl+"/api/v1/auth/google-mail/callback", []string{"https://www.googleapis.com/auth/gmail.readonly"}, state)
		if err != nil {
			log.Printf("Error getting auth server login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting auth server login url -> "+err.Error())
			return
		}

		parsedUrl, err := url.Parse(redirectUrl)
		if err != nil {
			log.Printf("Error parsing login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error parsing login url -> "+err.Error())
			return
		}

		query := parsedUrl.Query()
		query.Set("prompt", "consent")
		parsedUrl.RawQuery = query.Encode()

		http.Redirect(w, r, parsedUrl.String(), http.StatusFound)
	}
}

// redirects user to Microsoft login page and request access to *read* mail
func outlookLoginHandler(provider string, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, authServer domain.AuthServerGateway, sessionRepo domain.SessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletID := strings.Trim(r.URL.Query().Get("wallet_id"), " ")
		if walletID == "" {
			log.Printf("Error getting walletID from query params, walletID is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"Wallet ID is empty"}]}`)
			return
		}

		sessionExpirationDate := time.Now().Add(time.Minute * 10)
		state, err := ulid.NewFromDate(sessionExpirationDate)
		if err != nil {
			log.Printf("Error generating state: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error generating state","detail":%q}]}`, err.Error())
			return
		}

		authUser := r.Context().Value(userContextKey).(domain.User)

		state = state + "-" + walletID
		err = sessionRepo.Save(r.Context(), state, authUser.ID, sessionExpirationDate)
		if err != nil {
			log.Printf("Error storing state: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error storing state","detail":%q}]}`, err.Error())
			return
		}

		fmt.Println("Outlook OAuth State: ", state)

		redirectUrl, err := authServer.GetLoginUrl(provider, config.ApiUrl+"/api/v1/auth/microsoft/callback", []string{"user.readbasic.all", "mail.read", "user.read", "openid", "profile", "email", "offline_access"}, state)
		if err != nil {
			log.Printf("Error getting auth server login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting auth server login url -> "+err.Error())
			return
		}

		parsedUrl, err := url.Parse(redirectUrl)
		if err != nil {
			log.Printf("Error parsing login url: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error parsing login url -> "+err.Error())
			return
		}

		query := parsedUrl.Query()
		query.Set("prompt", "consent")
		parsedUrl.RawQuery = query.Encode()

		http.Redirect(w, r, parsedUrl.String(), http.StatusFound)
	}
}

func mailLoginCallbackHandler(provider string, config commonDomain.AppConfig, ulid commonDomain.ULIDGenerator, sessionRepo domain.SessionRepository, authServer domain.AuthServerGateway, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser := r.Context().Value(userContextKey).(domain.User)
		state := strings.Trim(r.URL.Query().Get("state"), " ")
		if state == "" {
			log.Printf("Error getting state from query params, state is empty")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"state is empty"}]}`)
			return
		}

		userID, err := sessionRepo.GetByID(r.Context(), state)
		if err != nil {
			log.Printf("Error getting state from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Error getting state from storage","detail":%q}]}`, err.Error())
			return
		}

		if userID == "" || userID != authUser.ID {
			log.Printf("State was not found in session storage or is mismatched with auth user ID: " + state)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":"State miss match"}]}`)
			return
		}

		err = sessionRepo.Delete(r.Context(), state)
		if err != nil {
			log.Printf("Error cleaning state from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error cleaning state from storage -> "+err.Error())
			return
		}

		walletID := strings.Split(state, "-")[1]
		code := strings.Trim(r.URL.Query().Get("code"), " ")
		tokens, err := authServer.GetTokens(r.Context(), provider, code, state)
		if err != nil {
			log.Printf("Error getting tokens from auth server: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting tokens from auth server -> "+err.Error())
			return
		}

		fmt.Printf("%s provider tokens: %+v \n", provider, tokens)

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

		userMailProfile, err := authServer.GetUserProfile(r.Context(), provider, tokens.AccessToken)
		if err != nil {
			log.Printf("Error getting user mail profile: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting user mail profile -> "+err.Error())
			return
		}

		err = mailSecretRepo.Save(r.Context(), ulid.New(), authUser.ID, walletID, provider, userMailProfile.Email, encryptedAccessToken, encryptedRefreshToken, tokens.ExpiresAt)
		if err != nil {
			log.Printf("Error writing tokens in storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error writing tokens in storage -> "+err.Error())
			return
		}

		http.Redirect(w, r, config.WebUrl+"/dashboard", http.StatusFound)
	}
}

func syncTransactionsFromEmailHandler(ulid commonDomain.ULIDGenerator, crypt commonDomain.Crypt, mailSecretRepo domain.MailCredentialRepository, mailGateway domain.MailGateway, mailMessageRepo domain.MailMessageRepository, walletRepo domain.WalletRepository, transactionRepo domain.TransactionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletID := r.PathValue("walletID")
		if walletID == "" {
			log.Printf("Error getting walletID from path params")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":%q}]}`, "Wallet ID is not valid")
			return
		}

		authUser := r.Context().Value(userContextKey).(domain.User)
		userWallets, err := walletRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting wallets from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting wallets from storage -> "+err.Error())
		}

		if !slices.ContainsFunc(userWallets, func(w domain.UserWallet) bool {
			return w.ID == walletID
		}) {
			log.Printf("Error wallet does not belong to user")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `{"errors":[{"status":"403","title":"Forbidden","detail":%q}]}`, "Wallet does not belong to user")
		}

		mailCredentials, err := mailSecretRepo.FindByWalletID(r.Context(), walletID)
		if err != nil {
			log.Printf("Error getting mail credentials from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting mail credentials from storage -> "+err.Error())
			return
		}

		for _, c := range mailCredentials {
			if c.MailProvider != "microsoft" {
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

			startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
			mailMessages, err := mailGateway.SearchFromDateAndSenders(
				r.Context(),
				c.MailProvider,
				domain.Tokens{AccessToken: decryptedAccessToken, RefreshToken: decryptedRefreshToken, ExpiresAt: c.ExpiresAt},
				startOfMonth,
				[]string{"nu@nu.com.co"},
			)

			if err != nil {
				log.Printf("Error getting mails from provider "+c.MailProvider+": %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting mails from provider "+c.MailProvider+" -> "+err.Error())
				return
			}

			for i := range mailMessages {
				mailMessages[i].UserID = c.UserID
			}

			err = mailMessageRepo.UpsertMany(r.Context(), mailMessages)

			if err != nil {
				log.Printf("Error persisting mails on storage: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error persisting mails on storage -> "+err.Error())
				return
			}

			transactions := make([]domain.Transaction, 0, len(mailMessages))
			for _, m := range mailMessages {
				var parserStrategy domain.EmailParserStrategy
				if strings.Contains(m.From, "nu@nu.com.co") {
					parserStrategy = &NuBankEmailParserStrategy{}
				}

				t := parserStrategy.Parse(m)
				transactions = append(transactions, t...)
			}

			for i := range transactions {
				transactions[i].ID = ulid.New()
				transactions[i].UserID = c.UserID
				transactions[i].WalletID = c.WalletID
				transactions[i].CreatedAt = time.Now()
			}

			err = transactionRepo.UpsertMany(r.Context(), transactions)

			if err != nil {
				log.Printf("Error persisting transactions on storage: %s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error persisting transactions on storage -> "+err.Error())
				return
			}

			fmt.Fprintf(w, `{"data":"ok"}`)
		}
	}
}

func searchWalletsHandler(walletRepo domain.WalletRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authUser := r.Context().Value(userContextKey).(domain.User)
		wallets, err := walletRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting wallets from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting wallets from storage -> "+err.Error())
			return
		}

		walletsJSON, err := json.Marshal(wallets)
		if err != nil {
			log.Printf("Error encoding wallets to JSON: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":"Error encoding wallets to JSON"}]}`)
			return
		}

		fmt.Fprintf(w, `{"data":%s}`, walletsJSON)
	}
}

func searchWalletTransactionsHandler(walletRepo domain.WalletRepository, transactionRepo domain.TransactionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletID := r.PathValue("walletID")
		if walletID == "" {
			log.Printf("Error getting walletID from path params")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":%q}]}`, "Wallet ID is not valid")
			return
		}

		authUser := r.Context().Value(userContextKey).(domain.User)
		userWallets, err := walletRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting wallets from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting wallets from storage -> "+err.Error())
			return
		}

		walletBelongsToUser := slices.ContainsFunc(userWallets, func(w domain.UserWallet) bool {
			return w.ID == walletID
		})

		if !walletBelongsToUser {
			log.Printf("Error wallet does not belong to user")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `{"errors":[{"status":"403","title":"Forbidden","detail":%q}]}`, "Wallet does not belong to user")
			return
		}

		transactions, err := transactionRepo.FindByWalletID(r.Context(), walletID)
		if err != nil {
			log.Printf("Error getting wallets from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting wallets from storage -> "+err.Error())
			return
		}

		transactionsJSON, err := json.Marshal(transactions)
		if err != nil {
			log.Printf("Error encoding transactions to JSON: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":"Error encoding transactions to JSON"}]}`)
			return
		}

		fmt.Fprintf(w, `{"data":%s}`, transactionsJSON)
	}
}

func getTransactionHandler(walletRepo domain.WalletRepository, transactionRepo domain.TransactionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletID := r.PathValue("walletID")
		transactionID := r.PathValue("transactionID")

		authUser := r.Context().Value(userContextKey).(domain.User)
		userWallets, err := walletRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting wallets from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting wallets from storage -> "+err.Error())
			return
		}

		walletBelongsToUser := slices.ContainsFunc(userWallets, func(w domain.UserWallet) bool {
			return w.ID == walletID
		})

		if !walletBelongsToUser {
			log.Printf("Error wallet does not belong to user")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `{"errors":[{"status":"403","title":"Forbidden","detail":%q}]}`, "Wallet does not belong to user")
			return
		}

		transaction, err := transactionRepo.GetByWalletIDAndID(r.Context(), walletID, transactionID)
		if err != nil {
			log.Printf("Error getting transaction from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting transaction from storage -> "+err.Error())
			return
		}

		if transaction.ID == "" {
			log.Printf("Transaction not found")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"errors":[{"status":"404","title":"Not found","detail":%q}]}`, "Transaction not found")
			return
		}

		transactionJSON, err := json.Marshal(transaction)
		if err != nil {
			log.Printf("Error encoding transaction to JSON: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":"Error encoding transaction to JSON"}]}`)
			return
		}

		fmt.Fprintf(w, `{"data":%s}`, transactionJSON)
	}
}

func searchWalletCategoriesHandler(walletRepo domain.WalletRepository, categoryRepo domain.CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletID := r.PathValue("walletID")
		if walletID == "" {
			log.Printf("Error getting walletID from path params")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"errors":[{"status":"400","title":"Bad request","detail":%q}]}`, "Wallet ID is not valid")
			return
		}

		authUser := r.Context().Value(userContextKey).(domain.User)
		userWallets, err := walletRepo.FindByUserID(r.Context(), authUser.ID)
		if err != nil {
			log.Printf("Error getting wallets from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}

		walletBelongsToUser := slices.ContainsFunc(userWallets, func(w domain.UserWallet) bool {
			return w.ID == walletID
		})

		if !walletBelongsToUser {
			log.Printf("Error wallet does not belong to user")
		}

		categories, err := categoryRepo.FindByWalletID(r.Context(), walletID)
		if err != nil {
			log.Printf("Error getting categories from storage: %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"errors":[{"status":"500","title":"Internal server error","detail":%q}]}`, "Error getting categories from storage -> "+err.Error())
			return
		}

		categoriesJSON, err := json.Marshal(categories)
		if err != nil {
			log.Printf("Error encoding categories to JSON: %s", err.Error())
		}

		fmt.Fprintf(w, `{"data":%s}`, categoriesJSON)
	}
}
