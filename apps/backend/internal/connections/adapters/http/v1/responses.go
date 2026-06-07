package v1

import "github.com/bowerbird/internal/connections/domain"

type connectionResponse struct {
	ID                   string `json:"id"`
	Provider             string `json:"provider"`
	ProviderAccountEmail string `json:"provider_account_email"`
	Status               string `json:"status"`
	SharingPolicy        string `json:"sharing_policy"`
}

func newConnectionResponse(connection *domain.Connection) connectionResponse {
	return connectionResponse{
		ID:                   connection.ID,
		Provider:             connection.Provider,
		ProviderAccountEmail: connection.ProviderAccountEmail,
		Status:               connection.Status,
		SharingPolicy:        connection.SharingPolicy,
	}
}
