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

// ToDo: state should be stored somewhere and be validated on callback to prevent CSRF attacks
func (m MicrosoftAuthStrategy) GetLoginUrl(redirectUrl string, scopes []string, state string) (string, error) {
	m.config.RedirectURL = redirectUrl
	m.config.Scopes = append(m.config.Scopes, scopes...)
	authCodeUrl := m.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Println("Microsoft Login URL:", authCodeUrl)
	parsedUrl, err := url.Parse(authCodeUrl)
	if err != nil {
		return "", err
	}

	query := parsedUrl.Query()
	query.Del("state")
	parsedUrl.RawQuery = query.Encode()
	fmt.Println("Microsoft Parsed Login URL:", parsedUrl.String())

	return authCodeUrl, nil
}

func (m MicrosoftAuthStrategy) GetTokens(ctx context.Context, authCode, state string) (domain.Tokens, error) {
	t, err := m.config.Exchange(ctx, authCode)
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
	var user domain.User

	headers := make(http.Header)
	headers.Add("Authorization", "Bearer "+accessToken)

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
		Scopes:       []string{"offline_access"},
	}

	return &MicrosoftAuthStrategy{config, ulid}
}
