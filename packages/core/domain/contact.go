package domain

// ContactKind is the kind of a contact: a person or an entity.
type ContactKind string

const (
	// ContactKindPerson is a natural person, identified by their identity
	// document.
	ContactKindPerson ContactKind = "person"

	// ContactKindEntity is a company or another entity, identified by its
	// tax id.
	ContactKindEntity ContactKind = "entity"
)

// Contact is a person or an entity that takes part in a transaction.
//
// A person carries an identity document and no tax id; an entity carries a
// tax id and a legal name and no identity document. The identity document of
// a person is not unique: the same person may appear twice, once as a
// service provider and once as a lender.
type Contact struct {
	// ID is the identity of the contact. It is zero before the contact is
	// stored.
	ID int64

	// Kind is the kind of the contact.
	Kind ContactKind

	// Name is the name of the contact.
	Name string

	// Phone is the phone number of the contact.
	Phone string

	// IDDocument is the identity document of a person.
	IDDocument string

	// TaxID is the tax id of an entity.
	TaxID string

	// LegalName is the legal name of an entity.
	LegalName string

	// Notes is the free text of the contact.
	Notes string

	// CreatedAt is the moment the contact was stored, as an ISO-8601
	// timestamp in UTC.
	CreatedAt string
}
