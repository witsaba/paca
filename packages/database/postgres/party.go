package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/party"
)

// PartyRepository is the adapter of the storage of the parties.
//
// The roles of a party travel as JSONB: they are the roles the party holds
// in the business. The adapter encodes them, and the domain reads them back
// through its own constructors: the row never decides what a party is.
//
// The legal human that represents a store or a farm travels in the party it
// belongs to, and nowhere else: it is a fact of the party, and it is never
// a row of the people of PACA.
type PartyRepository struct {
	pool *Pool
}

// NewPartyRepository returns the repository of the parties, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewPartyRepository(pool *Pool) (*PartyRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &PartyRepository{pool: pool}, nil
}

// Create writes a party of the business, and answers it with the identity
// that the database generated.
//
// The rules of the domain travel with the port: the kind of the party, the
// name, the roles and the legal human are the rules of the party, and the
// table holds the same ones as a defense in depth.
//
// The error is party.ErrNotFound when the business of the party is not
// there, party.ErrInvalidValue when the row breaks a rule of the table, and
// party.ErrNoStorage when the storage is not open.
func (r *PartyRepository) Create(ctx context.Context, p party.Party) (party.Party, error) {
	if r == nil || r.pool == nil {
		return party.Party{}, party.ErrNoStorage
	}

	roles, err := encodeRoles(p.Roles)
	if err != nil {
		return party.Party{}, err
	}

	row, err := r.pool.queriesFor(ctx).CreateParty(ctx, db.CreatePartyParams{
		BusinessID:     int64(p.BusinessID),
		Kind:           p.Kind.String(),
		Name:           p.Name,
		Roles:          roles,
		Phone:          optionalText(p.Phone),
		Email:          optionalText(p.Email),
		Address:        optionalText(p.Address),
		LegalHumanName: optionalText(p.LegalHuman.Name),
		LegalFigure:    optionalText(p.LegalHuman.Figure),
		TaxID:          optionalText(p.TaxID),
	})
	if err != nil {
		return party.Party{}, writePartyError(err)
	}

	return partyFromRow(row)
}

// List returns the parties of a business, in the order of their identity.
//
// The list holds every party of the business, the ones that share a name
// among them: the name of a party is not unique inside the business.
//
// The error is party.ErrNoStorage when the storage is not open.
func (r *PartyRepository) List(ctx context.Context, businessID business.ID) ([]party.Party, error) {
	if r == nil || r.pool == nil {
		return nil, party.ErrNoStorage
	}

	rows, err := r.pool.queriesFor(ctx).ListPartiesByBusiness(ctx, int64(businessID))
	if err != nil {
		return nil, writePartyError(err)
	}

	held := make([]party.Party, 0, len(rows))
	for _, row := range rows {
		found, err := partyFromRow(row)
		if err != nil {
			return nil, err
		}

		held = append(held, found)
	}

	return held, nil
}

// partyFromRow maps a row of the parties onto the model of the domain.
//
// The party is built by the domain: the row may come from a table that was
// changed by another writer, and the model does not trust a stranger with a
// party it does not know.
func partyFromRow(row db.Party) (party.Party, error) {
	roles, err := decodeRoles(row.Roles)
	if err != nil {
		return party.Party{}, err
	}

	kind, err := party.ParseKind(row.Kind)
	if err != nil {
		return party.Party{}, err
	}

	// The legal human that represents the party is built by the domain: the
	// row may come from a table that was changed by another writer, and the
	// model does not trust a stranger with a party it does not know.
	human, err := party.LegalHumanOf(kind, text(row.LegalHumanName), text(row.LegalFigure))
	if err != nil {
		return party.Party{}, err
	}

	built, err := party.New(
		business.ID(row.BusinessID),
		kind,
		row.Name,
		roles,
		text(row.Phone),
		text(row.Email),
		text(row.Address),
		human,
	)
	if err != nil {
		return party.Party{}, err
	}

	built.ID = party.ID(row.ID)
	built.CreatedAt = timestamp(row.CreatedAt)
	built.UpdatedAt = timestamp(row.UpdatedAt)

	// The tax identity of the party is an OPTIONAL fact of it: the column
	// is NULL when the party carries none, and the book never assumes
	// one.
	built = built.WithTaxIdentity(text(row.TaxID))

	return built, nil
}

// encodeRoles encodes the roles of the party for the storage.
//
// The roles travel as a list, the way the aliases of an ingredient travel.
func encodeRoles(roles []party.Role) ([]byte, error) {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.String())
	}

	encoded, err := json.Marshal(names)
	if err != nil {
		return nil, fmt.Errorf("%w: the roles of the party could not be encoded: %w", party.ErrInvalidValue, err)
	}

	return encoded, nil
}

// decodeRoles decodes the roles of the party from the storage.
//
// The roles are built by the domain: the row never decides what a role is.
func decodeRoles(raw []byte) ([]party.Role, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, fmt.Errorf("%w: the roles of the party are not a list: %w", party.ErrInvalidValue, err)
	}

	roles := make([]party.Role, 0, len(names))
	for _, name := range names {
		role, err := party.ParseRole(name)
		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

// writePartyError translates the truth of the database into the errors of
// the port of the parties.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business that is not there is the refusal of the foreign
// key.
func writePartyError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return party.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case foreignKeyViolation:
		// The refusal of the foreign key of the business says that the
		// business of the party is not there: it is one truth, and it has
		// one message for the agent.
		return fmt.Errorf("%w: the constraint %q refused the row",
			party.ErrBusinessNotFound, pgErr.ConstraintName)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", party.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
