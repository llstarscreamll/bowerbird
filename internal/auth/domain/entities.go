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
