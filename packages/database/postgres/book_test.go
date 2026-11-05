package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/user"
)

// The error the database raises when it refuses a row.
func theRefusal(code, constraint string) error {
	return &pgconn.PgError{Code: code, ConstraintName: constraint}
}

func TestTheRepositoryOfTheBusinessesNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewBusinessRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfTheAccountsNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewAccountRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheTransactionOfTheStorageNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewUnitOfWork(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoriesWithoutADatabaseRefuseToWork(t *testing.T) {
	t.Parallel()

	businesses := &BusinessRepository{}
	accounts := &AccountRepository{}
	work := &UnitOfWork{}

	if _, err := businesses.Create(t.Context(), 1, theBusiness()); !errors.Is(err, business.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, business.ErrNoStorage)
	}
	if _, err := businesses.Find(t.Context(), 1); !errors.Is(err, business.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, business.ErrNoStorage)
	}
	if _, err := businesses.FindPersonalByOwner(t.Context(), 1); !errors.Is(err, business.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, business.ErrNoStorage)
	}
	if _, err := accounts.Create(t.Context(), theAccount()); !errors.Is(err, account.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, account.ErrNoStorage)
	}
	if _, err := accounts.CreateMany(t.Context(), nil); !errors.Is(err, account.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, account.ErrNoStorage)
	}
	if _, err := accounts.Find(t.Context(), 1); !errors.Is(err, account.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, account.ErrNoStorage)
	}
	if _, err := accounts.List(t.Context(), 1); !errors.Is(err, account.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, account.ErrNoStorage)
	}
	if err := work.Do(t.Context(), func(context.Context) error { return nil }); !errors.Is(err, ErrNoPool) {
		t.Errorf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRowOfABusinessIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	moment := time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC)

	mapped, err := businessFromRow(db.Business{
		ID:        77,
		OwnerID:   1234567,
		Kind:      "personal",
		Name:      "Negocio Personal",
		Currency:  "COP",
		CreatedAt: pgtype.Timestamptz{Time: moment, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: moment, Valid: true},
	})
	if err != nil {
		t.Fatalf("the row could not be mapped: %v", err)
	}

	if mapped.ID != business.ID(77) {
		t.Errorf("the identity is %d, want 77", mapped.ID)
	}
	if mapped.OwnerID != user.ID(1234567) {
		t.Errorf("the owner is %d, want 1234567", mapped.OwnerID)
	}
	if mapped.Kind != business.KindPersonal {
		t.Errorf("the kind is %q, want %q", mapped.Kind, business.KindPersonal)
	}
	if mapped.Name != "Negocio Personal" {
		t.Errorf("the name is %q, want %q", mapped.Name, "Negocio Personal")
	}
	if mapped.Currency != "COP" {
		t.Errorf("the currency is %q, want %q", mapped.Currency, "COP")
	}
	if !mapped.CreatedAt.Equal(moment) {
		t.Errorf("the moment of the creation is %v, want %v", mapped.CreatedAt, moment)
	}
}

func TestTheRowOfABusinessWithAnUnknownKindIsRefused(t *testing.T) {
	t.Parallel()

	_, err := businessFromRow(db.Business{Kind: "bakery"})

	if !errors.Is(err, business.ErrUnknownKind) {
		t.Fatalf("the error is %v, want %v", err, business.ErrUnknownKind)
	}
}

func TestTheRowOfAnAccountIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	due := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	mapped, err := accountFromRow(db.Account{
		ID:           9,
		BusinessID:   77,
		Kind:         "liability",
		Name:         "Préstamo: Juan",
		Counterparty: pgtype.Text{String: "Juan", Valid: true},
		DueDate:      pgtype.Date{Time: due, Valid: true},
	})
	if err != nil {
		t.Fatalf("the row could not be mapped: %v", err)
	}

	if mapped.ID != account.ID(9) {
		t.Errorf("the identity is %d, want 9", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.Kind != account.KindLiability {
		t.Errorf("the kind is %q, want %q", mapped.Kind, account.KindLiability)
	}
	if mapped.Counterparty != "Juan" {
		t.Errorf("the counterparty is %q, want %q", mapped.Counterparty, "Juan")
	}
	if mapped.DueDate == nil || !mapped.DueDate.Equal(due) {
		t.Errorf("the due date is %v, want %v", mapped.DueDate, due)
	}
}

func TestTheRowOfAnAccountWithoutADebtIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := accountFromRow(db.Account{ID: 9, BusinessID: 77, Kind: "asset", Name: "Efectivo"})
	if err != nil {
		t.Fatalf("the row could not be mapped: %v", err)
	}

	if mapped.Counterparty != "" {
		t.Errorf("the counterparty is %q, want it absent", mapped.Counterparty)
	}
	if mapped.DueDate != nil {
		t.Errorf("the due date is %v, want it absent", mapped.DueDate)
	}
}

