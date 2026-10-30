//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/user"
)

// The codes of the errors of PostgreSQL the assertions expect.
const (
	// uniqueViolation is the code of a row that breaks a unique constraint.
	uniqueViolation = "23505"

	// notNullViolation is the code of a row that leaves a NOT NULL column
	// empty.
	notNullViolation = "23502"

	// checkViolation is the code of a row that breaks a check constraint.
	checkViolation = "23514"
)

// The names of the constraints of the table.
const (
	// primaryKeyConstraint is the constraint of the primary key.
	primaryKeyConstraint = "users_pkey"

	// statusConstraint is the constraint that checks the state of the user.
	statusConstraint = "users_status_check"
)

// optionalColumns are the columns of the table that hold a value only when
// the value is known.
const optionalColumns = "full_name, status_observations, phone, email"

// TestTheTableHasTheColumnsOfTheSpecification proves that the table exists
// with exactly the columns, the types, the nullability and the defaults the
// specification describes.
//
// The columns are read from the catalog of the database, and not from the
// migration file: the file is what the database was asked to be, and the
// catalog is what it became.
func TestTheTableHasTheColumnsOfTheSpecification(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	type column struct {
		name       string
		dataType   string
		nullable   string
		defaultSQL string
	}

	want := []column{
		{name: "id", dataType: "bigint", nullable: "NO", defaultSQL: ""},
		{name: "full_name", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "status", dataType: "text", nullable: "NO", defaultSQL: "'onboarding'::text"},
		{name: "status_observations", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "phone", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "email", dataType: "text", nullable: "YES", defaultSQL: ""},
		{name: "created_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
		{name: "updated_at", dataType: "timestamp with time zone", nullable: "NO", defaultSQL: "now()"},
	}

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'users'
		ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("the columns of the table could not be read: %v", err)
	}
	defer rows.Close()

	got := make([]column, 0, len(want))
	for rows.Next() {
		var (
			name         string
			dataType     string
			nullable     string
			defaultValue sql.NullString
		)
		if err := rows.Scan(&name, &dataType, &nullable, &defaultValue); err != nil {
			t.Fatalf("a column of the table could not be read: %v", err)
		}
		got = append(got, column{
			name:       name,
			dataType:   dataType,
			nullable:   nullable,
			defaultSQL: defaultValue.String,
		})
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("the columns of the table could not be read: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("the table has %d columns, want %d: %+v", len(got), len(want), got)
	}
	for i, expected := range want {
		if got[i] != expected {
			t.Errorf("the column %d is %+v, want %+v", i, got[i], expected)
		}
	}

	// The relation is an ordinary table, and not a view over something else.
	var kind string
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT relkind FROM pg_class WHERE oid = 'public.users'::regclass`,
	).Scan(&kind); err != nil {
		t.Fatalf("the relation of the users could not be read: %v", err)
	}
	if kind != "r" {
		t.Errorf("the relation of the users is a %q, want an ordinary table", kind)
	}
}

// TestTheIdentifierIsThePrimaryKey proves that the identifier is the primary
// key of the row: the same identifier cannot be stored twice, and the error
// names the constraint of the primary key.
func TestTheIdentifierIsThePrimaryKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	if _, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("the row could not be inserted: %v", err)
	}

	_, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id) VALUES ($1)`, id)
	if err == nil {
		t.Fatal("the same identifier was accepted twice, want the primary key to refuse it")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != uniqueViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, uniqueViolation)
	}
	if pgErr.ConstraintName != primaryKeyConstraint {
		t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, primaryKeyConstraint)
	}

	// The constraint of the primary key is on the identifier, and on nothing
	// else.
	var columns string
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid = 'public.users'::regclass AND contype = 'p'`,
	).Scan(&columns); err != nil {
		t.Fatalf("the primary key could not be read: %v", err)
	}
	if columns != "PRIMARY KEY (id)" {
		t.Errorf("the primary key is %q, want %q", columns, "PRIMARY KEY (id)")
	}
}

// TestTheIdentifierIsNeitherSequentialNorGenerated proves that the database
// never invents an identifier: the column has no default, it is not an
// identity, it is not generated, and no sequence is attached to it.
//
// The proof comes from the catalog of the database. The comment of the
// migration says the identifier is not generated, and a comment is a promise,
// not a fact.
func TestTheIdentifierIsNeitherSequentialNorGenerated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// The column has no default at all.
	var defaultValue sql.NullString
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'id'`,
	).Scan(&defaultValue); err != nil {
		t.Fatalf("the default of the identifier could not be read: %v", err)
	}
	if defaultValue.Valid {
		t.Errorf("the identifier has the default %q, want none", defaultValue.String)
	}

	// The column is neither an identity nor a generated column: the empty
	// string is what the catalog holds for a column that is neither.
	var identity, generated string
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT attidentity, attgenerated
		FROM pg_attribute
		WHERE attrelid = 'public.users'::regclass AND attname = 'id' AND NOT attisdropped`,
	).Scan(&identity, &generated); err != nil {
		t.Fatalf("the identity of the identifier could not be read: %v", err)
	}
	if identity != "" {
		t.Errorf("the identifier is the identity %q, want none", identity)
	}
	if generated != "" {
		t.Errorf("the identifier is the generated column %q, want none", generated)
	}

	// No sequence is attached to the column.
	var sequence sql.NullString
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT pg_get_serial_sequence('public.users', 'id')`,
	).Scan(&sequence); err != nil {
		t.Fatalf("the sequence of the identifier could not be read: %v", err)
	}
	if sequence.Valid {
		t.Errorf("the identifier is attached to the sequence %q, want none", sequence.String)
	}

	// And no sequence of the database depends on the table.
	var sequences int
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM pg_depend d
		JOIN pg_class s ON s.oid = d.objid
		WHERE d.refobjid = 'public.users'::regclass AND s.relkind = 'S'`,
	).Scan(&sequences); err != nil {
		t.Fatalf("the sequences of the table could not be counted: %v", err)
	}
	if sequences != 0 {
		t.Errorf("the table depends on %d sequences, want none", sequences)
	}
}

// TestTheIdentifierIsRequired proves that the database does not invent an
// identifier: a row without one is refused, and nothing is stored.
func TestTheIdentifierIsRequired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const name = "A Person Without An Identifier"

	_, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (full_name) VALUES ($1)`, name)
	if err == nil {
		t.Fatal("the row without an identifier was accepted, want it to be refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != notNullViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, notNullViolation)
	}
	if pgErr.ColumnName != "id" {
		t.Errorf("the column of the error is %q, want %q", pgErr.ColumnName, "id")
	}

	var rows int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM users WHERE full_name = $1`, name,
	).Scan(&rows); err != nil {
		t.Fatalf("the rows could not be counted: %v", err)
	}
	if rows != 0 {
		t.Errorf("the database stored %d rows without an identifier, want none", rows)
	}
}

// TestTheStatusDefaultsToOnboarding proves that a row that does not name its
// state starts in the onboarding state.
func TestTheStatusDefaultsToOnboarding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	if _, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("the row could not be inserted: %v", err)
	}

	var status string
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT status FROM users WHERE id = $1`, id,
	).Scan(&status); err != nil {
		t.Fatalf("the state of the row could not be read: %v", err)
	}

	if status != user.StatusOnboarding.String() {
		t.Errorf("the state of the row is %q, want %q", status, user.StatusOnboarding)
	}
}

