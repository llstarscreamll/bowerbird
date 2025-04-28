package domain

import "time"

type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	PictureUrl string `json:"picture"`
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type MailCredential struct {
	ID           string
	WalletID     string
	UserID       string
	MailProvider string
	MailAddress  string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type MailMessage struct {
	ID         string
	ExternalID string
	UserID     string
	From       string
	To         string
	Subject    string
	Body       string
	ReceivedAt time.Time
}

type UserWallet struct {
	ID        string    `json:"ID"`
	UserID    string    `json:"userID,omitempty"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joinedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type Transaction struct {
	ID                string    `json:"ID"`
	WalletID          string    `json:"walletID"`
	UserID            string    `json:"userID"`
	UserName          string    `json:"userName"`
	Origin            string    `json:"origin"`
	Reference         string    `json:"reference"`
	Type              string    `json:"type"`
	Amount            float32   `json:"amount"`
	UserDescription   string    `json:"userDescription"`
	SystemDescription string    `json:"systemDescription"`
	ProcessedAt       time.Time `json:"processedAt"`
	CreatedAt         time.Time `json:"createdAt"`
}
