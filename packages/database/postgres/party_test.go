package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/party"
)

func TestTheRepositoryOfThePartiesNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewPartyRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfThePartiesWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	parties := &PartyRepository{}

	if _, err := parties.List(t.Context(), business.ID(77)); !errors.Is(err, party.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, party.ErrNoStorage)
	}
	if _, err := parties.Create(t.Context(), party.Party{}); !errors.Is(err, party.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, party.ErrNoStorage)
	}
}

// TestTheRolesOfAPartyTravelBothWays proves that the roles the party holds
// survive the trip to the storage and back.
func TestTheRolesOfAPartyTravelBothWays(t *testing.T) {
	t.Parallel()

	encoded, err := encodeRoles([]party.Role{party.RoleClient, party.RoleProvider})
	if err != nil {
		t.Fatalf("the roles could not be encoded: %v", err)
	}

	decoded, err := decodeRoles(encoded)
	if err != nil {
		t.Fatalf("the roles could not be decoded: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("the storage answered %d roles, want 2", len(decoded))
	}
	if decoded[0] != party.RoleClient || decoded[1] != party.RoleProvider {
		t.Errorf("the roles are %v, want the client and the provider", decoded)
	}
}

// TestTheRolesThatAreNotAListAreRefused proves that a row that another
// writer broke is refused, and not read as a list of nothing.
func TestTheRolesThatAreNotAListAreRefused(t *testing.T) {
	t.Parallel()

	if _, err := decodeRoles([]byte("not a list")); !errors.Is(err, party.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, party.ErrInvalidValue)
	}
}

// TestTheRoleThatTheDomainDoesNotKnowIsRefused proves that the roles are
// built by the domain: the row never decides what a role is.
func TestTheRoleThatTheDomainDoesNotKnowIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := decodeRoles([]byte(`["owner"]`)); !errors.Is(err, party.ErrUnknownRole) {
		t.Fatalf("the error is %v, want %v", err, party.ErrUnknownRole)
	}
}

// TestTheRowOfAPartyIsMappedOntoTheModel proves that the row is mapped
// onto the model of the domain, and that the model decides what a party is.
func TestTheRowOfAPartyIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped, err := partyFromRow(db.Party{
		ID:             10,
		BusinessID:     77,
		Kind:           "store",
		Name:           "La Panadería de Don Jaime",
		Roles:          []byte(`["client","provider"]`),
		Phone:          pgtype.Text{String: "573212539984", Valid: true},
		Email:          pgtype.Text{String: "jaime@panaderia.com", Valid: true},
		Address:        pgtype.Text{String: "Calle 5 # 12-30", Valid: true},
		LegalHumanName: pgtype.Text{String: "Jaime Ortiz", Valid: true},
		LegalFigure:    pgtype.Text{String: "S.A.S.", Valid: true},
	})
	if err != nil {
		t.Fatalf("the row was refused: %v", err)
	}

	if mapped.ID != party.ID(10) {
		t.Errorf("the identity is %d, want 10", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.Kind != party.KindStore {
		t.Errorf("the kind is %q, want %q", mapped.Kind, party.KindStore)
	}
	if !mapped.Is(party.RoleClient) || !mapped.Is(party.RoleProvider) {
		t.Errorf("the roles are %v, want the client and the provider", mapped.Roles)
	}
	if mapped.Phone != "573212539984" {
		t.Errorf("the phone is %q, want %q", mapped.Phone, "573212539984")
	}
	if mapped.Email != "jaime@panaderia.com" {
		t.Errorf("the email is %q, want %q", mapped.Email, "jaime@panaderia.com")
	}
	if mapped.Address != "Calle 5 # 12-30" {
		t.Errorf("the address is %q, want %q", mapped.Address, "Calle 5 # 12-30")
	}
	if mapped.LegalHuman.Name != "Jaime Ortiz" {
		t.Errorf("the legal human is %q, want %q", mapped.LegalHuman.Name, "Jaime Ortiz")
	}
	if mapped.LegalHuman.Figure != "S.A.S." {
		t.Errorf("the legal figure is %q, want %q", mapped.LegalHuman.Figure, "S.A.S.")
	}
}

// TestTheRowOfAPartyThatBreaksTheRuleOfTheDomainIsRefused proves that the
// model refuses a row that another writer broke: the kind, the roles and
// the legal human are rules of the party.
func TestTheRowOfAPartyThatBreaksTheRuleOfTheDomainIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		row  db.Party
		want error
	}{
		{
			name: "the kind the domain does not know",
			row: db.Party{
				BusinessID: 77,
				Kind:       "bakery",
				Name:       "La Panadería",
				Roles:      []byte(`["client"]`),
			},
			want: party.ErrUnknownKind,
		},
		{
			name: "the party without a name",
			row: db.Party{
				BusinessID: 77,
				Kind:       "person",
				Name:       "  ",
				Roles:      []byte(`["client"]`),
			},
			want: party.ErrInvalidName,
		},
		{
			name: "the party without a role",
			row: db.Party{
				BusinessID: 77,
				Kind:       "person",
				Name:       "Doña Rosalba",
				Roles:      []byte(`[]`),
			},
			want: party.ErrNoRole,
		},
		{
			name: "the store without its legal human",
			row: db.Party{
				BusinessID: 77,
				Kind:       "store",
				Name:       "La Panadería de Don Jaime",
				Roles:      []byte(`["client"]`),
			},
			want: party.ErrMissingLegalHuman,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := partyFromRow(testCase.row)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

// TestTheRefusalsOfTheDatabaseAreTranslatedForTheParty proves that the
// truth of the database reaches the caller as the errors of the port.
func TestTheRefusalsOfTheDatabaseAreTranslatedForTheParty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     *pgconn.PgError
		want    error
		message string
	}{
		{
			name: "the business that is not there",
			err: &pgconn.PgError{
				Code:           foreignKeyViolation,
				ConstraintName: "parties_business_id_fkey",
			},
			want:    party.ErrNotFound,
			message: "the business of the party is not there",
		},
		{
			name: "the row that breaks a rule of the table",
			err: &pgconn.PgError{
				Code:           checkViolation,
				ConstraintName: "parties_legal_human_check",
			},
			want:    party.ErrInvalidValue,
			message: "refused the row",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := writePartyError(testCase.err)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
			if !strings.Contains(err.Error(), testCase.message) {
				t.Errorf("the error is %q, want it to say %q", err.Error(), testCase.message)
			}
		})
	}
}
