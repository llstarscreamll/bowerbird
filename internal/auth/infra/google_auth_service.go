package infra

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"llstarscreamll/bowerbird/internal/auth/domain"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthServer struct {
	config   *oauth2.Config
	verifier string
}

func (g GoogleAuthServer) GetLoginUrl() string {
	return g.config.AuthCodeURL("", oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(g.verifier))
}

func (g GoogleAuthServer) GetUserInfo(ctx context.Context, authCode string) (domain.User, error) {
	var user domain.User

	token, err := g.config.Exchange(ctx, authCode, oauth2.VerifierOption(g.verifier))
	if err != nil {
		return user, err
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
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

func NewGoogleAuthService(clientID, clientSecret, redirectUrl string) *GoogleAuthServer {
	c := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	}

	return &GoogleAuthServer{config: c, verifier: oauth2.GenerateVerifier()}
}
