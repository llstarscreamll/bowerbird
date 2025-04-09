package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	authInfra "llstarscreamll/bowerbird/internal/auth/infra"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
	commonInfra "llstarscreamll/bowerbird/internal/common/infra"
	"llstarscreamll/bowerbird/internal/common/infra/postgresql"
)

var server *http.Server

func lambdaHandler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// Log incoming request for debugging
	reqBytes, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("Error marshalling request: %v\n", err)
	} else {
		fmt.Printf("Received API Gateway V2 request: %s\n", string(reqBytes))
	}

	// Extract path from request
	path := strings.Replace(req.RawPath, "/prod", "", 1)
	if path == "" {
		path = "/"
	}

	// Decode base64 body if necessary
	body := req.Body
	if req.IsBase64Encoded {
		decodedBody, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			fmt.Printf("Error decoding base64 body: %v\n", err)
		}

		if err == nil {
			body = string(decodedBody)
		}
	}

	fmt.Printf("Processing request: Method=%s, Path=%s\n", req.RequestContext.HTTP.Method, path)

	// Create a new HTTP request for our server handler
	r, err := http.NewRequest(req.RequestContext.HTTP.Method, path, strings.NewReader(body))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)

		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Internal Server Error",
		}, nil
	}

	// Add headers from the API Gateway request
	for key, value := range req.Headers {
		r.Header.Set(key, value)
	}

	// Add query string parameters
	if len(req.QueryStringParameters) > 0 {
		q := r.URL.Query()
		for key, value := range req.QueryStringParameters {
			q.Add(key, value)
		}
		r.URL.RawQuery = q.Encode()
	}

	// Add cookies if present
	if cookies := req.Cookies; len(cookies) > 0 {
		for _, cookie := range cookies {
			r.Header.Add("Cookie", cookie)
		}
	}

	w := httptest.NewRecorder()
	server.Handler.ServeHTTP(w, r)

	resp := w.Result()
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body from Go server: %v\n", err)
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Internal Server Error",
		}, nil
	}

	bodyString := string(bodyBytes)

	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response body: %s\n", bodyString)

	// Prepare headers for the response
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Extract cookies for the response
	cookies := make([]string, 0)
	for _, cookie := range resp.Cookies() {
		cookies = append(cookies, cookie.String())
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:      resp.StatusCode,
		Headers:         headers,
		Cookies:         cookies,
		Body:            bodyString,
		IsBase64Encoded: false,
	}, nil
}

func main() {
	if server == nil {
		setUpAPIServer()
	}

	lambda.Start(lambdaHandler)
}

func setUpAPIServer() {
	ctx := context.Background()

	// get app secrets from parameter store
	ps := commonInfra.NewAwsParameterStore(ctx)
	jsonConfig, err := ps.GetParameter(ctx, os.Getenv("PARAMETER_STORE_KEY_NAME"), true)
	if err != nil {
		log.Fatalf("Error getting secrets from parameter store: %v", err)
	}

	var config commonDomain.AppConfig
	err = json.Unmarshal([]byte(jsonConfig), &config)
	if err != nil {
		log.Fatalf("Error un-marshalling json secrets: %v", err)
	}

	db := postgresql.CreatePgxConnectionPool(ctx, config.PostgresDbUrl)
	defer db.Close()

	ulid := commonInfra.OklogULIDGenerator{}
	crypt := commonInfra.NewGoCrypt(config.CryptSecret)
	userRepo := authInfra.NewPgxUserRepository(db)
	sessionRepo := authInfra.NewPgxSessionRepository(db)
	mailMessageRepo := authInfra.NewPgxMailMessageRepository(db)
	mailCredentialRepo := authInfra.NewPgxMailCredentialRepository(db)
	walletRepo := authInfra.NewPgxWalletRepository(db)
	transactionRepo := authInfra.NewPgxTransactionRepository(db)

	googleAuth := *authInfra.NewGoogleAuthStrategy(
		config.GoogleClientID,
		config.GoogleClientSecret,
		config.GoogleOAuthRedirectUrl,
		ulid,
	)
	microsoftAuth := *authInfra.NewMicrosoftAuthStrategy(
		config.MicrosoftClientID,
		config.MicrosoftClientSecret,
		config.MicrosoftOAuthRedirectUrl,
		ulid,
	)
	authServerGateway := authInfra.NewAuthServerGateway(googleAuth, microsoftAuth)

	gMailProvider := authInfra.NewGoogleMailProvider(
		config.GoogleClientID,
		config.GoogleClientSecret,
		config.GoogleOAuthRedirectUrl,
		ulid,
	)
	mailGateway := authInfra.NewMailGateway(gMailProvider)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1", func(w http.ResponseWriter, r *http.Request) {
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

	server = &http.Server{
		Addr:           config.ServerPort,
		Handler:        corsHandler(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   20 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
}
