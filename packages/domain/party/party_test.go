package party

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/user"
)

// TestNewBuildsAParty proves that the domain builds the party of a
// business: the person, the store or the farm, with the roles it holds and
// with the facts that belong to it.
func TestNewBuildsAParty(t *testing.T) {
	t.Parallel()

	built, err := New(
		business.ID(77),
		KindPerson,
		"  Doña Rosalba  ",
		[]Role{RoleClient},
		" +57 321 253 9984 ",
		" Rosalba@Panaderia.COM ",
		" Calle 5 # 12-30 ",
		LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the party was refused: %v", err)
	}

	if built.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", built.BusinessID)
	}
	if built.Kind != KindPerson {
		t.Errorf("the kind is %q, want %q", built.Kind, KindPerson)
	}
	if built.Name != "Doña Rosalba" {
		t.Errorf("the name is %q, want %q", built.Name, "Doña Rosalba")
	}
	if len(built.Roles) != 1 || built.Roles[0] != RoleClient {
		t.Errorf("the roles are %v, want the one of the client", built.Roles)
	}
	if built.Phone != "573212539984" {
		t.Errorf("the phone is %q, want %q", built.Phone, "573212539984")
	}
	if built.Email != "rosalba@panaderia.com" {
		t.Errorf("the email is %q, want %q", built.Email, "rosalba@panaderia.com")
	}
	if built.Address != "Calle 5 # 12-30" {
		t.Errorf("the address is %q, want %q", built.Address, "Calle 5 # 12-30")
	}
	if built.LegalHuman != (LegalHuman{}) {
		t.Errorf("the legal human is %v, want none", built.LegalHuman)
	}
	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
}

