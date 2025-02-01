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
		ServerHost:  os.Getenv("SERVER_HOST"),
		ServerPort:  ":8080",
		FrontendUrl: os.Getenv("FRONTEND_URL"),
	}

	db := postgresql.CreatePgxConnectionPool(ctx, os.Getenv("POSTGRES_DATABASE_URL"))
	defer db.Close()

	ulid := commonInfra.OklogULIDGenerator{}
	crypt := commonInfra.NewGoCrypt(os.Getenv("CRYPT_SECRET"))
	userRepo := authInfra.NewPgxUserRepository(db)
	sessionRepo := authInfra.NewPgxSessionRepository(db)
	mailMessageRepo := authInfra.NewPgxMailMessageRepository(db)
	mailCredentialRepo := authInfra.NewPgxMailCredentialRepository(db)

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
	)
	mailGateway := authInfra.NewMailGateway(gMailProvider)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data": "Welcome to API V1"}`)
	})

	authInfra.RegisterRoutes(mux, config, ulid, authServerGateway, userRepo, sessionRepo, crypt, mailCredentialRepo, mailGateway, mailMessageRepo)

	s := &http.Server{
		Addr:           config.ServerPort,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Fatal(s.ListenAndServe())
}