// TestEveryStateOfTheUserIsAccepted proves that the four states of the
// domain are the four states the database stores.
//
// The list comes from the domain, so a state added there and not here fails
// the test instead of passing it by accident.
func TestEveryStateOfTheUserIsAccepted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	for _, status := range user.AllStatuses() {
		t.Run(status.String(), func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()

			// The two closed states carry an observation: the database
			// refuses a closed state without one, and the test proves the
			// four states of the domain, not the four states of a row
			// without an observation.
			query := `INSERT INTO users (id, status) VALUES ($1, $2)`
			args := []any{id, status.String()}
			if user.RequiresObservations(status) {
				query = `INSERT INTO users (id, status, status_observations) VALUES ($1, $2, $3)`
				args = append(args, "the person asked to leave")
			}

			if _, err := theSuite.db.ExecContext(ctx, query, args...); err != nil {
				t.Fatalf("the state %q was refused: %v", status, err)
			}

			var stored string
			if err := theSuite.db.QueryRowContext(ctx,
				`SELECT status FROM users WHERE id = $1`, id,
			).Scan(&stored); err != nil {
				t.Fatalf("the state of the row could not be read: %v", err)
			}

			if stored != status.String() {
				t.Errorf("the state of the row is %q, want %q", stored, status)
			}
		})
	}
}

