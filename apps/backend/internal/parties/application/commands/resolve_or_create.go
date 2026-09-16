package commands

import (
	"context"
	"errors"
	"time"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/bowerbird/internal/platform/id"
)

type IssuerAddress struct {
	Line        string
	City        string
	Department  string
	PostalZone  string
	CountryCode string
	Kind        string
}

type IssuerProfile struct {
	TaxID        string
	Name         string
	SchemeID     string
	TaxpayerKind string
	TaxLevelCode string
	Emails       []string
	Phones       []string
	Addresses    []IssuerAddress
}

type ResolveOrCreateFromIssuerCommand struct {
	repo  ports.PartyRepository
	now   func() time.Time
	newID func() string
}

func NewResolveOrCreateFromIssuerCommand(repo ports.PartyRepository) *ResolveOrCreateFromIssuerCommand {
	if repo == nil {
		panic("party repository is required")
	}
	return &ResolveOrCreateFromIssuerCommand{repo: repo, now: time.Now, newID: id.NewULID}
}

func (cmd *ResolveOrCreateFromIssuerCommand) Execute(ctx context.Context, profile IssuerProfile) (*domain.Party, error) {
	taxID, err := domain.ParseTaxID(profile.TaxID)
	if err != nil {
		if errors.Is(err, domain.ErrMissingTaxID) {
			return nil, nil
		}
		return nil, err
	}

	existing, err := cmd.repo.GetByTaxID(ctx, taxID.String())
	if err != nil {
		return nil, err
	}

	now := cmd.now()
	if existing == nil {
		party := domain.NewProvisionalSupplier(cmd.newID(), taxID, profile.Name, now)
		if err := cmd.applyIssuerProfile(&party, profile, now); err != nil {
			return nil, err
		}
		if err := cmd.repo.Create(ctx, party); err != nil {
			again, getErr := cmd.repo.GetByTaxID(ctx, taxID.String())
			if getErr == nil && again != nil {
				return again, nil
			}
			return nil, err
		}
		return &party, nil
	}

	added, err := cmd.enrichExisting(existing, profile, now)
	if err != nil {
		return nil, err
	}
	hasChannels := len(added.emails)+len(added.phones)+len(added.addresses) > 0
	if added.identity || hasChannels {
		if err := cmd.repo.Update(ctx, *existing); err != nil {
			return nil, err
		}
	}
	if err := cmd.persistAdded(ctx, existing.ID, added); err != nil {
		return nil, err
	}
	return existing, nil
}

type issuerAdds struct {
	identity  bool
	emails    []domain.PartyEmail
	phones    []domain.PartyPhone
	addresses []domain.PartyAddress
}

func (cmd *ResolveOrCreateFromIssuerCommand) applyIssuerProfile(party *domain.Party, profile IssuerProfile, now time.Time) error {
	_, err := cmd.enrichExisting(party, profile, now)
	return err
}

func (cmd *ResolveOrCreateFromIssuerCommand) enrichExisting(party *domain.Party, profile IssuerProfile, now time.Time) (issuerAdds, error) {
	adds := issuerAdds{}
	if party.EnsureSupplierRole(now) {
		adds.identity = true
	}
	if ok, err := fillOptional(party.FillScheme(profile.SchemeID, now)); err != nil {
		return adds, err
	} else if ok {
		adds.identity = true
	}
	if ok, err := fillOptional(party.FillTaxpayerKind(profile.TaxpayerKind, now)); err != nil {
		return adds, err
	} else if ok {
		adds.identity = true
	}
	if party.UnionTaxLevelCodes(domain.ParseTaxLevelCodes(profile.TaxLevelCode), now) {
		adds.identity = true
	}
	for _, raw := range profile.Emails {
		email, err := party.AddEmail(cmd.newID(), raw, domain.SourceInvoice, now)
		if skipChannel(err) {
			continue
		}
		if err != nil {
			return adds, err
		}
		adds.emails = append(adds.emails, *email)
	}
	for _, raw := range profile.Phones {
		phone, err := party.AddPhone(cmd.newID(), raw, domain.SourceInvoice, now)
		if skipChannel(err) {
			continue
		}
		if err != nil {
			return adds, err
		}
		adds.phones = append(adds.phones, *phone)
	}
	for _, addr := range profile.Addresses {
		added, err := party.AddAddress(cmd.newID(), addr.Line, addr.City, addr.Department, addr.PostalZone, addr.CountryCode, addr.Kind, domain.SourceInvoice, now)
		if skipChannel(err) {
			continue
		}
		if err != nil {
			return adds, err
		}
		adds.addresses = append(adds.addresses, *added)
	}
	return adds, nil
}

func (cmd *ResolveOrCreateFromIssuerCommand) persistAdded(ctx context.Context, partyID string, adds issuerAdds) error {
	for _, email := range adds.emails {
		if err := ignoreConflict(cmd.repo.InsertEmail(ctx, partyID, email)); err != nil {
			return err
		}
	}
	for _, phone := range adds.phones {
		if err := ignoreConflict(cmd.repo.InsertPhone(ctx, partyID, phone)); err != nil {
			return err
		}
	}
	for _, addr := range adds.addresses {
		if err := ignoreConflict(cmd.repo.InsertAddress(ctx, partyID, addr)); err != nil {
			return err
		}
	}
	return nil
}

func fillOptional(changed bool, err error) (bool, error) {
	if errors.Is(err, domain.ErrInvalidScheme) || errors.Is(err, domain.ErrInvalidTaxpayerKind) {
		return false, nil
	}
	return changed, err
}

func skipChannel(err error) bool {
	return errors.Is(err, domain.ErrDuplicateChannel) ||
		errors.Is(err, domain.ErrInvalidEmail) ||
		errors.Is(err, domain.ErrMissingPhone) ||
		errors.Is(err, domain.ErrMissingAddress) ||
		errors.Is(err, domain.ErrInvalidAddressKind)
}

func ignoreConflict(err error) error {
	if err == nil {
		return nil
	}
	var appErr *appErrors.AppError
	if errors.As(err, &appErr) && appErr.Code == appErrors.CodeConflict {
		return nil
	}
	return err
}
