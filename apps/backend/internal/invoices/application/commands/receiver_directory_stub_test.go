package commands

import (
	"context"

	"github.com/bowerbird/internal/invoices/application/ports"
)

type receiverDirectoryStub struct {
	hasAny bool
	match  bool
}

func matchingReceivers() ports.ReceiverDirectory {
	return receiverDirectoryStub{hasAny: true, match: true}
}

func (s receiverDirectoryStub) HasAny(ctx context.Context) (bool, error) {
	return s.hasAny, nil
}

func (s receiverDirectoryStub) ReceiverMatches(ctx context.Context, taxID string) (bool, error) {
	return s.match, nil
}
