package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"fmt"
	"os"
	"time"

	authInfra "llstarscreamll/bowerbird/internal/auth/infra"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
	commonInfra "llstarscreamll/bowerbird/internal/common/infra"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
)

var mux *http.ServeMux
var server *http.Server

func lambdaHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	r, err := http.NewRequest(req.HTTPMethod, req.Path, strings.NewReader(req.Body))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Internal Server Error",
		}, nil
	}

	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from Go server: %v\n", err)

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Internal Server Error",
		}, nil
	}

	bodyString := string(bodyBytes)

	fmt.Printf("Response status: %s\n", resp.Status)
	fmt.Printf("Response body: %s\n", bodyString)

	return events.APIGatewayProxyResponse{
		StatusCode: resp.StatusCode,
		Body:       bodyString,
	}, nil
}

func main() {
	if mux == nil || server == nil {
		setUpAPIServer()
	}

	lambda.Start(lambdaHandler)
}

func setUpAPIServer() {
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

	mux = http.NewServeMux()

	mux.HandleFunc("GET /v1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data": "Welcome to API V1"}`)
	})

	authInfra.RegisterRoutes(mux, config, ulid, authServerGateway, userRepo, sessionRepo, crypt, mailCredentialRepo, mailGateway, mailMessageRepo, walletRepo, transactionRepo)

	// Enable CORS
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", config.FrontendUrl)
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

	server = &http.Server{
		Addr:           config.ServerPort,
		Handler:        corsHandler(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   20 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
}
