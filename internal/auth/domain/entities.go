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
	ID        string    `json:"id"`
	UserID    string    `json:"userID,omitempty"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joinedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type Transaction struct {
	ID                string
	WalletID          string
	UserID            string
	Origin            string
	Reference         string
	Type              string
	Amount            float32
	UserDescription   string
	SystemDescription string
	ProcessedAt       time.Time
	CreatedAt         time.Time
}
