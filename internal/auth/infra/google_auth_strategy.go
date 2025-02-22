package infra

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthStrategy struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

func (g GoogleAuthStrategy) GetLoginUrl(redirectUrl string, scopes []string, state string) (string, error) {
	g.config.RedirectURL = redirectUrl
	g.config.Scopes = append(g.config.Scopes, scopes...)
	challenge := oauth2.S256ChallengeOption(state)

	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline, challenge), nil
}

func (g GoogleAuthStrategy) GetTokens(ctx context.Context, authCode string, challengeVerifier string) (domain.Tokens, error) {
	t, err := g.config.Exchange(ctx, authCode, oauth2.VerifierOption(challengeVerifier))
	if err != nil {
		return domain.Tokens{}, err
	}

	return domain.Tokens{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		ExpiresAt:    t.Expiry,
	}, nil
}

// ToDo: state should be validated to prevent CSRF attacks
func (g GoogleAuthStrategy) GetUserProfile(ctx context.Context, accessToken string) (domain.User, error) {
	var user domain.User

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return user, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return user, err
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		return user, err
	}

	return user, nil
}

func NewGoogleAuthStrategy(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *GoogleAuthStrategy {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	}

	return &GoogleAuthStrategy{config, ulid}
}
