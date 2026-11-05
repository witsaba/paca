package postgres

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
)

func TestTheRepositoryOfTheDraftsNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewPurchaseRepository(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

func TestTheRepositoryOfTheDraftsWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	drafts := &PurchaseRepository{}

	if _, err := drafts.Save(t.Context(), purchase.Draft{}); !errors.Is(err, purchase.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}
	if _, err := drafts.Find(t.Context(), 1); !errors.Is(err, purchase.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}
}

// TestTheLinesOfTheDraftTravelBothWays proves that the lines of a draft
// survive the trip to the storage and back.
func TestTheLinesOfTheDraftTravelBothWays(t *testing.T) {
	t.Parallel()

	lines := []purchase.Line{
		{
			Name:    "Harina",
			Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
			Price:   purchase.Price{Total: 12000},
		},
		{
			Name:    "Pan",
			Measure: purchase.Measure{Quantity: 3, Unit: "unidad"},
		},
	}

	encoded, err := encodeLines(lines)
	if err != nil {
		t.Fatalf("the lines could not be encoded: %v", err)
	}

	decoded, err := decodeLines(encoded)
	if err != nil {
		t.Fatalf("the lines could not be decoded: %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("the storage answered %d lines, want 2", len(decoded))
	}
	if decoded[0].Name != "Harina" {
		t.Errorf("the name of the first line is %q, want %q", decoded[0].Name, "Harina")
	}
	if decoded[0].Measure.Quantity != 25 {
		t.Errorf("the quantity of the first line is %v, want 25", decoded[0].Measure.Quantity)
	}
	if decoded[0].Measure.Unit != "kg" {
		t.Errorf("the unit of the first line is %q, want %q", decoded[0].Measure.Unit, "kg")
	}
	if decoded[0].Price.Total != 12000 {
		t.Errorf("the money of the first line is %d, want 12000", decoded[0].Price.Total)
	}
	if decoded[1].Price.Present() {
		t.Errorf("the second line carries money: %+v", decoded[1].Price)
	}
}

// TestTheAmbiguitiesOfTheDraftTravelBothWays proves that the ambiguities
// of a draft survive the trip to the storage and back.
func TestTheAmbiguitiesOfTheDraftTravelBothWays(t *testing.T) {
	t.Parallel()

	ambiguities := []purchase.Ambiguity{
		{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}},
	}

	encoded, err := encodeAmbiguities(ambiguities)
	if err != nil {
		t.Fatalf("the ambiguities could not be encoded: %v", err)
	}

	decoded, err := decodeAmbiguities(encoded)
	if err != nil {
		t.Fatalf("the ambiguities could not be decoded: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("the storage answered %d ambiguities, want 1", len(decoded))
	}
	if decoded[0].Name != "harina" {
		t.Errorf("the name is %q, want %q", decoded[0].Name, "harina")
	}
	if len(decoded[0].Options) != 2 {
		t.Errorf("the options are %v, want the two of the test", decoded[0].Options)
	}
}

// TestTheListsOfTheDraftTravelBothWays proves that what the draft still
// needs and what PACA filled survive the trip to the storage and back.
func TestTheListsOfTheDraftTravelBothWays(t *testing.T) {
	t.Parallel()

	missing := []string{"the price of Pan"}
	inferred := []string{"the day of the purchase is the day of today"}

	encodedMissing, err := encodeList(missing)
	if err != nil {
		t.Fatalf("the list could not be encoded: %v", err)
	}
	encodedInferred, err := encodeList(inferred)
	if err != nil {
		t.Fatalf("the list could not be encoded: %v", err)
	}

	decodedMissing, err := decodeList(encodedMissing)
	if err != nil {
		t.Fatalf("the list could not be decoded: %v", err)
	}
	decodedInferred, err := decodeList(encodedInferred)
	if err != nil {
		t.Fatalf("the list could not be decoded: %v", err)
	}

	if len(decodedMissing) != 1 || decodedMissing[0] != "the price of Pan" {
		t.Errorf("the list is %v, want the price of the bread", decodedMissing)
	}
	if len(decodedInferred) != 1 || decodedInferred[0] != "the day of the purchase is the day of today" {
		t.Errorf("the list is %v, want the day of the purchase", decodedInferred)
	}
}

// TestTheRowOfADraftIsMappedOntoTheModel proves that the row of a draft is
// read as the model of the domain, with its lines and its three parts.
func TestTheRowOfADraftIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	proposed := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	expires := time.Date(2026, time.March, 22, 0, 0, 0, 0, time.UTC)
	moment := time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC)

	lines, err := encodeLines([]purchase.Line{
		{
			Name:    "Harina",
			Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
			Price:   purchase.Price{Total: 12000},
		},
	})
	if err != nil {
		t.Fatalf("the lines could not be encoded: %v", err)
	}

	missing, err := encodeList([]string{"the price of Pan"})
	if err != nil {
		t.Fatalf("the list could not be encoded: %v", err)
	}

	ambiguous, err := encodeAmbiguities([]purchase.Ambiguity{
		{Name: "harina", Options: []string{"Harina panadera"}},
	})
	if err != nil {
		t.Fatalf("the ambiguities could not be encoded: %v", err)
	}

	inferred, err := encodeList([]string{"the day of the purchase is the day of today"})
	if err != nil {
		t.Fatalf("the list could not be encoded: %v", err)
	}

	mapped, err := purchaseDraftFromRow(db.PurchaseDraft{
		ID:         42,
		BusinessID: 77,
		State:      "proposed",
		Revision:   2,
		Utterance:  "Compré 25 kilos de harina",
		Lines:      lines,
		Missing:    missing,
		Ambiguous:  ambiguous,
		Inferred:   inferred,
		Day:        pgtype.Date{Time: proposed, Valid: true},
		Market:     pgtype.Text{String: "Plaza de Mercado", Valid: true},
		Invoice:    pgtype.Text{String: "FV-001", Valid: true},
		ProposedOn: pgtype.Date{Time: proposed, Valid: true},
		ExpiresOn:  pgtype.Date{Time: expires, Valid: true},
		CreatedAt:  pgtype.Timestamptz{Time: moment, Valid: true},
		UpdatedAt:  pgtype.Timestamptz{Time: moment, Valid: true},
	})
	if err != nil {
		t.Fatalf("the row could not be mapped: %v", err)
	}

	if mapped.ID != purchase.DraftID(42) {
		t.Errorf("the identity is %d, want 42", mapped.ID)
	}
	if mapped.BusinessID != business.ID(77) {
		t.Errorf("the business is %d, want 77", mapped.BusinessID)
	}
	if mapped.State != purchase.StateProposed {
		t.Errorf("the state is %q, want %q", mapped.State, purchase.StateProposed)
	}
	if mapped.Revision != 2 {
		t.Errorf("the revision is %d, want 2", mapped.Revision)
	}
	if mapped.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", mapped.Utterance, "Compré 25 kilos de harina")
	}
	if len(mapped.Lines) != 1 || mapped.Lines[0].Price.Total != 12000 {
		t.Errorf("the lines of the draft are %v, want the line of the flour", mapped.Lines)
	}
	if len(mapped.Missing) != 1 {
		t.Errorf("the draft needs %v, want the price of the bread", mapped.Missing)
	}
	if len(mapped.Ambiguous) != 1 {
		t.Errorf("the draft holds %v, want the ambiguity of the flour", mapped.Ambiguous)
	}
	if len(mapped.Inferred) != 1 {
		t.Errorf("PACA filled %v, want the day of the purchase", mapped.Inferred)
	}
	if mapped.Day != ledger.NewDate(proposed) {
		t.Errorf("the day is %v, want %v", mapped.Day, proposed)
	}
	if mapped.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", mapped.Market, "Plaza de Mercado")
	}
	if mapped.Invoice != "FV-001" {
		t.Errorf("the invoice is %q, want %q", mapped.Invoice, "FV-001")
	}
	if mapped.ExpiresOn != ledger.NewDate(expires) {
		t.Errorf("the day of the expiry is %v, want %v", mapped.ExpiresOn, expires)
	}
	if !mapped.CreatedAt.Equal(moment) {
		t.Errorf("the moment of the creation is %v, want %v", mapped.CreatedAt, moment)
	}
}

