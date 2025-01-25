package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftAuthStrategy struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

// ToDo: state should be stored somewhere and be validated on callback to prevent CSRF attacks
func (g MicrosoftAuthStrategy) GetLoginUrl(redirectUrl string, scopes []string) (string, error) {
	return g.config.AuthCodeURL(g.ulid.New(), oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(oauth2.GenerateVerifier())), nil
}

func (g MicrosoftAuthStrategy) GetTokens(ctx context.Context, authCode string) (domain.Tokens, error) {
	t, err := g.config.Exchange(ctx, authCode, oauth2.VerifierOption(oauth2.GenerateVerifier()))
	if err != nil {
		return domain.Tokens{}, err
	}

	return domain.Tokens{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Second * time.Duration(t.ExpiresIn)),
	}, nil
}

// ToDo: state should be validated to prevent CSRF attacks
func (g MicrosoftAuthStrategy) GetUserProfile(ctx context.Context, authCode string) (domain.User, error) {
	var user domain.User

	tokens, err := g.GetTokens(ctx, authCode)
	if err != nil {
		return user, err
	}

	headers := make(http.Header)
	headers.Add("Authorization", "Bearer "+tokens.AccessToken)

	r := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "https", Host: "graph.microsoft.com", Path: "oidc/userinfo"},
		Header: headers,
	}

	client := http.DefaultClient
	resp, err := client.Do(r)
	if err != nil {
		return user, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return user, err
	}

	fmt.Printf("Microsoft user info: %s", string(body))

	err = json.Unmarshal(body, &user)
	if err != nil {
		return user, err
	}

	return user, nil
}

func NewMicrosoftAuthStrategy(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *MicrosoftAuthStrategy {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     microsoft.LiveConnectEndpoint,
		Scopes:       []string{},
	}

	return &MicrosoftAuthStrategy{config, ulid}
}
