package testdata

import (
	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"
)

var Config = commonDomain.AppConfig{
	ServerHost:  "http://localhost:8080",
	ServerPort:  ":8080",
	FrontendUrl: "http://localhost:4200",
}

var TestUser = domain.User{
	ID:         "01JGCZXZEC00000000000000U1",
	Email:      "john@doe.com",
	Name:       "John Doe",
	GivenName:  "John",
	FamilyName: "Doe",
	PictureUrl: "https://some-google.com/picture.jpg",
}