func TestTheRowOfAnAccountWithAnUnknownKindIsRefused(t *testing.T) {
	t.Parallel()

	_, err := accountFromRow(db.Account{Kind: "savings"})

	if !errors.Is(err, account.ErrUnknownKind) {
		t.Fatalf("the error is %v, want %v", err, account.ErrUnknownKind)
	}
}

func TestTheDateOfAnAccountTravelsBothWays(t *testing.T) {
	t.Parallel()

	if optionalDate(nil).Valid {
		t.Error("the absent date is a value")
	}

	due := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !optionalDate(&due).Valid {
		t.Error("the date of the debt is absent")
	}
	if date(pgtype.Date{}) != nil {
		t.Error("the absent date of the row is a value")
	}

	read := date(pgtype.Date{Time: due, Valid: true})
	if read == nil || !read.Equal(due) {
		t.Errorf("the date of the row is %v, want %v", read, due)
	}
}

func TestTheRefusalsOfTheDatabaseAreTranslatedForTheBusiness(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "a business that is already there",
			err:  theRefusal(uniqueViolation, businessesPrimaryKeyConstraint),
			want: business.ErrAlreadyExists,
		},
		{
			name: "a second personal business of the same owner",
			err:  theRefusal(uniqueViolation, businessesPersonalUniqueIndex),
			want: business.ErrAlreadyExists,
		},
		{
			name: "an owner that is not a person",
			err:  theRefusal(foreignKeyViolation, "businesses_owner_id_fkey"),
			want: business.ErrNotFound,
		},
		{
			name: "a row the table refuses",
			err:  theRefusal(checkViolation, "businesses_kind_check"),
			want: business.ErrInvalidValue,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := writeBusinessError(testCase.err)
			if !errors.Is(got, testCase.want) {
				t.Fatalf("the error is %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestTheRefusalsOfTheDatabaseAreTranslatedForTheAccount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "an account whose name is taken",
			err:  theRefusal(uniqueViolation, accountsNameUniqueIndex),
			want: account.ErrAlreadyExists,
		},
		{
			name: "an account of a business that is not there",
			err:  theRefusal(foreignKeyViolation, "accounts_business_id_fkey"),
			want: account.ErrNotFound,
		},
		{
			name: "an account the table refuses",
			err:  theRefusal(checkViolation, "accounts_debt_check"),
			want: account.ErrInvalidValue,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := writeAccountError(testCase.err)
			if !errors.Is(got, testCase.want) {
				t.Fatalf("the error is %v, want %v", got, testCase.want)
			}
		})
	}
}

// theBusiness is the business the tests of the adapter write.
func theBusiness() business.Business {
	built, err := business.New(user.ID(1), business.KindPersonal, business.DefaultPersonalName, business.DefaultCurrency)
	if err != nil {
		panic("the business of the test could not be built: " + err.Error())
	}

	return built
}

// theAccount is the account the tests of the adapter write.
func theAccount() account.Account {
	built, err := account.New(business.ID(1), account.KindAsset, "Efectivo", "", nil)
	if err != nil {
		panic("the account of the test could not be built: " + err.Error())
	}

	return built
}

func TestARowThatIsNotThereIsNotFound(t *testing.T) {
	t.Parallel()

	if !errors.Is(writeBusinessError(pgxErrNoRows()), business.ErrNotFound) {
		t.Error("the error of the business is not the error of a row that is not there")
	}
	if !errors.Is(writeAccountError(pgxErrNoRows()), account.ErrNotFound) {
		t.Error("the error of the account is not the error of a row that is not there")
	}
}

// pgxErrNoRows is the error the driver returns when a query reads no row.
func pgxErrNoRows() error {
	return pgx.ErrNoRows
}