// TestTheStatusIsCheckedByTheDatabase proves that the check lives in the
// database: a state that does not exist is refused, the error names the
// constraint, and nothing is stored.
func TestTheStatusIsCheckedByTheDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name   string
		status string
	}{
		{name: "an unknown state", status: "unknown"},
		{name: "the empty state", status: ""},
		{name: "a state in another case", status: "ACTIVE"},
		{name: "a state with a trailing space", status: "active "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()

			_, err := theSuite.db.ExecContext(ctx,
				`INSERT INTO users (id, status) VALUES ($1, $2)`, id, tc.status,
			)
			if err == nil {
				t.Fatalf("the state %q was accepted, want the check to refuse it", tc.status)
			}

			pgErr := postgresError(t, err)
			if pgErr.Code != checkViolation {
				t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
			}
			if pgErr.ConstraintName != statusConstraint {
				t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, statusConstraint)
			}

			var rows int
			if err := theSuite.db.QueryRowContext(ctx,
				`SELECT count(*) FROM users WHERE id = $1`, id,
			).Scan(&rows); err != nil {
				t.Fatalf("the rows could not be counted: %v", err)
			}
			if rows != 0 {
				t.Errorf("the database stored %d rows with the state %q, want none", rows, tc.status)
			}
		})
	}
}

// TestTheStatusIsRequired proves that the state of a row is never empty: the
// column does not accept NULL, not even when the default would fill it.
func TestTheStatusIsRequired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	_, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id, status) VALUES ($1, NULL)`, id)
	if err == nil {
		t.Fatal("the row without a state was accepted, want it to be refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != notNullViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, notNullViolation)
	}
	if pgErr.ColumnName != "status" {
		t.Errorf("the column of the error is %q, want %q", pgErr.ColumnName, "status")
	}
}

// TestTheOptionalColumnsAcceptNothing proves that the four optional columns
// hold nothing when the value is not known: they accept NULL, and they may
// be left out of the insert entirely.
//
// The two ways of leaving a value out are the same fact in the database, so
// a reader of the row never has to guess whether a value is missing.
func TestTheOptionalColumnsAcceptNothing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name  string
		named bool
	}{
		{name: "the columns are named and empty", named: true},
		{name: "the columns are left out", named: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()

			query := `INSERT INTO users (id) VALUES ($1)`
			args := []any{id}
			if tc.named {
				query = `INSERT INTO users (id, ` + optionalColumns + `) VALUES ($1, $2, $3, $4, $5)`
				args = append(args, nil, nil, nil, nil)
			}

			if _, err := theSuite.db.ExecContext(ctx, query, args...); err != nil {
				t.Fatalf("the row could not be inserted: %v", err)
			}

			var (
				fullName           sql.NullString
				statusObservations sql.NullString
				phone              sql.NullString
				email              sql.NullString
			)
			if err := theSuite.db.QueryRowContext(ctx, `
				SELECT full_name, status_observations, phone, email
				FROM users
				WHERE id = $1`,
				id,
			).Scan(&fullName, &statusObservations, &phone, &email); err != nil {
				t.Fatalf("the optional columns of the row could not be read: %v", err)
			}

			for name, value := range map[string]sql.NullString{
				"full_name":           fullName,
				"status_observations": statusObservations,
				"phone":               phone,
				"email":               email,
			} {
				if value.Valid {
					t.Errorf("the column %q holds %q, want NULL", name, value.String)
				}
			}
		})
	}
}

// TestTheTimestampsAreFilledByTheSystem proves that the database fills the
// two timestamps of a row on insert, and that it fills them with the moment
// of the insert.
func TestTheTimestampsAreFilledByTheSystem(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	before := time.Now().UTC()
	if _, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("the row could not be inserted: %v", err)
	}
	after := time.Now().UTC()

	var createdAt, updatedAt time.Time
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM users WHERE id = $1`, id,
	).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("the timestamps of the row could not be read: %v", err)
	}

	// The clock of the machine and the clock of the container are the same
	// clock, so a moment of tolerance is enough for the two readings.
	tolerance := time.Second

	if createdAt.IsZero() {
		t.Error("the creation time of the row is empty, want the system to fill it")
	}
	if updatedAt.IsZero() {
		t.Error("the update time of the row is empty, want the system to fill it")
	}

	if createdAt.Before(before.Add(-tolerance)) || createdAt.After(after.Add(tolerance)) {
		t.Errorf("the creation time of the row is %s, want it between %s and %s", createdAt, before, after)
	}
	if updatedAt.Before(before.Add(-tolerance)) || updatedAt.After(after.Add(tolerance)) {
		t.Errorf("the update time of the row is %s, want it between %s and %s", updatedAt, before, after)
	}
}

