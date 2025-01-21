# Bowerbird App

Expenses tracker application.

## Technologies

- Golang: backend language
- OAuth0 2.0: to know about the auth flow used, read [here](https://cloud.google.com/go/getting-started/authenticate-users-with-iap#external_authentication_with_oauth2) and [here](https://developers.google.com/identity/protocols/oauth2/web-server)
- Typescript: frontend language
- Angular: frontend framework

## Running the project

Execute database migrations:

```bash
brew install golang-migrate
migrate -source file://internal/common/infra/postgresql/migrations -database "postgres://johan:@localhost:5432/bowerbird_test?sslmode=disable" up
```

Start the API:

```bash
CRYPT_SECRET="6cxaEM3EJm6672FEaFTOlA==" \
GOOGLE_CLIENT_ID="google-client-id" \
GOOGLE_CLIENT_SECRET="google-client-secret" \
GOOGLE_OAUTH_REDIRECT_URL="http://localhost:8080/v1/auth/google/callback" \
FRONTEND_URL="http://localhost:4200" \
go run cmd/api/main.go
```
