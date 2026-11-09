package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/party"
)

// fakeParties is the fake of the storage of the parties.
//
// It remembers the parties it holds and the ones it wrote, and it answers
// what the test tells it to answer.
type fakeParties struct {
	held    []party.Party
	written []party.Party
	err     error
	reads   int
	listed  business.ID
}

// Create writes the party and remembers it.
func (f *fakeParties) Create(_ context.Context, p party.Party) (party.Party, error) {
	if f.err != nil {
		return party.Party{}, f.err
	}

	p.ID = party.ID(len(f.written) + 100)
	f.written = append(f.written, p)

	return p, nil
}

// List answers the parties of the business, and remembers that it was
// asked.
func (f *fakeParties) List(_ context.Context, businessID business.ID) ([]party.Party, error) {
	f.reads++
	f.listed = businessID

	if f.err != nil {
		return nil, f.err
	}

	held := make([]party.Party, 0, len(f.held))
	for _, one := range f.held {
		if one.BusinessID == businessID {
			held = append(held, one)
		}
	}

	return held, nil
}

// theBakeryOfTheParties is the business that owns the parties of the tests.
func theBakeryOfTheParties() business.Business {
	return business.Business{
		ID:       business.ID(77),
		Kind:     business.KindBusiness,
		Name:     "La Panadería",
		Currency: "COP",
	}
}

// TestThePartyIsOpenedInTheBusinessThatKnowsIt proves that the use case
// writes the party in the business the agent names, with the facts the
// owner declared.
func TestThePartyIsOpenedInTheBusinessThatKnowsIt(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{}
	bakery := theBakeryOfTheParties()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewPartyService(parties, businesses)

	opened, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID:     77,
		Kind:           "person",
		Name:           "Doña Rosalba",
		Roles:          []string{"client"},
		Phone:          "+57 321 253 9984",
		Email:          "Rosalba@Panaderia.com",
		Address:        "Calle 5 # 12-30",
		LegalHumanName: "",
		LegalFigure:    "",
	})
	if err != nil {
		t.Fatalf("the party was refused: %v", err)
	}

	if businesses.found != business.ID(77) {
		t.Errorf("the business that was read is the one of %d, want 77", businesses.found)
	}
	if len(parties.written) != 1 {
		t.Fatalf("the storage holds %d parties, want 1", len(parties.written))
	}

	written := parties.written[0]
	if written.BusinessID != business.ID(77) {
		t.Errorf("the business of the party is %d, want 77", written.BusinessID)
	}
	if written.Kind != party.KindPerson {
		t.Errorf("the kind of the party is %q, want %q", written.Kind, party.KindPerson)
	}
	if written.Name != "Doña Rosalba" {
		t.Errorf("the name of the party is %q, want %q", written.Name, "Doña Rosalba")
	}
	if len(written.Roles) != 1 || written.Roles[0] != party.RoleClient {
		t.Errorf("the roles of the party are %v, want the client", written.Roles)
	}
	if written.Phone != "573212539984" {
		t.Errorf("the phone of the party is %q, want %q", written.Phone, "573212539984")
	}
	if written.Email != "rosalba@panaderia.com" {
		t.Errorf("the email of the party is %q, want %q", written.Email, "rosalba@panaderia.com")
	}
	if written.Address != "Calle 5 # 12-30" {
		t.Errorf("the address of the party is %q, want %q", written.Address, "Calle 5 # 12-30")
	}
	if opened.ID == 0 {
		t.Error("the party has no identity, want the one the storage generated")
	}
}

// TestThePartyIsRefusedWhenTheBusinessIsNotThere proves that the use case
// reads the business that owns the party, and that it writes nothing when
// there is none.
func TestThePartyIsRefusedWhenTheBusinessIsNotThere(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{}
	businesses := &fakeBusinesses{}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "person",
		Name:       "Doña Rosalba",
		Roles:      []string{"client"},
	})
	if !errors.Is(err, business.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, business.ErrNotFound)
	}
	if len(parties.written) != 0 {
		t.Errorf("the storage holds %d parties, want none", len(parties.written))
	}
}

// TestThePartyIsRefusedWhenTheBusinessIsThePersonalOne proves the rule of
// the domain: the parties belong to a business of its own, and the Personal
// Business of a person holds the money of that person and not the clients
// of the bakery.
func TestThePartyIsRefusedWhenTheBusinessIsThePersonalOne(t *testing.T) {
	t.Parallel()

	personal := theBakeryOfTheParties()
	personal.Kind = business.KindPersonal

	parties := &fakeParties{}
	businesses := &fakeBusinesses{existing: &personal}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "person",
		Name:       "Doña Rosalba",
		Roles:      []string{"client"},
	})
	if !errors.Is(err, party.ErrWrongBusiness) {
		t.Fatalf("the error is %v, want %v", err, party.ErrWrongBusiness)
	}
	if len(parties.written) != 0 {
		t.Errorf("the storage holds %d parties, want none", len(parties.written))
	}
}

// TestThePartyIsRefusedWhenTheDomainRefusesIt proves that the use case
// never decides what a party is: the domain refuses a store without the
// legal human that represents it, and nothing is written.
func TestThePartyIsRefusedWhenTheDomainRefusesIt(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{}
	bakery := theBakeryOfTheParties()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "store",
		Name:       "La Panadería de Don Jaime",
		Roles:      []string{"client"},
	})
	if !errors.Is(err, party.ErrMissingLegalHuman) {
		t.Fatalf("the error is %v, want %v", err, party.ErrMissingLegalHuman)
	}
	if len(parties.written) != 0 {
		t.Errorf("the storage holds %d parties, want none", len(parties.written))
	}
}