// TestTheRowOfADraftWithBrokenLinesIsRefused proves that the rules of the
// domain reach the row: a row whose line has no unit is refused.
func TestTheRowOfADraftWithBrokenLinesIsRefused(t *testing.T) {
	t.Parallel()

	lines, err := encodeLines([]purchase.Line{
		{
			Name:    "Harina",
			Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
			Price:   purchase.Price{Total: 12000},
		},
	})
	if err != nil {
		t.Fatalf("the lines could not be encoded: %v", err)
	}

	broken := []byte(`[{"name":"Harina","quantity":25,"unit":"","total_paid":12000}]`)

	_, err = purchaseDraftFromRow(db.PurchaseDraft{Lines: broken})
	if !errors.Is(err, purchase.ErrMissingUnit) {
		t.Fatalf("the error is %v, want %v", err, purchase.ErrMissingUnit)
	}

	_, err = purchaseDraftFromRow(db.PurchaseDraft{Lines: lines})
	if err != nil {
		t.Fatalf("the row could not be mapped: %v", err)
	}
}

// TestTheRevisionOfTheDraftIsTheOneTheColumnHolds proves that the
// revision of a draft is the number of the column of the table, and that
// a revision that does not fit it is refused.
func TestTheRevisionOfTheDraftIsTheOneTheColumnHolds(t *testing.T) {
	t.Parallel()

	revision, err := revisionOf(2)
	if err != nil {
		t.Fatalf("the revision was refused: %v", err)
	}
	if revision != 2 {
		t.Errorf("the revision is %d, want 2", revision)
	}

	if _, err := revisionOf(0); !errors.Is(err, purchase.ErrInvalidValue) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrInvalidValue)
	}
	if _, err := revisionOf(math.MaxInt32 + 1); !errors.Is(err, purchase.ErrInvalidValue) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrInvalidValue)
	}
}

