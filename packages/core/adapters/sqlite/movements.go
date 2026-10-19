package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// MovementRepository is the SQLite implementation of the movement port.
//
// It follows the shape of the account repository: the same naming, the same
// rules about the errors, and the same reading of the optional columns.
type MovementRepository struct {
	db *DB
}

// NewMovementRepository returns the movement repository over the given
// database.
func NewMovementRepository(db *DB) *MovementRepository {
	return &MovementRepository{db: db}
}

// The repository implements the port of the application.
var _ ports.MovementRepository = (*MovementRepository)(nil)

// The columns of a movement, in the order the scanner reads them.
//
// The two references of the schema are read as nullable, and the texts are
// read as nullable too: the domain has no nulls, only empty values and
// absent references.
const movementColumns = `id, account_id, kind, amount, occurred_on, category,
	counterparty_contact_id, loan_id, notes, created_at`

// Create stores a new movement and returns it with its identity filled in.
func (r *MovementRepository) Create(ctx context.Context, movement domain.Movement) (domain.Movement, error) {
	var id int64
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO movements
			(account_id, kind, amount, occurred_on, category,
			 counterparty_contact_id, loan_id, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		movement.AccountID, movement.Kind, movement.Amount, movement.OccurredOn,
		optionalText(movement.Category),
		optionalReference(movement.CounterpartyContactID),
		optionalReference(movement.LoanID),
		optionalText(movement.Notes),
	).Scan(&id, &createdAt)
	if err != nil {
		return domain.Movement{}, err
	}

	movement.ID = id
	movement.CreatedAt = createdAt

	return movement, nil
}

// GetByID returns the movement with the given identity.
func (r *MovementRepository) GetByID(ctx context.Context, id int64) (domain.Movement, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+movementColumns+` FROM movements WHERE id = ?`, id)

	movement, err := scanMovement(row.Scan)
	if err != nil {
		return domain.Movement{}, notFound(err, "the movement does not exist")
	}

	return movement, nil
}

// List returns the movements that match the filter, most recent first.
//
// The filter narrows the query with conditions over parameters: the values
// of the filter never become part of the query itself.
func (r *MovementRepository) List(ctx context.Context, filter ports.MovementFilter) ([]domain.Movement, error) {
	conditions := []string{}
	args := []any{}

	if filter.AccountID != 0 {
		conditions = append(conditions, "account_id = ?")
		args = append(args, filter.AccountID)
	}

	if len(filter.Kinds) > 0 {
		placeholders := make([]string, 0, len(filter.Kinds))
		for _, kind := range filter.Kinds {
			placeholders = append(placeholders, "?")
			args = append(args, kind)
		}

		conditions = append(conditions, "kind IN ("+strings.Join(placeholders, ", ")+")")
	}

	if filter.LoanID != 0 {
		conditions = append(conditions, "loan_id = ?")
		args = append(args, filter.LoanID)
	}

	period, periodArgs := periodConditions(filter.Period)
	conditions = append(conditions, period...)
	args = append(args, periodArgs...)

	query := `SELECT ` + movementColumns + ` FROM movements` + whereClause(conditions)

	query += ` ORDER BY occurred_on DESC, id DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	return r.list(ctx, query, args...)
}

// Update stores the changes of an existing movement.
func (r *MovementRepository) Update(ctx context.Context, movement domain.Movement) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE movements
		SET account_id = ?, kind = ?, amount = ?, occurred_on = ?, category = ?,
			counterparty_contact_id = ?, loan_id = ?, notes = ?
		WHERE id = ?`,
		movement.AccountID, movement.Kind, movement.Amount, movement.OccurredOn,
		optionalText(movement.Category),
		optionalReference(movement.CounterpartyContactID),
		optionalReference(movement.LoanID),
		optionalText(movement.Notes),
		movement.ID,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return domain.New(domain.CodeNotFound, "the movement does not exist")
	}

	return nil
}

