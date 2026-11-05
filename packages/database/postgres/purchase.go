package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

// The names the database gives to the constraints of the table of the
// drafts.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	purchaseDraftsPrimaryKeyConstraint = "purchase_drafts_pkey"
	purchaseDraftsUtteranceConstraint  = "purchase_drafts_utterance_not_empty_check"
)

// PurchaseRepository is the adapter of the storage of the drafts.
//
// The lines of a draft, what it still needs, what it was not sure about
// and what PACA filled by herself travel as JSONB: they are the reading of
// a sentence and not the rows of a fact. The adapter encodes them, and the
// domain reads them back through its own constructors: the row never
// decides what a draft is.
type PurchaseRepository struct {
	pool *Pool
}

// NewPurchaseRepository returns the repository of the drafts, bound to the
// pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewPurchaseRepository(pool *Pool) (*PurchaseRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &PurchaseRepository{pool: pool}, nil
}

// Save writes a draft and answers it with the identity that the database
// generated.
//
// A draft without an identity is written for the first time, and a draft
// with one replaces the row it names: the amendment of a draft is the same
// draft, one revision further.
//
// The error is purchase.ErrInvalidValue when the row breaks a rule of the
// table, and purchase.ErrNotFound when the business of the draft is not
// there.
func (r *PurchaseRepository) Save(ctx context.Context, draft purchase.Draft) (purchase.Draft, error) {
	if r == nil || r.pool == nil {
		return purchase.Draft{}, purchase.ErrNoStorage
	}

	lines, err := encodeLines(draft.Lines)
	if err != nil {
		return purchase.Draft{}, err
	}

	missing, err := encodeList(draft.Missing)
	if err != nil {
		return purchase.Draft{}, err
	}

	ambiguous, err := encodeAmbiguities(draft.Ambiguous)
	if err != nil {
		return purchase.Draft{}, err
	}

	inferred, err := encodeList(draft.Inferred)
	if err != nil {
		return purchase.Draft{}, err
	}

	revision, err := revisionOf(draft.Revision)
	if err != nil {
		return purchase.Draft{}, err
	}

	values := db.CreatePurchaseDraftParams{
		BusinessID: int64(draft.BusinessID),
		State:      draft.State.String(),
		Revision:   revision,
		Utterance:  draft.Utterance,
		Lines:      lines,
		Missing:    missing,
		Ambiguous:  ambiguous,
		Inferred:   inferred,
		Day:        dateOf(draft.Day),
		Market:     optionalText(draft.Market),
		Invoice:    optionalText(draft.Invoice),
		ProposedOn: dateOf(draft.ProposedOn),
		ExpiresOn:  dateOf(draft.ExpiresOn),
	}

	if draft.ID == 0 {
		row, err := r.pool.queriesFor(ctx).CreatePurchaseDraft(ctx, values)
		if err != nil {
			return purchase.Draft{}, writePurchaseError(err)
		}

		return purchaseDraftFromRow(row)
	}

	row, err := r.pool.queriesFor(ctx).UpdatePurchaseDraft(ctx, db.UpdatePurchaseDraftParams{
		State:      values.State,
		Revision:   values.Revision,
		Utterance:  values.Utterance,
		Lines:      values.Lines,
		Missing:    values.Missing,
		Ambiguous:  values.Ambiguous,
		Inferred:   values.Inferred,
		Day:        values.Day,
		Market:     values.Market,
		Invoice:    values.Invoice,
		ProposedOn: values.ProposedOn,
		ExpiresOn:  values.ExpiresOn,
		ID:         int64(draft.ID),
	})
	if err != nil {
		return purchase.Draft{}, writePurchaseError(err)
	}

	return purchaseDraftFromRow(row)
}

// Find returns the draft of the identity.
//
// The error is purchase.ErrNotFound when no draft carries that identity.
func (r *PurchaseRepository) Find(ctx context.Context, id purchase.DraftID) (purchase.Draft, error) {
	if r == nil || r.pool == nil {
		return purchase.Draft{}, purchase.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).GetPurchaseDraftByID(ctx, int64(id))
	if err != nil {
		return purchase.Draft{}, writePurchaseError(err)
	}

	return purchaseDraftFromRow(row)
}

