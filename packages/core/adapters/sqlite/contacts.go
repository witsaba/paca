package sqlite

import (
	"context"
	"database/sql"

	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// ContactRepository is the SQLite implementation of the contact port.
//
// It follows the shape of the account repository: the same naming, the same
// rules about the errors, and the same reading of the optional columns.
type ContactRepository struct {
	db *DB
}

// NewContactRepository returns the contact repository over the given
// database.
func NewContactRepository(db *DB) *ContactRepository {
	return &ContactRepository{db: db}
}

// The repository implements the port of the application.
var _ ports.ContactRepository = (*ContactRepository)(nil)

// The columns of a contact, in the order the scanner reads them.
//
// The shape of a contact is decided by its kind, and the schema checks it: a
// person carries an identity document and no tax id, an entity carries a tax
// id and a legal name and no identity document. The columns are read as they
// are stored, and a contact that breaks the shape is refused by the schema
// itself.
const contactColumns = `id, kind, name, phone, id_document, tax_id, legal_name, notes, created_at`

// Create stores a new contact and returns it with its identity filled in.
func (r *ContactRepository) Create(ctx context.Context, contact domain.Contact) (domain.Contact, error) {
	var id int64
	var createdAt string

	err := r.db.QueryRowContext(ctx, `
		INSERT INTO contacts
			(kind, name, phone, id_document, tax_id, legal_name, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at`,
		contact.Kind, contact.Name, contact.Phone,
		optionalText(contact.IDDocument), optionalText(contact.TaxID),
		optionalText(contact.LegalName), optionalText(contact.Notes),
	).Scan(&id, &createdAt)
	if err != nil {
		return domain.Contact{}, err
	}

	contact.ID = id
	contact.CreatedAt = createdAt

	return contact, nil
}

// GetByID returns the contact with the given identity.
func (r *ContactRepository) GetByID(ctx context.Context, id int64) (domain.Contact, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+contactColumns+` FROM contacts WHERE id = ?`, id)

	contact, err := scanContact(row.Scan)
	if err != nil {
		return domain.Contact{}, notFound(err, "the contact does not exist")
	}

	return contact, nil
}

// List returns every contact, ordered by identity.
func (r *ContactRepository) List(ctx context.Context) ([]domain.Contact, error) {
	return r.list(ctx, `SELECT `+contactColumns+` FROM contacts ORDER BY id`)
}

// ListByKind returns the contacts of the given kind, ordered by identity.
func (r *ContactRepository) ListByKind(ctx context.Context, kind domain.ContactKind) ([]domain.Contact, error) {
	return r.list(ctx, `SELECT `+contactColumns+` FROM contacts WHERE kind = ? ORDER BY id`, kind)
}

// list runs a query over the contacts and reads every row it returns.
func (r *ContactRepository) list(ctx context.Context, query string, args ...any) ([]domain.Contact, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	contacts := []domain.Contact{}
	for rows.Next() {
		contact, err := scanContact(rows.Scan)
		if err != nil {
			return nil, err
		}

		contacts = append(contacts, contact)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contacts, nil
}

// scanContact reads one contact from a row.
//
// The optional columns of the schema are read as nullable, and a value that
// is not there is the empty string of the domain: the domain has no nulls,
// only empty values.
func scanContact(scan func(dest ...any) error) (domain.Contact, error) {
	var contact domain.Contact
	var idDocument, taxID, legalName, notes sql.NullString

	err := scan(
		&contact.ID, &contact.Kind, &contact.Name, &contact.Phone,
		&idDocument, &taxID, &legalName, &notes, &contact.CreatedAt,
	)
	if err != nil {
		return domain.Contact{}, err
	}

	contact.IDDocument = idDocument.String
	contact.TaxID = taxID.String
	contact.LegalName = legalName.String
	contact.Notes = notes.String

	return contact, nil
}
