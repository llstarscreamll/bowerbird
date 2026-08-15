package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/bowerbird/internal/inbox/adapters/provider/gmail"
	"github.com/bowerbird/internal/inbox/adapters/provider/microsoft"
	"github.com/bowerbird/internal/inbox/domain"
)

type BuildClientFunc func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error)

type Factory struct {
	builders map[string]BuildClientFunc
}

func NewFactory() *Factory {
	return &Factory{builders: map[string]BuildClientFunc{}}
}

type DefaultFactoryConfig struct {
	Gmail     gmail.OAuthConfig
	Microsoft microsoft.OAuthConfig
}

func NewDefaultFactory(gmailOAuthConfig gmail.OAuthConfig) *Factory {
	return NewDefaultFactoryWithConfig(DefaultFactoryConfig{Gmail: gmailOAuthConfig})
}

func NewDefaultFactoryWithConfig(cfg DefaultFactoryConfig) *Factory {
	factory := NewFactory()
	if cfg.Gmail.ClientID != "" {
		factory.Register(domain.ProviderGmail, func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error) {
			return gmail.NewOAuthHTTPClient(ctx, cfg.Gmail, credentialsJSON)
		})
	}
	if cfg.Microsoft.ClientID != "" {
		builder := func(ctx context.Context, credentialsJSON []byte) (domain.MailProviderClient, error) {
			return microsoft.NewOAuthHTTPClient(ctx, cfg.Microsoft, credentialsJSON)
		}
		factory.Register(domain.ProviderMicrosoft, builder)
		factory.Register(domain.ProviderOutlook, builder)
		factory.Register(domain.ProviderHotmail, builder)
	}
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