// Confirm moves the draft to the confirmed state and answers the draft.
//
// The row must be the one the owner saw: the revision is the one she saw,
// and the state is the one of a draft that waits for the yes of its owner.
// The table holds the rule as a defense in depth, and the adapter says so
// with the errors of the port.
//
// The error is purchase.ErrStaleRevision when the draft is not the one the
// owner saw, and purchase.ErrNoStorage when the storage is not open.
func (r *PurchaseRepository) Confirm(ctx context.Context, id purchase.DraftID, revision int) (purchase.Draft, error) {
	if r == nil || r.pool == nil {
		return purchase.Draft{}, purchase.ErrNoStorage
	}

	revision32, err := revisionOf(revision)
	if err != nil {
		return purchase.Draft{}, err
	}

	row, err := r.pool.queriesFor(ctx).ConfirmPurchaseDraft(ctx, db.ConfirmPurchaseDraftParams{
		State:         purchase.StateConfirmed.String(),
		ID:            int64(id),
		Revision:      revision32,
		ExpectedState: purchase.StateProposed.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return purchase.Draft{}, fmt.Errorf("%w: the draft %d is not the one the owner saw",
				purchase.ErrStaleRevision, id)
		}

		return purchase.Draft{}, writePurchaseError(err)
	}

	return purchaseDraftFromRow(row)
}

// PurchaseRecorder is the adapter of the storage of the purchases.
//
// A purchase is a fact: it is written once, and it never changes. Its
// lines are rows of their own table, and the adapter writes them with it:
// a purchase without its lines is not a purchase.
type PurchaseRecorder struct {
	pool *Pool
}

// NewPurchaseRecorder returns the recorder of the purchases, bound to the
// pool.
//
// The recorder needs an open pool, and the constructor refuses anything
// else: a recorder built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil.
//
// The error is ErrNoPool when the pool is not there.
func NewPurchaseRecorder(pool *Pool) (*PurchaseRecorder, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &PurchaseRecorder{pool: pool}, nil
}

// Record writes the purchase and its lines, and answers the purchase with
// the identity that the database generated.
//
// The error is purchase.ErrInvalidValue when the row breaks a rule of the
// table, purchase.ErrNotFound when the business or the draft of the
// purchase is not there, and purchase.ErrNoStorage when the storage is not
// open.
func (r *PurchaseRecorder) Record(ctx context.Context, written purchase.Purchase) (purchase.Purchase, error) {
	if r == nil || r.pool == nil {
		return purchase.Purchase{}, purchase.ErrNoStorage
	}

	row, err := r.pool.queriesFor(ctx).CreatePurchase(ctx, db.CreatePurchaseParams{
		BusinessID:    int64(written.BusinessID),
		DraftID:       int64(written.DraftID),
		Day:           dateOf(written.Day),
		Market:        optionalText(written.Market),
		Invoice:       optionalText(written.Invoice),
		Utterance:     written.Utterance,
		ConfirmedWith: written.ConfirmedWith,
		Total:         int64(written.Total),
	})
	if err != nil {
		return purchase.Purchase{}, writePurchaseError(err)
	}

	recorded := purchase.Purchase{
		ID:            purchase.ID(row.ID),
		DraftID:       purchase.DraftID(row.DraftID),
		BusinessID:    business.ID(row.BusinessID),
		Day:           ledger.NewDate(row.Day.Time),
		Market:        row.Market.String,
		Invoice:       row.Invoice.String,
		Utterance:     row.Utterance,
		ConfirmedWith: row.ConfirmedWith,
		Total:         ledger.Amount(row.Total),
		Lines:         make([]purchase.Line, 0, len(written.Lines)),
		CreatedAt:     timestamp(row.CreatedAt),
	}

	for position, one := range written.Lines {
		line, err := r.pool.queriesFor(ctx).CreatePurchaseLine(ctx, db.CreatePurchaseLineParams{
			PurchaseID:   row.ID,
			Position:     int32(position + 1),
			Name:         one.Name,
			Quantity:     one.Measure.Quantity,
			Unit:         one.Measure.Unit,
			Total:        int64(one.Price.Total),
			UnitPrice:    int64(one.Price.Unit),
			IngredientID: ingredientID(one.IngredientID),
		})
		if err != nil {
			return purchase.Purchase{}, writePurchaseError(err)
		}

		recorded.Lines = append(recorded.Lines, purchaseLineFromRow(line))
	}

	return recorded, nil
}

