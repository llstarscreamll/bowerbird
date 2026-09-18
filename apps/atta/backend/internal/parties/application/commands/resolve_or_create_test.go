package commands

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/atta/internal/parties/application/ports"
	"github.com/atta/internal/parties/domain"
	appErrors "github.com/atta/internal/platform/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryPartyRepo struct {
	byID    map[string]domain.Party
	byTaxID map[string]string
}

func newMemoryPartyRepo() *memoryPartyRepo {
	return &memoryPartyRepo{byID: map[string]domain.Party{}, byTaxID: map[string]string{}}
}

func (r *memoryPartyRepo) Create(ctx context.Context, party domain.Party) error {
	r.byID[party.ID] = party
	r.byTaxID[party.TaxID] = party.ID
	return nil
}

func (r *memoryPartyRepo) Update(ctx context.Context, party domain.Party) error {
	r.byID[party.ID] = party
	r.byTaxID[party.TaxID] = party.ID
	return nil
}

func (r *memoryPartyRepo) GetByID(ctx context.Context, id string) (*domain.Party, error) {
	p, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (r *memoryPartyRepo) GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error) {
	id, ok := r.byTaxID[taxID]
	if !ok {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *memoryPartyRepo) List(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	out := make([]domain.Party, 0, len(r.byID))
	for _, p := range r.byID {
		out = append(out, p)
	}
	return out, nil
}

func (r *memoryPartyRepo) InsertEmail(ctx context.Context, partyID string, email domain.PartyEmail) error {
	p, ok := r.byID[partyID]
	if !ok {
		return errors.New("party not found")
	}
	for _, existing := range p.Emails {
		if existing.ID == email.ID || existing.Value == email.Value {
			return appErrors.New(appErrors.CodeConflict, "email already exists")
		}
	}
	p.Emails = append(p.Emails, email)
	r.byID[partyID] = p
	return nil
}

func (r *memoryPartyRepo) InsertPhone(ctx context.Context, partyID string, phone domain.PartyPhone) error {
	p, ok := r.byID[partyID]
	if !ok {
		return errors.New("party not found")
	}
	for _, existing := range p.Phones {
		if existing.ID == phone.ID || existing.Normalized == phone.Normalized {
			return appErrors.New(appErrors.CodeConflict, "phone already exists")
		}
	}
	p.Phones = append(p.Phones, phone)
	r.byID[partyID] = p
	return nil
}

func (r *memoryPartyRepo) InsertAddress(ctx context.Context, partyID string, address domain.PartyAddress) error {
	p, ok := r.byID[partyID]
	if !ok {
		return errors.New("party not found")
	}
	for _, existing := range p.Addresses {
		if existing.ID == address.ID || existing.Fingerprint == address.Fingerprint {
			return appErrors.New(appErrors.CodeConflict, "address already exists")
		}
	}
	p.Addresses = append(p.Addresses, address)
	r.byID[partyID] = p
	return nil
}

func (r *memoryPartyRepo) DeleteEmail(ctx context.Context, partyID, emailID string) error {
	p, ok := r.byID[partyID]
	if !ok {
		return errors.New("party not found")
	}
	filtered := make([]domain.PartyEmail, 0, len(p.Emails))
	found := false
	for _, e := range p.Emails {
		if e.ID == emailID {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return appErrors.New(appErrors.CodeNotFound, "channel not found")
	}
	p.Emails = filtered
	r.byID[partyID] = p
	return nil
}

func (r *memoryPartyRepo) DeletePhone(ctx context.Context, partyID, phoneID string) error {
	p, ok := r.byID[partyID]
	if !ok {
		return errors.New("party not found")
	}
	filtered := make([]domain.PartyPhone, 0, len(p.Phones))
	found := false
	for _, e := range p.Phones {
		if e.ID == phoneID {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return appErrors.New(appErrors.CodeNotFound, "channel not found")
	}
	p.Phones = filtered
	r.byID[partyID] = p
	return nil
}

func (r *memoryPartyRepo) DeleteAddress(ctx context.Context, partyID, addressID string) error {
	p, ok := r.byID[partyID]
	if !ok {
		return errors.New("party not found")
	}
	filtered := make([]domain.PartyAddress, 0, len(p.Addresses))
	found := false
	for _, e := range p.Addresses {
		if e.ID == addressID {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return appErrors.New(appErrors.CodeNotFound, "channel not found")
	}
	p.Addresses = filtered
	r.byID[partyID] = p
	return nil
}

func TestResolveOrCreateFromIssuer_CreatesProvisionalSupplier(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewResolveOrCreateFromIssuerCommand(repo)
	cmd.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC) }
	cmd.newID = func() string { return "01PARTY000000000000000001" }

	party, err := cmd.Execute(context.Background(), IssuerProfile{TaxID: "900123", Name: "Proveedor SA"})
	require.NoError(t, err)
	require.NotNil(t, party)
	assert.Equal(t, "900123", party.TaxID)
	assert.Equal(t, "Proveedor SA", party.Name)
	assert.Equal(t, domain.StatusProvisional, party.Status)
	assert.Equal(t, domain.CreationSourceInvoice, party.CreationSource)
	assert.True(t, party.HasRole(domain.RoleSupplier))
}

func TestResolveOrCreateFromIssuer_ReusesExisting(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewResolveOrCreateFromIssuerCommand(repo)
	cmd.newID = func() string { return "01PARTY000000000000000001" }

	first, err := cmd.Execute(context.Background(), IssuerProfile{TaxID: "900123", Name: "Proveedor"})
	require.NoError(t, err)
	cmd.newID = func() string { return "01PARTY000000000000000002" }
	second, err := cmd.Execute(context.Background(), IssuerProfile{TaxID: "900123", Name: "Other Name"})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, repo.byID, 1)
}

func TestResolveOrCreateFromIssuer_EmptyTaxID(t *testing.T) {
	cmd := NewResolveOrCreateFromIssuerCommand(newMemoryPartyRepo())
	party, err := cmd.Execute(context.Background(), IssuerProfile{TaxID: "  ", Name: "Name"})
	require.NoError(t, err)
	assert.Nil(t, party)
}

func TestResolveOrCreateFromIssuer_EnrichesUnionWithoutOverwrite(t *testing.T) {
	repo := newMemoryPartyRepo()
	cmd := NewResolveOrCreateFromIssuerCommand(repo)
	cmd.now = func() time.Time { return time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC) }
	ids := []string{"P1", "E1", "PH1", "A1", "E2"}
	i := 0
	cmd.newID = func() string {
		id := ids[i]
		if i < len(ids)-1 {
			i++
		}
		return id
	}

	first, err := cmd.Execute(context.Background(), IssuerProfile{
		TaxID: "900277370", Name: "I Shop", SchemeID: "31", TaxpayerKind: "1",
		TaxLevelCode: "O-13;O-15",
		Emails:       []string{"dte_9002773704@dte.paperless.com.co"},
		Phones:       []string{"(1) 3289133"},
		Addresses:    []IssuerAddress{{Line: "AV 1", City: "TUNJA", Department: "BOYACÁ", CountryCode: "CO", Kind: domain.AddressKindPhysical}},
	})
	require.NoError(t, err)
	assert.Equal(t, "31", first.SchemeID)
	assert.Equal(t, "1", first.TaxpayerKind)
	assert.Equal(t, []string{"O-13", "O-15"}, first.TaxLevelCodes)
	require.Len(t, first.Emails, 1)
	assert.Equal(t, domain.EmailKindTaxMailbox, first.Emails[0].Kind)
	assert.Equal(t, domain.SourceInvoice, first.Emails[0].Source)

	second, err := cmd.Execute(context.Background(), IssuerProfile{
		TaxID: "900277370", Name: "Other Name", SchemeID: "13", TaxpayerKind: "2",
		TaxLevelCode: "O-15;O-23",
		Emails:       []string{"dte_9002773704@dte.paperless.com.co", "ops@ishop.example"},
		Phones:       []string{"(1) 3289133"},
		Addresses:    []IssuerAddress{{Line: "AV 1", City: "TUNJA", Department: "BOYACÁ", CountryCode: "CO", Kind: domain.AddressKindPhysical}},
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "I Shop", second.Name)
	assert.Equal(t, "31", second.SchemeID)
	assert.Equal(t, "1", second.TaxpayerKind)
	assert.Equal(t, []string{"O-13", "O-15", "O-23"}, second.TaxLevelCodes)
	require.Len(t, second.Emails, 2)
	assert.Equal(t, "ops@ishop.example", second.Emails[1].Value)
}
