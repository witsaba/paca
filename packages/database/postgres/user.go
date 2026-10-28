package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/domain/user"
	"github.com/witsaba/paca/packages/database/internal/db"
)

// userFromRow maps a row of the users table onto the model of the domain.
//
// The nullable columns of the row become the empty fields of the model, and
// the status is read again by the domain: the row may come from a table
// that was changed by another writer, and the model does not trust a
// stranger with a state it does not know.
func userFromRow(row db.User) (user.User, error) {
	status, err := user.ParseStatus(row.Status)
	if err != nil {
		return user.User{}, fmt.Errorf("the row carries an unknown status: %w", err)
	}

	return user.User{
		ID:                 user.ID(row.ID),
		FullName:           text(row.FullName),
		Status:             status,
		StatusObservations: text(row.StatusObservations),
		Phone:              text(row.Phone),
		Email:              text(row.Email),
		CreatedAt:          timestamp(row.CreatedAt),
		UpdatedAt:          timestamp(row.UpdatedAt),
	}, nil
}

// text returns the value of a nullable text, or the empty string when the
// column holds NULL.
//
// The domain and the table agree on the rule: a value that is not set is
// NULL in the table and empty in the model. The library never writes an empty
// string, so the two facts never drift apart.
func text(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

// timestamp returns the value of a nullable timestamp, or the zero time when
// the column holds NULL.
func timestamp(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