// purchaseLineFromRow maps a row of the lines of a purchase onto the model
// of the domain.
//
// The identity of the ingredient is empty when the catalog did not know the
// name: the line keeps the name the owner said.
func purchaseLineFromRow(row db.PurchaseLine) purchase.Line {
	line := purchase.Line{
		Name:    row.Name,
		Measure: purchase.Measure{Quantity: row.Quantity, Unit: row.Unit},
		Price: purchase.Price{
			Total: ledger.Amount(row.Total),
			Unit:  ledger.Amount(row.UnitPrice),
		},
	}

	if row.IngredientID.Valid {
		line.IngredientID = ingredient.ID(row.IngredientID.Int64)
	}

	return line
}

// ingredientID answers the identity of the ingredient of a line as the
// column of the table holds it.
//
// The column is optional: a line of a name the catalog does not know
// carries nothing, and the name it carries is the name the owner said.
func ingredientID(id ingredient.ID) pgtype.Int8 {
	if id == 0 {
		return pgtype.Int8{}
	}

	return pgtype.Int8{Int64: int64(id), Valid: true}
}

// purchaseDraftFromRow maps a row of the drafts table onto the model of the
// domain.
//
// The lines of the row are read by the domain: the row may come from a
// table that was changed by another writer, and the model does not trust a
// stranger with a line it does not know.
func purchaseDraftFromRow(row db.PurchaseDraft) (purchase.Draft, error) {
	lines, err := decodeLines(row.Lines)
	if err != nil {
		return purchase.Draft{}, err
	}

	missing, err := decodeList(row.Missing)
	if err != nil {
		return purchase.Draft{}, err
	}

	ambiguous, err := decodeAmbiguities(row.Ambiguous)
	if err != nil {
		return purchase.Draft{}, err
	}

	inferred, err := decodeList(row.Inferred)
	if err != nil {
		return purchase.Draft{}, err
	}

	return purchase.Draft{
		ID:         purchase.DraftID(row.ID),
		BusinessID: business.ID(row.BusinessID),
		State:      purchase.State(row.State),
		Revision:   int(row.Revision),
		Utterance:  row.Utterance,
		Lines:      lines,
		Missing:    missing,
		Ambiguous:  ambiguous,
		Inferred:   inferred,
		Day:        ledger.NewDate(row.Day.Time),
		Market:     row.Market.String,
		Invoice:    row.Invoice.String,
		ProposedOn: ledger.NewDate(row.ProposedOn.Time),
		ExpiresOn:  ledger.NewDate(row.ExpiresOn.Time),
		CreatedAt:  timestamp(row.CreatedAt),
		UpdatedAt:  timestamp(row.UpdatedAt),
	}, nil
}

// lineJSON is the shape of a line of a draft in the storage.
//
// The adapter encodes the model of the domain and decodes it: the shape of
// the row is a fact of the adapter, and the model never sees it.
type lineJSON struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Quantity is how much of it the owner bought.
	Quantity float64 `json:"quantity"`

	// Unit is the unit of that quantity.
	Unit string `json:"unit"`

	// Total is the money that was paid for the whole line.
	Total int64 `json:"total_paid"`

	// UnitPrice is the price of one unit of the line.
	UnitPrice int64 `json:"unit_price"`

	// Inferred says that the price was filled by the agent.
	Inferred bool `json:"price_inferred"`

	// IngredientID is the identity of the ingredient of the catalog the
	// name belongs to. It is zero when the catalog does not know the name.
	IngredientID int64 `json:"ingredient_id"`
}

// ambiguityJSON is the shape of an ambiguity of a draft in the storage.
type ambiguityJSON struct {
	// Name is the name as the owner said it.
	Name string `json:"name"`

	// Options are the candidates the agent found.
	Options []string `json:"options"`
}

