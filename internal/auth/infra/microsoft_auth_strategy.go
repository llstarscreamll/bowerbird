package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftAuthStrategy struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

func (m MicrosoftAuthStrategy) GetLoginUrl(redirectUrl string, scopes []string, state string) (string, error) {
	m.config.RedirectURL = redirectUrl
	m.config.Scopes = append(m.config.Scopes, scopes...)
	challenge := oauth2.S256ChallengeOption(state)

	return m.config.AuthCodeURL(state, oauth2.AccessTypeOffline, challenge), nil
}

func (m MicrosoftAuthStrategy) GetTokens(ctx context.Context, authCode string, challengeVerifier string) (domain.Tokens, error) {
	t, err := m.config.Exchange(ctx, authCode, oauth2.VerifierOption(challengeVerifier))
	if err != nil {
		return domain.Tokens{}, err
	}

	return domain.Tokens{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    t.Expiry,
	}, nil
}

func (m MicrosoftAuthStrategy) GetUserProfile(ctx context.Context, accessToken string) (domain.User, error) {
	var microsoftUser struct {
		FirstName  string `json:"givenName"`
		LastName   string `json:"surname"`
		Email      string `json:"userPrincipalName"`
		PictureUrl string `json:"picture"`
	}

	headers := make(http.Header)
	headers.Add("Authorization", "Bearer "+accessToken)

	r := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "https", Host: "graph.microsoft.com", Path: "/v1.0/me"},
		Header: headers,
	}

	client := http.DefaultClient
	resp, err := client.Do(r)
	if err != nil {
		return domain.User{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.User{}, err
	}

	fmt.Printf("Microsoft user info: %s\n", string(body))

	err = json.Unmarshal(body, &microsoftUser)
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		GivenName:  microsoftUser.FirstName,
		FamilyName: microsoftUser.LastName,
		Email:      microsoftUser.Email,
		PictureUrl: microsoftUser.PictureUrl,
	}, nil
}

func NewMicrosoftAuthStrategy(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *MicrosoftAuthStrategy {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     microsoft.AzureADEndpoint(""),
		Scopes:       []string{},
	}

	return &MicrosoftAuthStrategy{config, ulid}
}
