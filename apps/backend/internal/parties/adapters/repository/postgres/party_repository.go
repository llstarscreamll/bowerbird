package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bowerbird/internal/parties/application/ports"
	"github.com/bowerbird/internal/parties/domain"
	"github.com/bowerbird/internal/platform/database"
	appErrors "github.com/bowerbird/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PartyRepository struct {
	registry *database.Registry
}

func NewPartyRepository(registry *database.Registry) *PartyRepository {
	return &PartyRepository{registry: registry}
}

var _ ports.PartyRepository = (*PartyRepository)(nil)

const partySelect = `
	SELECT id, COALESCE(tax_id, ''), name, roles, status, creation_source,
		COALESCE(scheme_id, ''), COALESCE(taxpayer_kind, ''), tax_level_codes,
		created_at, updated_at
	FROM parties
`

func (r *PartyRepository) Create(ctx context.Context, party domain.Party) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create party: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO parties (id, tax_id, name, roles, status, creation_source, scheme_id, taxpayer_kind, tax_level_codes, created_at, updated_at)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9, $10, $11)
	`, party.ID, party.TaxID, party.Name, party.Roles, party.Status, party.CreationSource, party.SchemeID, party.TaxpayerKind, party.TaxLevelCodes, party.CreatedAt, party.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a party with this tax id already exists")
		}
		return fmt.Errorf("create party: %w", err)
	}
	if err := insertPartyChannels(ctx, tx, party); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create party: %w", err)
	}
	return nil
}

func (r *PartyRepository) Update(ctx context.Context, party domain.Party) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}

	tag, err := pool.Exec(ctx, `
		UPDATE parties
		SET tax_id = NULLIF($2, ''), name = $3, roles = $4, status = $5,
			scheme_id = NULLIF($6, ''), taxpayer_kind = NULLIF($7, ''), tax_level_codes = $8, updated_at = $9
		WHERE id = $1
	`, party.ID, party.TaxID, party.Name, party.Roles, party.Status, party.SchemeID, party.TaxpayerKind, party.TaxLevelCodes, party.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return appErrors.New(appErrors.CodeConflict, "a party with this tax id already exists")
		}
		return fmt.Errorf("update party: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "party not found")
	}
	return nil
}

func (r *PartyRepository) GetByID(ctx context.Context, id string) (*domain.Party, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	party, err := scanParty(pool.QueryRow(ctx, partySelect+` WHERE id = $1`, id))
	if err != nil || party == nil {
		return party, err
	}
	if err := r.loadChannels(ctx, party); err != nil {
		return nil, err
	}
	return party, nil
}

func (r *PartyRepository) GetByTaxID(ctx context.Context, taxID string) (*domain.Party, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}
	party, err := scanParty(pool.QueryRow(ctx, partySelect+` WHERE tax_id = $1`, taxID))
	if err != nil || party == nil {
		return party, err
	}
	if err := r.loadChannels(ctx, party); err != nil {
		return nil, err
	}
	return party, nil
}

func (r *PartyRepository) List(ctx context.Context, filter ports.ListFilter) ([]domain.Party, error) {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("get tenant db pool: %w", err)
	}

	query := partySelect + ` WHERE 1=1`
	args := []any{}
	argN := 1
	if role := strings.TrimSpace(filter.Role); role != "" {
		query += fmt.Sprintf(` AND $%d = ANY(roles)`, argN)
		args = append(args, role)
		argN++
	}
	if source := strings.TrimSpace(filter.CreationSource); source != "" {
		query += fmt.Sprintf(` AND creation_source = $%d`, argN)
		args = append(args, source)
		argN++
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		query += fmt.Sprintf(` AND (name ILIKE $%d OR COALESCE(tax_id, '') ILIKE $%d)`, argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}
	query += ` ORDER BY name ASC`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list parties: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Party, 0)
	for rows.Next() {
		party, err := scanParty(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *party)
	}
	return out, rows.Err()
}

func (r *PartyRepository) InsertEmail(ctx context.Context, partyID string, email domain.PartyEmail) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO party_emails (id, party_id, value, kind, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, email.ID, partyID, email.Value, email.Kind, email.Source, email.CreatedAt)
	if isUniqueViolation(err) {
		return appErrors.New(appErrors.CodeConflict, "email already exists")
	}
	if err != nil {
		return fmt.Errorf("insert party email: %w", err)
	}
	return nil
}

func (r *PartyRepository) InsertPhone(ctx context.Context, partyID string, phone domain.PartyPhone) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO party_phones (id, party_id, value, normalized, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, phone.ID, partyID, phone.Value, phone.Normalized, phone.Source, phone.CreatedAt)
	if isUniqueViolation(err) {
		return appErrors.New(appErrors.CodeConflict, "phone already exists")
	}
	if err != nil {
		return fmt.Errorf("insert party phone: %w", err)
	}
	return nil
}

func (r *PartyRepository) InsertAddress(ctx context.Context, partyID string, address domain.PartyAddress) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO party_addresses (id, party_id, line, city, department, postal_zone, country_code, kind, fingerprint, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, address.ID, partyID, address.Line, address.City, address.Department, address.PostalZone, address.CountryCode, address.Kind, address.Fingerprint, address.Source, address.CreatedAt)
	if isUniqueViolation(err) {
		return appErrors.New(appErrors.CodeConflict, "address already exists")
	}
	if err != nil {
		return fmt.Errorf("insert party address: %w", err)
	}
	return nil
}

func (r *PartyRepository) DeleteEmail(ctx context.Context, partyID, emailID string) error {
	return r.deleteChannel(ctx, `DELETE FROM party_emails WHERE party_id = $1 AND id = $2`, partyID, emailID)
}

func (r *PartyRepository) DeletePhone(ctx context.Context, partyID, phoneID string) error {
	return r.deleteChannel(ctx, `DELETE FROM party_phones WHERE party_id = $1 AND id = $2`, partyID, phoneID)
}

func (r *PartyRepository) DeleteAddress(ctx context.Context, partyID, addressID string) error {
	return r.deleteChannel(ctx, `DELETE FROM party_addresses WHERE party_id = $1 AND id = $2`, partyID, addressID)
}

func (r *PartyRepository) deleteChannel(ctx context.Context, q, partyID, channelID string) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	tag, err := pool.Exec(ctx, q, partyID, channelID)
	if err != nil {
		return fmt.Errorf("delete party channel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return appErrors.New(appErrors.CodeNotFound, "channel not found")
	}
	return nil
}

func (r *PartyRepository) loadChannels(ctx context.Context, party *domain.Party) error {
	pool, err := r.registry.GetPool(ctx)
	if err != nil {
		return fmt.Errorf("get tenant db pool: %w", err)
	}
	emails, err := pool.Query(ctx, `
		SELECT id, value, kind, source, created_at FROM party_emails WHERE party_id = $1 ORDER BY created_at, id
	`, party.ID)
	if err != nil {
		return fmt.Errorf("list party emails: %w", err)
	}
	defer emails.Close()
	party.Emails = []domain.PartyEmail{}
	for emails.Next() {
		var e domain.PartyEmail
		if err := emails.Scan(&e.ID, &e.Value, &e.Kind, &e.Source, &e.CreatedAt); err != nil {
			return fmt.Errorf("scan party email: %w", err)
		}
		party.Emails = append(party.Emails, e)
	}
	if err := emails.Err(); err != nil {
		return err
	}

	phones, err := pool.Query(ctx, `
		SELECT id, value, normalized, source, created_at FROM party_phones WHERE party_id = $1 ORDER BY created_at, id
	`, party.ID)
	if err != nil {
		return fmt.Errorf("list party phones: %w", err)
	}
	defer phones.Close()
	party.Phones = []domain.PartyPhone{}
	for phones.Next() {
		var p domain.PartyPhone
		if err := phones.Scan(&p.ID, &p.Value, &p.Normalized, &p.Source, &p.CreatedAt); err != nil {
			return fmt.Errorf("scan party phone: %w", err)
		}
		party.Phones = append(party.Phones, p)
	}
	if err := phones.Err(); err != nil {
		return err
	}

	addrs, err := pool.Query(ctx, `
		SELECT id, line, city, department, postal_zone, country_code, kind, fingerprint, source, created_at
		FROM party_addresses WHERE party_id = $1 ORDER BY created_at, id
	`, party.ID)
	if err != nil {
		return fmt.Errorf("list party addresses: %w", err)
	}
	defer addrs.Close()
	party.Addresses = []domain.PartyAddress{}
	for addrs.Next() {
		var a domain.PartyAddress
		if err := addrs.Scan(&a.ID, &a.Line, &a.City, &a.Department, &a.PostalZone, &a.CountryCode, &a.Kind, &a.Fingerprint, &a.Source, &a.CreatedAt); err != nil {
			return fmt.Errorf("scan party address: %w", err)
		}
		party.Addresses = append(party.Addresses, a)
	}
	return addrs.Err()
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func insertPartyChannels(ctx context.Context, tx execer, party domain.Party) error {
	for _, email := range party.Emails {
		if _, err := tx.Exec(ctx, `
			INSERT INTO party_emails (id, party_id, value, kind, source, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, email.ID, party.ID, email.Value, email.Kind, email.Source, email.CreatedAt); err != nil {
			return fmt.Errorf("insert party email: %w", err)
		}
	}
	for _, phone := range party.Phones {
		if _, err := tx.Exec(ctx, `
			INSERT INTO party_phones (id, party_id, value, normalized, source, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, phone.ID, party.ID, phone.Value, phone.Normalized, phone.Source, phone.CreatedAt); err != nil {
			return fmt.Errorf("insert party phone: %w", err)
		}
	}
	for _, addr := range party.Addresses {
		if _, err := tx.Exec(ctx, `
			INSERT INTO party_addresses (id, party_id, line, city, department, postal_zone, country_code, kind, fingerprint, source, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, addr.ID, party.ID, addr.Line, addr.City, addr.Department, addr.PostalZone, addr.CountryCode, addr.Kind, addr.Fingerprint, addr.Source, addr.CreatedAt); err != nil {
			return fmt.Errorf("insert party address: %w", err)
		}
	}
	return nil
}

func scanParty(row pgx.Row) (*domain.Party, error) {
	var party domain.Party
	err := row.Scan(
		&party.ID, &party.TaxID, &party.Name, &party.Roles, &party.Status, &party.CreationSource,
		&party.SchemeID, &party.TaxpayerKind, &party.TaxLevelCodes, &party.CreatedAt, &party.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan party: %w", err)
	}
	if party.TaxLevelCodes == nil {
		party.TaxLevelCodes = []string{}
	}
	party.Emails = []domain.PartyEmail{}
	party.Phones = []domain.PartyPhone{}
	party.Addresses = []domain.PartyAddress{}
	return &party, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