// encodeLines encodes the lines of a draft for the storage.
func encodeLines(lines []purchase.Line) ([]byte, error) {
	encoded := make([]lineJSON, 0, len(lines))
	for _, one := range lines {
		encoded = append(encoded, lineJSON{
			Name:         one.Name,
			Quantity:     one.Measure.Quantity,
			Unit:         one.Measure.Unit,
			Total:        int64(one.Price.Total),
			UnitPrice:    int64(one.Price.Unit),
			Inferred:     one.Price.Inferred,
			IngredientID: int64(one.IngredientID),
		})
	}

	return json.Marshal(encoded)
}

// decodeLines decodes the lines of a draft of the storage.
//
// The lines are built by the domain: the row never decides what a line is.
func decodeLines(raw []byte) ([]purchase.Line, error) {
	if len(raw) == 0 {
		return []purchase.Line{}, nil
	}

	var decoded []lineJSON
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%w: the lines of the draft are not a list: %w", purchase.ErrInvalidValue, err)
	}

	lines := make([]purchase.Line, 0, len(decoded))
	for _, one := range decoded {
		measure, err := purchase.NewMeasure(one.Quantity, one.Unit)
		if err != nil {
			return nil, err
		}

		line, err := purchase.NewLine(one.Name, measure, purchase.Price{
			Total:    ledger.Amount(one.Total),
			Unit:     ledger.Amount(one.UnitPrice),
			Inferred: one.Inferred,
		})
		if err != nil {
			return nil, err
		}

		lines = append(lines, line.WithIngredient(ingredient.ID(one.IngredientID)))
	}

	return lines, nil
}

// encodeAmbiguities encodes the ambiguities of a draft for the storage.
func encodeAmbiguities(ambiguities []purchase.Ambiguity) ([]byte, error) {
	encoded := make([]ambiguityJSON, 0, len(ambiguities))
	for _, one := range ambiguities {
		encoded = append(encoded, ambiguityJSON{Name: one.Name, Options: one.Options})
	}

	return json.Marshal(encoded)
}

// decodeAmbiguities decodes the ambiguities of a draft of the storage.
//
// The ambiguities are built by the domain: an ambiguity without options is
// refused there, and not here.
func decodeAmbiguities(raw []byte) ([]purchase.Ambiguity, error) {
	if len(raw) == 0 {
		return []purchase.Ambiguity{}, nil
	}

	var decoded []ambiguityJSON
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%w: the ambiguities of the draft are not a list: %w", purchase.ErrInvalidValue, err)
	}

	ambiguities := make([]purchase.Ambiguity, 0, len(decoded))
	for _, one := range decoded {
		ambiguity, err := purchase.NewAmbiguity(one.Name, one.Options)
		if err != nil {
			return nil, err
		}

		ambiguities = append(ambiguities, ambiguity)
	}

	return ambiguities, nil
}

// encodeList encodes a list of things the draft carries for the storage.
func encodeList(values []string) ([]byte, error) {
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("%w: the list of the draft could not be encoded: %w", purchase.ErrInvalidValue, err)
	}

	return encoded, nil
}

// decodeList decodes a list of things the draft carries from the storage.
func decodeList(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}

	var decoded []string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%w: the list of the draft is not a list: %w", purchase.ErrInvalidValue, err)
	}

	return decoded, nil
}

// revisionOf returns the revision of a draft as the column of the table
// holds it.
//
// The column is an integer of thirty-two bits, and the revision of a draft
// is the number of the amendments it received: a revision that does not
// fit the column is a revision the table cannot hold.
func revisionOf(revision int) (int32, error) {
	if revision <= 0 || revision > math.MaxInt32 {
		return 0, fmt.Errorf("%w: the revision of the draft is %d", purchase.ErrInvalidValue, revision)
	}

	return int32(revision), nil
}

// writePurchaseError translates the truth of the database into the errors of
// the port of the drafts.
//
// The constraints of the table hold the rules of the domain as a defense in
// depth, and the business that is not there is the refusal of the foreign
// key.
func writePurchaseError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return purchase.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		if pgErr.ConstraintName == purchaseDraftsPrimaryKeyConstraint {
			return fmt.Errorf("%w: that draft is already there", purchase.ErrInvalidValue)
		}
	case foreignKeyViolation:
		return fmt.Errorf("%w: the business of the draft is not there", purchase.ErrNotFound)
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", purchase.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