// TestTheRefusalsOfTheDatabaseAreTranslatedForTheDraft proves that the
// truth of the database reaches the caller as the errors of the port.
func TestTheRefusalsOfTheDatabaseAreTranslatedForTheDraft(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "a draft that is already there",
			err:  theRefusal(uniqueViolation, purchaseDraftsPrimaryKeyConstraint),
			want: purchase.ErrInvalidValue,
		},
		{
			name: "a business that is not there",
			err:  theRefusal(foreignKeyViolation, "purchase_drafts_business_id_fkey"),
			want: purchase.ErrNotFound,
		},
		{
			name: "a row the table refuses",
			err:  theRefusal(checkViolation, purchaseDraftsUtteranceConstraint),
			want: purchase.ErrInvalidValue,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := writePurchaseError(testCase.err)
			if !errors.Is(got, testCase.want) {
				t.Fatalf("the error is %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestADraftThatIsNotThereIsNotFound proves that a row that is not there is
// a draft that is not there.
func TestADraftThatIsNotThereIsNotFound(t *testing.T) {
	t.Parallel()

	if !errors.Is(writePurchaseError(pgxErrNoRows()), purchase.ErrNotFound) {
		t.Error("the error of the draft is not the error of a row that is not there")
	}
}

// TestTheRecorderOfThePurchasesNeedsAPool proves that the recorder of the
// purchases refuses to be built without a pool: a recorder built on nothing
// would be a nil pointer inside an interface, and the interface would not
// be nil.
func TestTheRecorderOfThePurchasesNeedsAPool(t *testing.T) {
	t.Parallel()

	if _, err := NewPurchaseRecorder(nil); !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
}

// TestTheRecorderOfThePurchasesWithoutADatabaseRefusesToWork proves that
// the recorder of the purchases says that it has no storage.
func TestTheRecorderOfThePurchasesWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	recorder := &PurchaseRecorder{}

	if _, err := recorder.Record(t.Context(), purchase.Purchase{}); !errors.Is(err, purchase.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}
}

// TestTheDraftsWithoutADatabaseRefuseToConfirm proves that the drafts say
// that they have no storage.
func TestTheDraftsWithoutADatabaseRefuseToConfirm(t *testing.T) {
	t.Parallel()

	drafts := &PurchaseRepository{}

	if _, err := drafts.Confirm(t.Context(), 1, 1); !errors.Is(err, purchase.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNoStorage)
	}
}

// TestTheRowOfALineOfThePurchaseIsMappedOntoTheModel proves that the row of
// a line of a purchase is read as the model of the domain.
func TestTheRowOfALineOfThePurchaseIsMappedOntoTheModel(t *testing.T) {
	t.Parallel()

	mapped := purchaseLineFromRow(db.PurchaseLine{
		ID:         3,
		PurchaseID: 2,
		Position:   1,
		Name:       "Harina",
		Quantity:   25,
		Unit:       "kg",
		Total:      12000,
		UnitPrice:  480,
	})

	if mapped.Name != "Harina" {
		t.Errorf("the name is %q, want %q", mapped.Name, "Harina")
	}
	if mapped.Measure.Quantity != 25 {
		t.Errorf("the quantity is %v, want 25", mapped.Measure.Quantity)
	}
	if mapped.Measure.Unit != "kg" {
		t.Errorf("the unit is %q, want %q", mapped.Measure.Unit, "kg")
	}
	if mapped.Price.Total != 12000 {
		t.Errorf("the money of the line is %d, want 12000", mapped.Price.Total)
	}
	if mapped.Price.Unit != 480 {
		t.Errorf("the price of the unit is %d, want 480", mapped.Price.Unit)
	}
}