// TestThePartiesOfTheBusinessAreRead proves that the use case reads the
// parties of the business the agent names, and that it answers all of them:
// the ones that share a name among them.
func TestThePartiesOfTheBusinessAreRead(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{held: []party.Party{
		{ID: 10, BusinessID: 77, Kind: party.KindPerson, Name: "La Panadería", Roles: []party.Role{party.RoleClient}},
		{ID: 20, BusinessID: 77, Kind: party.KindStore, Name: "La Panadería", Roles: []party.Role{party.RoleClient}},
		{ID: 30, BusinessID: 88, Kind: party.KindPerson, Name: "Otro", Roles: []party.Role{party.RoleClient}},
	}}
	bakery := theBakeryOfTheParties()
	service := application.NewPartyService(parties, &fakeBusinesses{existing: &bakery})

	held, err := service.Read(t.Context(), ports.ReadParties{BusinessID: 77})
	if err != nil {
		t.Fatalf("the parties were refused: %v", err)
	}

	if parties.reads != 1 {
		t.Errorf("the storage was read %d times, want 1", parties.reads)
	}
	if parties.listed != business.ID(77) {
		t.Errorf("the parties that were read are the ones of %d, want 77", parties.listed)
	}
	if len(held) != 2 {
		t.Fatalf("the answer holds %d parties, want 2", len(held))
	}
	if held[0].ID != 10 || held[1].ID != 20 {
		t.Errorf("the parties of the answer are %d and %d, want 10 and 20", held[0].ID, held[1].ID)
	}
}

// TestThePartyIsRefusedWhenTheKindIsNotOneTheDomainKnows proves that the
// use case asks the domain to read the kind of the party, and that it
// writes nothing when the kind is not one of them.
func TestThePartyIsRefusedWhenTheKindIsNotOneTheDomainKnows(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{}
	bakery := theBakeryOfTheParties()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "bakery",
		Name:       "Doña Rosalba",
		Roles:      []string{"client"},
	})
	if !errors.Is(err, party.ErrUnknownKind) {
		t.Fatalf("the error is %v, want %v", err, party.ErrUnknownKind)
	}
	if len(parties.written) != 0 {
		t.Errorf("the storage holds %d parties, want none", len(parties.written))
	}
}

// TestThePartyIsRefusedWhenTheRoleIsNotOneTheDomainKnows proves that the
// use case asks the domain to read the roles of the party, and that it
// writes nothing when one of them is not a role.
func TestThePartyIsRefusedWhenTheRoleIsNotOneTheDomainKnows(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{}
	bakery := theBakeryOfTheParties()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "person",
		Name:       "Doña Rosalba",
		Roles:      []string{"owner"},
	})
	if !errors.Is(err, party.ErrUnknownRole) {
		t.Fatalf("the error is %v, want %v", err, party.ErrUnknownRole)
	}
	if len(parties.written) != 0 {
		t.Errorf("the storage holds %d parties, want none", len(parties.written))
	}
}

// TestThePartyIsRefusedWhenTheLegalFigureHasNoName proves that the use
// case asks the domain to build the legal human of the party, and that a
// legal figure without the name of its human is refused.
func TestThePartyIsRefusedWhenTheLegalFigureHasNoName(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{}
	bakery := theBakeryOfTheParties()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID:  77,
		Kind:        "farm",
		Name:        "Finca El Mirador",
		Roles:       []string{"provider"},
		LegalFigure: "S.A.S.",
	})
	if !errors.Is(err, party.ErrMissingLegalHuman) {
		t.Fatalf("the error is %v, want %v", err, party.ErrMissingLegalHuman)
	}
	if len(parties.written) != 0 {
		t.Errorf("the storage holds %d parties, want none", len(parties.written))
	}
}

// TestThePartyIsRefusedWhenTheStorageRefusesIt proves that the use case
// answers the error of the storage, and that it does not hide it.
func TestThePartyIsRefusedWhenTheStorageRefusesIt(t *testing.T) {
	t.Parallel()

	parties := &fakeParties{err: party.ErrInvalidValue}
	bakery := theBakeryOfTheParties()
	businesses := &fakeBusinesses{existing: &bakery}
	service := application.NewPartyService(parties, businesses)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "person",
		Name:       "Doña Rosalba",
		Roles:      []string{"client"},
	})
	if !errors.Is(err, party.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, party.ErrInvalidValue)
	}

	if _, err := service.Read(t.Context(), ports.ReadParties{BusinessID: 77}); !errors.Is(err, party.ErrInvalidValue) {
		t.Fatalf("the error is %v, want %v", err, party.ErrInvalidValue)
	}
}

// TestTheUseCaseOfThePartiesHasNoStorage proves that the service without a
// database says so, and that it writes nothing.
func TestTheUseCaseOfThePartiesHasNoStorage(t *testing.T) {
	t.Parallel()

	service := application.NewPartyService(nil, nil)

	_, err := service.Open(t.Context(), ports.OpenParty{
		BusinessID: 77,
		Kind:       "person",
		Name:       "Doña Rosalba",
		Roles:      []string{"client"},
	})
	if !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}

	if _, err := service.Read(t.Context(), ports.ReadParties{BusinessID: 77}); !errors.Is(err, ports.ErrNoDatabase) {
		t.Fatalf("the error is %v, want %v", err, ports.ErrNoDatabase)
	}
}
