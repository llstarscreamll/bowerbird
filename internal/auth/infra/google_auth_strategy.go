package infra

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"llstarscreamll/bowerbird/internal/auth/domain"
	commonDomain "llstarscreamll/bowerbird/internal/common/domain"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthStrategy struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

// ToDo: verifier code should be stored in a database 'cause must match on login, callback and requesting tokens
var foo string = oauth2.GenerateVerifier()

// ToDo: state should be stored somewhere and be validated on callback to prevent CSRF attacks
func (g GoogleAuthStrategy) GetLoginUrl(redirectUrl string, scopes []string) (string, error) {
	g.config.RedirectURL = redirectUrl
	g.config.Scopes = append(g.config.Scopes, scopes...)
	return g.config.AuthCodeURL(g.ulid.New(), oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(foo)), nil
}

func (g GoogleAuthStrategy) GetTokens(ctx context.Context, authCode string) (domain.Tokens, error) {
	t, err := g.config.Exchange(ctx, authCode, oauth2.VerifierOption(foo))
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
func (g GoogleAuthStrategy) GetUserProfile(ctx context.Context, authCode string) (domain.User, error) {
	var user domain.User

	tokens, err := g.GetTokens(ctx, authCode)
	if err != nil {
		return user, err
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + tokens.AccessToken)
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
