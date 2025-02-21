package domain

type AppConfig struct {
	IsProduction bool
	ServerHost   string
	ServerPort   string
	FrontendUrl  string
}
