package sqlite_test

import (
	"context"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
)

// personContact returns a person, ready to be stored.
func personContact(name string, idDocument string) domain.Contact {
	return domain.Contact{
		Kind:       domain.ContactKindPerson,
		Name:       name,
		Phone:      "3001112233",
		IDDocument: idDocument,
	}
}

// entityContact returns an entity, ready to be stored.
func entityContact(name string, taxID string) domain.Contact {
	return domain.Contact{
		Kind:      domain.ContactKindEntity,
		Name:      name,
		Phone:     "6015554433",
		TaxID:     taxID,
		LegalName: name + " S.A.S.",
		Notes:     "A regular supplier",
	}
}

// A stored person comes back with its identity, its creation moment and its
// fields.
func TestContactRepositoryStoresAndReadsAPerson(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewContactRepository(db)

	stored, err := repository.Create(ctx, personContact("Ana Restrepo", "CC 12345678"))
	if err != nil {
		t.Fatalf("store the person: %v", err)
	}

	if stored.ID == 0 {
		t.Error("the stored person has no identity")
	}

	if stored.CreatedAt == "" {
		t.Error("the stored person has no creation moment")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the person: %v", err)
	}

	if read.Kind != domain.ContactKindPerson {
		t.Errorf("the kind of the contact is %q, want %q", read.Kind, domain.ContactKindPerson)
	}

	if read.Name != "Ana Restrepo" {
		t.Errorf("the name of the contact is %q, want %q", read.Name, "Ana Restrepo")
	}

	if read.Phone != "3001112233" {
		t.Errorf("the phone of the contact is %q, want %q", read.Phone, "3001112233")
	}

	if read.IDDocument != "CC 12345678" {
		t.Errorf("the identity document of the contact is %q, want %q", read.IDDocument, "CC 12345678")
	}

	if read.TaxID != "" || read.LegalName != "" {
		t.Errorf("the person carries the columns of an entity: %+v", read)
	}
}

// A stored entity comes back with its identity, its creation moment and its
// fields.
func TestContactRepositoryStoresAndReadsAnEntity(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewContactRepository(db)

	stored, err := repository.Create(ctx, entityContact("Ferretería Norte", "900123456-7"))
	if err != nil {
		t.Fatalf("store the entity: %v", err)
	}

	if stored.ID == 0 {
		t.Error("the stored entity has no identity")
	}

	if stored.CreatedAt == "" {
		t.Error("the stored entity has no creation moment")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the entity: %v", err)
	}

	if read.Kind != domain.ContactKindEntity {
		t.Errorf("the kind of the contact is %q, want %q", read.Kind, domain.ContactKindEntity)
	}

	if read.TaxID != "900123456-7" {
		t.Errorf("the tax id of the contact is %q, want %q", read.TaxID, "900123456-7")
	}

	if read.LegalName != "Ferretería Norte S.A.S." {
		t.Errorf("the legal name of the contact is %q, want %q", read.LegalName, "Ferretería Norte S.A.S.")
	}

	if read.Notes != "A regular supplier" {
		t.Errorf("the notes of the contact are %q, want %q", read.Notes, "A regular supplier")
	}

	if read.IDDocument != "" {
		t.Errorf("the entity carries the column of a person: %+v", read)
	}
}

// The contacts come back in the order of their identity.
func TestContactRepositoryListsTheContactsInOrder(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewContactRepository(db)

	first, err := repository.Create(ctx, personContact("First", "CC 11111111"))
	if err != nil {
		t.Fatalf("store the first contact: %v", err)
	}

	second, err := repository.Create(ctx, entityContact("Second", "900222222-7"))
	if err != nil {
		t.Fatalf("store the second contact: %v", err)
	}

	contacts, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list the contacts: %v", err)
	}

	if len(contacts) != 2 {
		t.Fatalf("the list holds %d contacts, want 2", len(contacts))
	}

	if contacts[0].ID != first.ID || contacts[1].ID != second.ID {
		t.Errorf("the list is not ordered by identity: %d, %d", contacts[0].ID, contacts[1].ID)
	}
}

// The list of a kind holds only the contacts of that kind.
func TestContactRepositoryListsTheContactsOfOneKind(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewContactRepository(db)

	if _, err := repository.Create(ctx, personContact("First person", "CC 11111111")); err != nil {
		t.Fatalf("store the first person: %v", err)
	}

	entity, err := repository.Create(ctx, entityContact("The entity", "900222222-7"))
	if err != nil {
		t.Fatalf("store the entity: %v", err)
	}

	people, err := repository.ListByKind(ctx, domain.ContactKindPerson)
	if err != nil {
		t.Fatalf("list the people: %v", err)
	}

	if len(people) != 1 {
		t.Fatalf("the list of the people holds %d contacts, want 1", len(people))
	}

	if people[0].Kind != domain.ContactKindPerson {
		t.Errorf("the kind of the contact is %q, want %q", people[0].Kind, domain.ContactKindPerson)
	}

	entities, err := repository.ListByKind(ctx, domain.ContactKindEntity)
	if err != nil {
		t.Fatalf("list the entities: %v", err)
	}

	if len(entities) != 1 {
		t.Fatalf("the list of the entities holds %d contacts, want 1", len(entities))
	}

	if entities[0].ID != entity.ID {
		t.Errorf("the entity of the list is %d, want %d", entities[0].ID, entity.ID)
	}

	all, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("list the contacts: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("the list of every contact holds %d contacts, want 2", len(all))
	}
}

// A contact that does not exist is a not-found failure.
func TestContactRepositoryReportsAMissingContact(t *testing.T) {
	db := openMigratedDB(t)

	repository := sqlite.NewContactRepository(db)

	_, err := repository.GetByID(context.Background(), 404)
	if err == nil {
		t.Fatal("the missing contact was returned, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// An entity is identified by its tax id, so a second one with the same tax
// id is refused as a validation failure.
func TestContactRepositoryRefusesADuplicateTaxID(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewContactRepository(db)

	if _, err := repository.Create(ctx, entityContact("First", "900123456-7")); err != nil {
		t.Fatalf("store the first entity: %v", err)
	}

	_, err := repository.Create(ctx, entityContact("Second", "900123456-7"))
	if err == nil {
		t.Fatal("the second entity was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}

	if strings.Contains(domain.DetailOf(err), "UNIQUE constraint failed") {
		t.Errorf("the detail leaks the message of the driver: %q", domain.DetailOf(err))
	}
}

// A person is identified by its identity document, so a person without one is
// refused as a validation failure.
func TestContactRepositoryRefusesAPersonWithoutAnIdentityDocument(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewContactRepository(db)

	_, err := repository.Create(ctx, personContact("Ana Restrepo", ""))
	if err == nil {
		t.Fatal("the person without an identity document was stored, want a refusal")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the refusal is %q, want %q", got, domain.CodeValidationFailed)
	}
}
