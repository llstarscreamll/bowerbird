package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/bowerbird/internal/inbox/adapters/provider/gmail"
	"github.com/bowerbird/internal/inbox/domain"
)

type BuildClientFunc func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error)

type Factory struct {
	builders map[string]BuildClientFunc
}

func NewFactory() *Factory {
	return &Factory{builders: map[string]BuildClientFunc{}}
}

func NewDefaultFactory(gmailOAuthConfig gmail.OAuthConfig) *Factory {
	factory := NewFactory()
	factory.Register(domain.ProviderGmail, func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error) {
		return gmail.NewOAuthHTTPClient(ctx, gmailOAuthConfig, credentialsJSON)
	})
	return factory
}

func (f *Factory) Register(provider string, builder BuildClientFunc) {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	f.builders[normalizedProvider] = builder
}

func (f *Factory) Build(ctx context.Context, provider string, credentialsJSON []byte) (domain.MailProviderClient, error) {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	builder, ok := f.builders[normalizedProvider]
	if !ok {
		return nil, fmt.Errorf("mail provider %q is not supported", provider)
	}

	client, err := builder(ctx, credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("build provider client for %s: %w", provider, err)
	}

	return client, nil
}
