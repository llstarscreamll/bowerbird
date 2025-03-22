package domain

type AppConfig struct {
	IsProduction              bool   `json:"IS_PRODUCTION"`
	ApiUrl                    string `json:"API_URL"`
	WebUrl                    string `json:"WEB_URL"`
	ServerPort                string `json:"SERVER_PORT"`
	CryptSecret               string `json:"CRYPT_SECRET"`
	GoogleClientID            string `json:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret        string `json:"GOOGLE_CLIENT_SECRET"`
	GoogleOAuthRedirectUrl    string `json:"GOOGLE_OAUTH_REDIRECT_URL"`
	MicrosoftClientID         string `json:"MICROSOFT_CLIENT_ID"`
	MicrosoftClientSecret     string `json:"MICROSOFT_CLIENT_SECRET"`
	MicrosoftOAuthRedirectUrl string `json:"MICROSOFT_OAUTH_REDIRECT_URL"`
	PostgresDbUrl             string `json:"POSTGRES_DATABASE_URL"`
}