// Delete removes a movement.
func (r *MovementRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM movements WHERE id = ?`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return domain.New(domain.CodeNotFound, "the movement does not exist")
	}

	return nil
}

// BalanceByAccount returns the balance of every account that holds movements.
//
// The balance is the money in minus the money out: the sign of the kind of a
// movement says which side of the balance it lands on. An account without
// movements is not part of the map.
func (r *MovementRepository) BalanceByAccount(ctx context.Context) (map[int64]int64, error) {
	balances := map[int64]int64{}

	err := eachAggregate(ctx, r.db,
		`SELECT account_id, kind, SUM(amount) FROM movements
		 GROUP BY account_id, kind`, nil,
		func() *int64 { return new(int64) },
		func(accountID *int64, kind domain.MovementKind, amount int64) {
			balances[*accountID] += int64(kind.Sign()) * amount
		})
	if err != nil {
		return nil, err
	}

	return balances, nil
}

// TotalByKind returns the total of every kind of movement over a period.
//
// A total follows the sign of its kind: the total of an income is positive
// and the total of an expense is negative. A kind without movements is not
// part of the map.
func (r *MovementRepository) TotalByKind(ctx context.Context, period ports.Period) (map[domain.MovementKind]int64, error) {
	totals := map[domain.MovementKind]int64{}

	conditions, args := periodConditions(period)

	rows, err := r.db.QueryContext(ctx,
		`SELECT kind, SUM(amount) FROM movements`+whereClause(conditions)+` GROUP BY kind`, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var kind domain.MovementKind
		var amount int64

		if err := rows.Scan(&kind, &amount); err != nil {
			return nil, err
		}

		totals[kind] = int64(kind.Sign()) * amount
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return totals, nil
}

// TotalByCategory returns the total of the movements of every category over
// a period.
//
// A total follows the sign of the kind of its movements, and the movements of
// a category may not share one kind. A movement without a category is
// grouped under the empty string. A category without movements is not part of
// the map.
func (r *MovementRepository) TotalByCategory(ctx context.Context, period ports.Period) (map[string]int64, error) {
	totals := map[string]int64{}

	conditions, args := periodConditions(period)

	err := eachAggregate(ctx, r.db,
		`SELECT COALESCE(category, ''), kind, SUM(amount) FROM movements`+
			whereClause(conditions)+` GROUP BY COALESCE(category, ''), kind`, args,
		func() *string { return new(string) },
		func(category *string, kind domain.MovementKind, amount int64) {
			totals[*category] += int64(kind.Sign()) * amount
		})
	if err != nil {
		return nil, err
	}

	return totals, nil
}

// list runs a query over the movements and reads every row it returns.
func (r *MovementRepository) list(ctx context.Context, query string, args ...any) ([]domain.Movement, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	movements := []domain.Movement{}
	for rows.Next() {
		movement, err := scanMovement(rows.Scan)
		if err != nil {
			return nil, err
		}

		movements = append(movements, movement)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movements, nil
}

// eachAggregate runs a query that returns a group, the kind and the amount of
// every movement it groups, and hands every row to the given function.
//
// The group is the account of the balance and the category of the totals by
// category. The amount of a row is the sum of the movements of its group and
// of its kind. The function that reads the group returns the destination the
// row is scanned into.
func eachAggregate[T any](
	ctx context.Context,
	db *DB,
	query string,
	args []any,
	newGroup func() T,
	add func(group T, kind domain.MovementKind, amount int64),
) error {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		group := newGroup()
		var kind domain.MovementKind
		var amount int64

		if err := rows.Scan(group, &kind, &amount); err != nil {
			return err
		}

		add(group, kind, amount)
	}

	return rows.Err()
}

// scanMovement reads one movement from a row.
//
// The optional columns of the schema are read as nullable, and a value that
// is not there is the empty value of the domain: the domain has no nulls,
// only empty values and absent references.
func scanMovement(scan func(dest ...any) error) (domain.Movement, error) {
	var movement domain.Movement
	var category, notes sql.NullString
	var counterpartyContactID, loanID sql.NullInt64

	err := scan(
		&movement.ID, &movement.AccountID, &movement.Kind, &movement.Amount,
		&movement.OccurredOn, &category, &counterpartyContactID, &loanID,
		&notes, &movement.CreatedAt,
	)
	if err != nil {
		return domain.Movement{}, err
	}

	movement.Category = category.String
	movement.Notes = notes.String

	if counterpartyContactID.Valid {
		movement.CounterpartyContactID = &counterpartyContactID.Int64
	}

	if loanID.Valid {
		movement.LoanID = &loanID.Int64
	}

	return movement, nil
}

// optionalReference returns the value of an optional reference, or nil when
// there is none.
//
// The schema stores the absence of a reference as NULL, and the domain holds
// it as a nil pointer: the two are the same absence.
func optionalReference(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

// periodConditions returns the conditions of a period and their arguments.
//
// Both ends of the period are included, and an empty end leaves that side of
// the period open.
func periodConditions(period ports.Period) ([]string, []any) {
	conditions := []string{}
	args := []any{}

	if period.From != "" {
		conditions = append(conditions, "occurred_on >= ?")
		args = append(args, period.From)
	}

	if period.To != "" {
		conditions = append(conditions, "occurred_on <= ?")
		args = append(args, period.To)
	}

	return conditions, args
}

// whereClause returns the condition of a query, or nothing when there is
// none.
func whereClause(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}

	return ` WHERE ` + strings.Join(conditions, ` AND `)
}