// TestTheTimestampsAreReadInUTC proves that the timestamps of a row are
// instants, and that they are read in UTC.
//
// The session of the reader is moved away from UTC before the row is read,
// so the reading cannot be right by accident: a column that held a wall
// clock would bring the row back in another offset.
//
// The driver of the suite reads a timestamp in the zone of the process, the
// way every Go program reads one, so the reader that wants UTC asks the
// driver for it. Both readings carry the same instant, and the one that asks
// for UTC gets it in UTC.
func TestTheTimestampsAreReadInUTC(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	before := time.Now().UTC()
	if _, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("the row could not be inserted: %v", err)
	}
	after := time.Now().UTC()

	conn, err := theSuite.db.Conn(ctx)
	if err != nil {
		t.Fatalf("a connection could not be taken: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SET TIME ZONE INTERVAL '+09:00' HOUR TO MINUTE`); err != nil {
		t.Fatalf("the timezone of the session could not be moved: %v", err)
	}

	var zone string
	if err := conn.QueryRowContext(ctx, `SHOW TIME ZONE`).Scan(&zone); err != nil {
		t.Fatalf("the timezone of the session could not be read: %v", err)
	}
	if zone == "UTC" {
		t.Fatal("the timezone of the session is still UTC, want it moved away from UTC")
	}

	var createdAt, updatedAt time.Time
	if err := conn.QueryRowContext(ctx,
		`SELECT created_at, updated_at FROM users WHERE id = $1`, id,
	).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("the timestamps of the row could not be read: %v", err)
	}

	tolerance := time.Second
	for name, value := range map[string]time.Time{
		"created_at": createdAt,
		"updated_at": updatedAt,
	} {
		instant := value.UTC()
		if instant.Before(before.Add(-tolerance)) || instant.After(after.Add(tolerance)) {
			t.Errorf("the column %q holds the instant %s, want it between %s and %s", name, instant, before, after)
		}
	}

	// The database holds the same instant, and the epoch of it does not
	// depend on the session of the reader.
	var epoch float64
	if err := conn.QueryRowContext(ctx,
		`SELECT extract(epoch FROM created_at) FROM users WHERE id = $1`, id,
	).Scan(&epoch); err != nil {
		t.Fatalf("the epoch of the creation time could not be read: %v", err)
	}
	if math.Abs(epoch-float64(createdAt.UTC().UnixMicro())/1e6) > 0.001 {
		t.Errorf("the epoch of the creation time is %f, want %f", epoch, float64(createdAt.UTC().UnixMicro())/1e6)
	}

	// And the reader that asks for UTC reads the instant in UTC.
	utc := openUTCConnection(ctx, t, theSuite.url)

	var readInUTC time.Time
	if err := utc.QueryRow(ctx,
		`SELECT created_at FROM users WHERE id = $1`, id,
	).Scan(&readInUTC); err != nil {
		t.Fatalf("the creation time of the row could not be read: %v", err)
	}

	if readInUTC.Location().String() != "UTC" {
		t.Errorf("the creation time of the row was read in %q, want UTC", readInUTC.Location())
	}
	if _, offset := readInUTC.Zone(); offset != 0 {
		t.Errorf("the creation time of the row was read with the offset %d, want zero", offset)
	}
	if !readInUTC.Equal(createdAt) {
		t.Errorf("the creation time read in UTC is %s, want %s", readInUTC, createdAt)
	}
}

