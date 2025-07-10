package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

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
	LastReadAt   time.Time
	CreatedAt    time.Time
}

type MailMessage struct {
	ID          string
	ExternalID  string
	UserID      string
	From        string
	To          string
	Subject     string
	Body        string
	Attachments []MailAttachment
	ReceivedAt  time.Time
}

type MailAttachment struct {
	Name        string
	ContentType string
	Content     string
	Password    string
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
	CategoryID        string    `json:"categoryID"`
	CategorySetterID  string    `json:"categorySetterID"`
	CategoryName      string    `json:"categoryName"`
	CategoryColor     string    `json:"categoryColor"`
	CategoryIcon      string    `json:"categoryIcon"`
	UserName          string    `json:"userName"`
	Origin            string    `json:"origin"`
	Type              string    `json:"type"`
	Amount            float32   `json:"amount"`
	UserDescription   string    `json:"userDescription"`
	SystemDescription string    `json:"systemDescription"`
	UniquenessCount   int       `json:"uniquenessCount"`
	ProcessedAt       time.Time `json:"processedAt"`
	CreatedAt         time.Time `json:"createdAt"`
}

func (t *Transaction) Reference() string {
	desc := strings.ToLower(t.SystemDescription)
	desc = regexp.MustCompile(` \([\w|\d]+\)$`).ReplaceAllString(desc, "")

	if len(desc) > 33 {
		desc = desc[:33]
	}

	date := t.ProcessedAt.In(time.FixedZone("UTC-5", -5*60*60)).Format("20060102")

	return fmt.Sprintf("%s/%s/%s/%f/%d", date, t.Origin, desc, t.Amount, t.UniquenessCount)
}

type Category struct {
	ID          string    `json:"ID"`
	WalletID    string    `json:"walletID"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Icon        string    `json:"icon"`
	Patterns    []string  `json:"patterns"`
	CreatedByID string    `json:"createdByID"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Session struct {
	ID        string    `json:"ID"`
	UserID    string    `json:"userID"`
	ExpiresAt time.Time `json:"expiresAt"`
}
