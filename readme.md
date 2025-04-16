# Bowerbird App

Expenses tracker application.

## Technologies

- Golang: backend language
- OAuth0 2.0: to know about the auth flow used, read [here](https://cloud.google.com/go/getting-started/authenticate-users-with-iap#external_authentication_with_oauth2) and [here](https://developers.google.com/identity/protocols/oauth2/web-server)
- Typescript: frontend language
- Angular: frontend framework

## Deploy to production for the first time

These are some topics to have in mind to deploy this project in a production environment:

- Setup a GCP project.
- Setup the Google Login Concent page and get app client ID and secret from GCP.
- Enable the Gmail API on your GCP project.
- Create an Azure account
- Create and setup outlook read only messages for an Azure Entra application and get the client ID and secret.

Check the next links to remove permissions from identity providers platforms:

- https://account.live.com/consent/Manage

## Deploy

```bash
BOWERBIRD=true \
AWS_ACCESS_KEY_ID=some-access \
AWS_SECRET_ACCESS_KEY=some-secret \
AWS_DEFAULT_REGION=us-east-1 \
make deploy
```

## Running the project locally

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
MICROSOFT_CLIENT_ID="microsoft-client-id" \
MICROSOFT_CLIENT_SECRET="microsoft-client-secret" \
MICROSOFT_OAUTH_REDIRECT_URL="http://localhost:8080/v1/auth/microsoft/callback" \
POSTGRES_DATABASE_URL="postgres://user:@localhost:5432/bowerbird_local?sslmode=disable" \
SERVER_HOST="http://localhost:8080" \
FRONTEND_URL="http://localhost:4200" \
go run cmd/api/main.go
```

Start the SPA:

```bash
cd static/web-app
npm run start
```