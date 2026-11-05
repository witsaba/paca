package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/user"
)

// The names the database gives to the constraints of the table of the
// businesses.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	businessesPrimaryKeyConstraint = "businesses_pkey"
	businessesPersonalUniqueIndex  = "businesses_one_personal_per_owner"
)

// BusinessRepository is the adapter of the storage of the businesses.
//
// It writes one statement at a time. It does not read a row to find out
// whether it may write it: the identity of the business and the partial
// unique index of the personal business tell the truth by themselves, and the
// adapter maps that truth to the errors of the port.
type BusinessRepository struct {
	pool *Pool
}

// NewBusinessRepository returns the repository of the businesses, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewBusinessRepository(pool *Pool) (*BusinessRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &BusinessRepository{pool: pool}, nil
}

// Create writes a business and answers it with the identity that the
// database generated.
//
// The data is cleaned by the domain before it is written, so the row never
// carries a name with its spaces, whatever the caller sends.
//
// The error is business.ErrAlreadyExists when the owner already holds a
// personal business, and business.ErrNotFound when the owner is not a person
// of the storage.
func (r *BusinessRepository) Create(ctx context.Context, owner user.ID, written business.Business) (business.Business, error) {
	clean, err := business.New(owner, written.Kind, written.Name, written.Currency)
	if err != nil {
		return business.Business{}, err
	}

	if r == nil || r.pool == nil {
		return business.Business{}, business.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreateBusiness(ctx, db.CreateBusinessParams{
		OwnerID:  int64(owner),
		Kind:     clean.Kind.String(),
		Name:     clean.Name,
		Currency: clean.Currency,
	})
	if err != nil {
		return business.Business{}, writeBusinessError(err)
	}

	return businessFromRow(row)
}

// Find returns the business of the identity.
//
// The error is business.ErrNotFound when no business carries that identity.
func (r *BusinessRepository) Find(ctx context.Context, id business.ID) (business.Business, error) {
	if r == nil || r.pool == nil {
		return business.Business{}, business.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).GetBusinessByID(ctx, int64(id))
	if err != nil {
		return business.Business{}, writeBusinessError(err)
	}

	return businessFromRow(row)
}

// FindPersonalByOwner returns the personal business of a person.
//
// The rule of the domain: a person has exactly one personal business, and
// the partial unique index of the table holds the same rule.
//
// The error is business.ErrNotFound when the person holds no personal
// business.
func (r *BusinessRepository) FindPersonalByOwner(ctx context.Context, owner user.ID) (business.Business, error) {
	if r == nil || r.pool == nil {
		return business.Business{}, business.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).GetPersonalBusinessByOwner(ctx, int64(owner))
	if err != nil {
		return business.Business{}, writeBusinessError(err)
	}

	return businessFromRow(row)
}

// businessFromRow maps a row of the businesses table onto the model of the
// domain.
//
// The kind is read again by the domain: the row may come from a table that
// was changed by another writer, and the model does not trust a stranger
// with a kind it does not know.
func businessFromRow(row db.Business) (business.Business, error) {
	kind, err := business.ParseKind(row.Kind)
	if err != nil {
		return business.Business{}, fmt.Errorf("the row carries an unknown kind: %w", err)
	}

	return business.Business{
		ID:        business.ID(row.ID),
		OwnerID:   user.ID(row.OwnerID),
		Kind:      kind,
		Name:      row.Name,
		Currency:  row.Currency,
		CreatedAt: timestamp(row.CreatedAt),
		UpdatedAt: timestamp(row.UpdatedAt),
	}, nil
}

// writeBusinessError translates the truth of the database into the errors of
// the port of the businesses.
//
// A business that is already there is the refusal of the identity or of the
// partial unique index of the personal business, and the owner that is not a
// person of the storage is the refusal of the foreign key.
func writeBusinessError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return business.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		switch pgErr.ConstraintName {
		case businessesPrimaryKeyConstraint, businessesPersonalUniqueIndex:
			return fmt.Errorf("%w: that business is already there", business.ErrAlreadyExists)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the owner is not a person of the storage", business.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", business.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