// TestTheUpdatedAtIsNotMaintainedByATrigger proves that the update time of
// a row is a default and not a promise of the database: an update that does
// not name it leaves it exactly as it was.
//
// The choice is deliberate. Whoever updates a row is responsible for the
// update time, and the database does not second-guess it.
func TestTheUpdatedAtIsNotMaintainedByATrigger(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	if _, err := theSuite.db.ExecContext(ctx, `INSERT INTO users (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("the row could not be inserted: %v", err)
	}

	var before time.Time
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT updated_at FROM users WHERE id = $1`, id,
	).Scan(&before); err != nil {
		t.Fatalf("the update time of the row could not be read: %v", err)
	}

	if _, err := theSuite.db.ExecContext(ctx,
		`UPDATE users SET full_name = $2 WHERE id = $1`, id, "A New Name",
	); err != nil {
		t.Fatalf("the row could not be updated: %v", err)
	}

	var after time.Time
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT updated_at FROM users WHERE id = $1`, id,
	).Scan(&after); err != nil {
		t.Fatalf("the update time of the row could not be read: %v", err)
	}

	if !after.Equal(before) {
		t.Errorf("the update time of the row moved from %s to %s, want it untouched", before, after)
	}

	// No trigger of the database touches the table.
	var triggers int
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM pg_trigger
		WHERE tgrelid = 'public.users'::regclass AND NOT tgisinternal`,
	).Scan(&triggers); err != nil {
		t.Fatalf("the triggers of the table could not be counted: %v", err)
	}
	if triggers != 0 {
		t.Errorf("the table carries %d triggers, want none", triggers)
	}
}

// TestApplyingTheMigrationsTwiceChangesNothing proves that the schema has
// one writer and that the writer is idempotent: applying the migrations
// again applies nothing, and the version of the schema stays the same.
//
// The test does not run in parallel with the others: the single writer takes
// a lock of the database, and two of them at once would wait for each other.
func TestApplyingTheMigrationsTwiceChangesNothing(t *testing.T) {
	ctx := context.Background()

	before, err := theSuite.migrator.Version(ctx)
	if err != nil {
		t.Fatalf("the version of the schema could not be read: %v", err)
	}

	applied, err := theSuite.migrator.Up(ctx)
	if err != nil {
		t.Fatalf("the migrations could not be applied a second time: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("the migrations applied %d migrations a second time, want none: %+v", len(applied), applied)
	}

	after, err := theSuite.migrator.Version(ctx)
	if err != nil {
		t.Fatalf("the version of the schema could not be read: %v", err)
	}
	if after != before {
		t.Errorf("the version of the schema moved from %d to %d, want it untouched", before, after)
	}

	pending, err := theSuite.migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("the pending migrations could not be read: %v", err)
	}
	if pending {
		t.Error("a migration is still pending, want the schema to be up to date")
	}
}

// TestTheSchemaWasWrittenByTheSingleWriter proves that the single writer
// wrote the schema it says it wrote.
//
// The test reads the table of the migrations of the database itself: the
// writer is the only one that writes it, and the table is the proof that it
// did.
//
// The test does not run in parallel with the others: the single writer takes
// a lock of the database, and two of them at once would wait for each other.
func TestTheSchemaWasWrittenByTheSingleWriter(t *testing.T) {
	ctx := context.Background()

	rows, err := theSuite.db.QueryContext(ctx, `
		SELECT version_id, is_applied, tstamp
		FROM goose_db_version
		ORDER BY version_id`)
	if err != nil {
		t.Fatalf("the migrations of the database could not be read: %v", err)
	}
	defer rows.Close()

	applied := 0
	for rows.Next() {
		var version int64
		var isApplied bool
		var moment time.Time
		if err := rows.Scan(&version, &isApplied, &moment); err != nil {
			t.Fatalf("a migration of the database could not be read: %v", err)
		}

		if !isApplied {
			t.Errorf("the migration %d is not applied, want it applied", version)
		}
		if moment.IsZero() {
			t.Errorf("the migration %d has no moment of application, want one", version)
		}
		applied++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("the migrations of the database could not be read: %v", err)
	}

	if applied == 0 {
		t.Fatal("the database holds no migration, want the ones of the users table")
	}
}
