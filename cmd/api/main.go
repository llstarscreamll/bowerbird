package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	authInfra "llstarscreamll/bowerbird/internal/auth/infra"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
	commonInfra "llstarscreamll/bowerbird/internal/common/infra"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
)

func main() {
	ctx := context.Background()

	config := commonDomain.AppConfig{
		ApiUrl:     os.Getenv("SERVER_HOST"),
		ServerPort: ":8080",
		WebUrl:     os.Getenv("FRONTEND_URL"),
	}

	db := postgresql.CreatePgxConnectionPool(ctx, os.Getenv("POSTGRES_DATABASE_URL"))
	defer db.Close()

	ulid := commonInfra.OklogULIDGenerator{}
	crypt := commonInfra.NewGoCrypt(os.Getenv("CRYPT_SECRET"))
	userRepo := authInfra.NewPgxUserRepository(db)
	sessionRepo := authInfra.NewPgxSessionRepository(db)
	mailMessageRepo := authInfra.NewPgxMailMessageRepository(db)
	mailCredentialRepo := authInfra.NewPgxMailCredentialRepository(db)
	walletRepo := authInfra.NewPgxWalletRepository(db)
	transactionRepo := authInfra.NewPgxTransactionRepository(db)

	googleAuth := *authInfra.NewGoogleAuthStrategy(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		ulid,
	)
	microsoftAuth := *authInfra.NewMicrosoftAuthStrategy(
		os.Getenv("MICROSOFT_CLIENT_ID"),
		os.Getenv("MICROSOFT_CLIENT_SECRET"),
		os.Getenv("MICROSOFT_OAUTH_REDIRECT_URL"),
		ulid,
	)
	authServerGateway := authInfra.NewAuthServerGateway(googleAuth, microsoftAuth)

	gMailProvider := authInfra.NewGoogleMailProvider(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"),
		ulid,
	)
	mailGateway := authInfra.NewMailGateway(gMailProvider)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data": "Welcome to API V1"}`)
	})

	authInfra.RegisterRoutes(mux, config, ulid, authServerGateway, userRepo, sessionRepo, crypt, mailCredentialRepo, mailGateway, mailMessageRepo, walletRepo, transactionRepo)

	// Enable CORS
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", config.WebUrl)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	s := &http.Server{
		Addr:           config.ServerPort,
		Handler:        corsHandler(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   20 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Fatal(s.ListenAndServe())
}