// TestNewRefusesWhatTheDomainRefuses proves that the domain refuses the
// party that breaks its rules, and that it names the rule it broke.
func TestNewRefusesWhatTheDomainRefuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		kind  Kind
		party string
		roles []Role
		human LegalHuman
		want  error
	}{
		{
			name:  "an unknown kind",
			kind:  Kind("bakery"),
			party: "Doña Rosalba",
			roles: []Role{RoleClient},
			want:  ErrUnknownKind,
		},
		{
			name:  "an empty name",
			kind:  KindPerson,
			party: "   ",
			roles: []Role{RoleClient},
			want:  ErrInvalidName,
		},
		{
			name:  "no role at all",
			kind:  KindPerson,
			party: "Doña Rosalba",
			roles: nil,
			want:  ErrNoRole,
		},
		{
			name:  "an unknown role",
			kind:  KindPerson,
			party: "Doña Rosalba",
			roles: []Role{Role("owner")},
			want:  ErrUnknownRole,
		},
		{
			name:  "a store without the legal human that represents it",
			kind:  KindStore,
			party: "La Panadería de Don Jaime",
			roles: []Role{RoleClient},
			want:  ErrMissingLegalHuman,
		},
		{
			name:  "a person with a legal human",
			kind:  KindPerson,
			party: "Doña Rosalba",
			roles: []Role{RoleClient},
			human: LegalHuman{Name: "Rosalba"},
			want:  ErrUnexpectedLegalHuman,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(
				business.ID(77),
				testCase.kind,
				testCase.party,
				testCase.roles,
				"",
				"",
				"",
				testCase.human,
			)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

// TestTheStoreRequiresTheLegalHumanThatRepresentsIt proves the rule of the
// domain: a store is represented by a legal human, and a store without one
// is refused.
func TestTheStoreRequiresTheLegalHumanThatRepresentsIt(t *testing.T) {
	t.Parallel()

	built, err := New(
		business.ID(77),
		KindStore,
		"La Panadería de Don Jaime",
		[]Role{RoleClient, RoleProvider},
		"",
		"",
		"",
		LegalHuman{Name: "Jaime Ortiz", Figure: "S.A.S."},
	)
	if err != nil {
		t.Fatalf("the store was refused: %v", err)
	}

	if built.LegalHuman.Name != "Jaime Ortiz" {
		t.Errorf("the legal human is %q, want %q", built.LegalHuman.Name, "Jaime Ortiz")
	}
	if built.LegalHuman.Figure != "S.A.S." {
		t.Errorf("the legal figure is %q, want %q", built.LegalHuman.Figure, "S.A.S.")
	}
}

// TestTheFarmCarriesItsLegalFigureOnlyWhenItHasOne proves the rule of the
// domain: the farm may carry a legal figure, and it may carry none.
func TestTheFarmCarriesItsLegalFigureOnlyWhenItHasOne(t *testing.T) {
	t.Parallel()

	withoutFigure, err := New(
		business.ID(77),
		KindFarm,
		"Finca El Mirador",
		[]Role{RoleProvider},
		"",
		"",
		"",
		LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the farm was refused: %v", err)
	}
	if withoutFigure.LegalHuman != (LegalHuman{}) {
		t.Errorf("the legal human is %v, want none", withoutFigure.LegalHuman)
	}

	withFigure, err := New(
		business.ID(77),
		KindFarm,
		"Finca El Mirador",
		[]Role{RoleProvider},
		"",
		"",
		"",
		LegalHuman{Name: "María Gómez", Figure: "Ltda."},
	)
	if err != nil {
		t.Fatalf("the farm was refused: %v", err)
	}
	if withFigure.LegalHuman.Figure != "Ltda." {
		t.Errorf("the legal figure is %q, want %q", withFigure.LegalHuman.Figure, "Ltda.")
	}
}

// TestThePersonRefusesTheLegalHuman proves the rule of the domain: a
// person is a person, and it is never represented by another human.
func TestThePersonRefusesTheLegalHuman(t *testing.T) {
	t.Parallel()

	_, err := New(
		business.ID(77),
		KindPerson,
		"Doña Rosalba",
		[]Role{RoleClient},
		"",
		"",
		"",
		LegalHuman{Name: "Rosalba", Figure: "S.A.S."},
	)
	if !errors.Is(err, ErrUnexpectedLegalHuman) {
		t.Fatalf("the error is %v, want %v", err, ErrUnexpectedLegalHuman)
	}
}

// TestTheLegalFigureNeedsTheNameOfItsHuman proves the rule of the domain: a
// legal figure without the name of the human it belongs to is a figure of
// nobody.
func TestTheLegalFigureNeedsTheNameOfItsHuman(t *testing.T) {
	t.Parallel()

	_, err := NewLegalHuman("  ", "S.A.S.")
	if !errors.Is(err, ErrMissingLegalHuman) {
		t.Fatalf("the error is %v, want %v", err, ErrMissingLegalHuman)
	}

	built, err := NewLegalHuman(" Jaime Ortiz ", " S.A.S. ")
	if err != nil {
		t.Fatalf("the legal human was refused: %v", err)
	}
	if built.Name != "Jaime Ortiz" {
		t.Errorf("the name is %q, want %q", built.Name, "Jaime Ortiz")
	}
	if built.Figure != "S.A.S." {
		t.Errorf("the legal figure is %q, want %q", built.Figure, "S.A.S.")
	}
}

// TestThePartyHoldsBothRoles proves the rule of the domain: a party may
// hold the role of the client and the role of the provider at the same
// time.
func TestThePartyHoldsBothRoles(t *testing.T) {
	t.Parallel()

	built, err := New(
		business.ID(77),
		KindStore,
		"La Panadería de Don Jaime",
		[]Role{RoleClient, RoleProvider},
		"",
		"",
		"",
		LegalHuman{Name: "Jaime Ortiz"},
	)
	if err != nil {
		t.Fatalf("the party was refused: %v", err)
	}

	if len(built.Roles) != 2 {
		t.Fatalf("the party holds %d roles, want 2", len(built.Roles))
	}
	if !built.Is(RoleClient) {
		t.Error("the party does not hold the role of the client")
	}
	if !built.Is(RoleProvider) {
		t.Error("the party does not hold the role of the provider")
	}
	if built.Is(Role("owner")) {
		t.Error("the party holds a role it does not have")
	}
}

// TestTheRoleThatIsSaidTwiceIsOneRole proves the rule of the domain: a
// role that is said twice is one role.
func TestTheRoleThatIsSaidTwiceIsOneRole(t *testing.T) {
	t.Parallel()

	built, err := New(
		business.ID(77),
		KindPerson,
		"Doña Rosalba",
		[]Role{RoleClient, RoleClient},
		"",
		"",
		"",
		LegalHuman{},
	)
	if err != nil {
		t.Fatalf("the party was refused: %v", err)
	}

	if len(built.Roles) != 1 {
		t.Errorf("the party holds %d roles, want 1", len(built.Roles))
	}
}

// TestTheWayOfReachingThePartyFollowsTheRulesOfThePerson proves that the
// party reuses the rules of the contact of a person, and that it never
// writes a second rule of its own.
func TestTheWayOfReachingThePartyFollowsTheRulesOfThePerson(t *testing.T) {
	t.Parallel()

	_, err := New(
		business.ID(77),
		KindPerson,
		"Doña Rosalba",
		[]Role{RoleClient},
		"123",
		"",
		"",
		LegalHuman{},
	)
	if !errors.Is(err, user.ErrInvalidPhone) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidPhone)
	}

	_, err = New(
		business.ID(77),
		KindPerson,
		"Doña Rosalba",
		[]Role{RoleClient},
		"",
		"not an address",
		"",
		LegalHuman{},
	)
	if !errors.Is(err, user.ErrInvalidEmail) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidEmail)
	}
}

