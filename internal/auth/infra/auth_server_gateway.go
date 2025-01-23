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

type GoogleAuthServer struct {
	config *oauth2.Config
	ulid   commonDomain.ULIDGenerator
}

// ToDo: state should be stored somewhere and be validated on callback to prevent CSRF attacks
func (g GoogleAuthServer) GetLoginUrl(provider string, scopes []string) (string, error) {
	return g.config.AuthCodeURL(g.ulid.New(), oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(oauth2.GenerateVerifier())), nil
}

func (g GoogleAuthServer) GetTokens(ctx context.Context, provider string, authCode string) (string, string, time.Time, error) {
	t, err := g.config.Exchange(ctx, authCode, oauth2.VerifierOption(oauth2.GenerateVerifier()))
	if err != nil {
		return "", "", time.Now().Add(time.Hour * -2), err
	}

	return t.AccessToken, t.RefreshToken, time.Now().Add(time.Second * time.Duration(t.ExpiresIn)), nil
}

// ToDo: state should be validated to prevent CSRF attacks
func (g GoogleAuthServer) GetUserProfile(ctx context.Context, provider string, authCode string) (domain.User, error) {
	var user domain.User

	accessToken, _, _, err := g.GetTokens(ctx, provider, authCode)
	if err != nil {
		return user, err
	}

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

func NewGoogleAuthService(clientID, clientSecret, redirectUrl string, ulid commonDomain.ULIDGenerator) *GoogleAuthServer {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	}

	return &GoogleAuthServer{config, ulid}
}
