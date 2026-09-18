package ports

import (
	"context"

	"github.com/atta/internal/parties/domain"
)

type PartyRepository interface {
	Create(ctx context.Context, party domain.Party) error
	Update(ctx context.Context, party domain.Party) error
	GetByID(ctx context.Context, id string) (*domain.Party, error)
	GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error)
	List(ctx context.Context, filter ListFilter) ([]domain.Party, error)
	InsertEmail(ctx context.Context, partyID string, email domain.PartyEmail) error
	InsertPhone(ctx context.Context, partyID string, phone domain.PartyPhone) error
	InsertAddress(ctx context.Context, partyID string, address domain.PartyAddress) error
	DeleteEmail(ctx context.Context, partyID, emailID string) error
	DeletePhone(ctx context.Context, partyID, phoneID string) error
	DeleteAddress(ctx context.Context, partyID, addressID string) error
}

type ListFilter struct {
	Role           string
	Search         string
	CreationSource string
}