// TestThePartiesBelongToABusinessOfItsOwn proves the rule of the domain:
// the parties belong to a business of its own, and the Personal Business of
// a person holds the money of that person and not the clients and the
// providers of the bakery.
func TestThePartiesBelongToABusinessOfItsOwn(t *testing.T) {
	t.Parallel()

	if err := OwnedBy(business.KindBusiness); err != nil {
		t.Fatalf("the business of its own was refused: %v", err)
	}

	if err := OwnedBy(business.KindPersonal); !errors.Is(err, ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, ErrWrongBusiness)
	}
}

// TestParseKind proves that the kind of a party is read from the name the
// storage holds, and that an unknown name is refused.
func TestParseKind(t *testing.T) {
	t.Parallel()

	kind, err := ParseKind("store")
	if err != nil {
		t.Fatalf("the kind was refused: %v", err)
	}
	if kind != KindStore {
		t.Errorf("the kind is %q, want %q", kind, KindStore)
	}

	if _, err := ParseKind("bakery"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownKind)
	}
}

// TestKindValid proves that the kinds of a party are the three of them.
func TestKindValid(t *testing.T) {
	t.Parallel()

	for _, kind := range AllKinds() {
		if !kind.Valid() {
			t.Errorf("the kind %q is not valid", kind)
		}
	}

	if Kind("bakery").Valid() {
		t.Error("an unknown kind is valid")
	}
}

// TestKindString proves that the name of a kind is the name the storage
// holds.
func TestKindString(t *testing.T) {
	t.Parallel()

	if KindPerson.String() != "person" {
		t.Errorf("the name of the kind is %q, want %q", KindPerson.String(), "person")
	}
}

// TestAllKindsNamesEveryKindOfAParty proves that the list of the kinds
// names the three of them, in the order they are presented.
func TestAllKindsNamesEveryKindOfAParty(t *testing.T) {
	t.Parallel()

	all := AllKinds()
	if len(all) != 3 {
		t.Fatalf("the list holds %d kinds, want 3", len(all))
	}

	if all[0] != KindPerson {
		t.Errorf("the first kind is %q, want %q", all[0], KindPerson)
	}
	if all[1] != KindStore {
		t.Errorf("the second kind is %q, want %q", all[1], KindStore)
	}
	if all[2] != KindFarm {
		t.Errorf("the third kind is %q, want %q", all[2], KindFarm)
	}
}

// TestAllKindsReturnsACopy proves that the caller of the list never changes
// the vocabulary of the domain.
func TestAllKindsReturnsACopy(t *testing.T) {
	t.Parallel()

	all := AllKinds()
	all[0] = Kind("changed")

	if AllKinds()[0] != KindPerson {
		t.Error("the list changed under the caller")
	}
}

// TestErrorsNamesEveryError proves that the catalogue of the domain names
// every error it owns.
func TestErrorsNamesEveryError(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) != 10 {
		t.Fatalf("the catalogue holds %d errors, want 10", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}
