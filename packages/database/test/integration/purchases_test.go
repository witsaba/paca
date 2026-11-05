//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/user"
)

// The codes of the errors of PostgreSQL the assertions of the drafts
// expect.
const (
	// foreignKeyViolation is the code of a row that breaks a foreign key.
	foreignKeyViolation = "23503"
)

// draftsRepository returns the repository of the drafts, bound to the
// database of the suite.
func draftsRepository(t *testing.T) *postgres.PurchaseRepository {
	t.Helper()

	repo, err := postgres.NewPurchaseRepository(theSuite.pool)
	if err != nil {
		t.Fatalf("the repository of the drafts could not be created: %v", err)
	}

	return repo
}

// theBakeryOfTheTest opens the business that owns the drafts of the test.
func theBakeryOfTheTest(t *testing.T, ctx context.Context) business.Business {
	t.Helper()

	owner := nextIdentifier()
	registerUser(t, usersRepository(t), owner, "The Owner Of The Drafts")

	created, err := businessesRepository(t).Create(ctx, user.ID(owner), businessOf(business.KindBusiness, "La Panadería"))
	if err != nil {
		t.Fatalf("the bakery could not be created: %v", err)
	}

	return created
}

// theDraftOfTheTest builds the draft the tests of the table write.
func theDraftOfTheTest(bakery business.Business) purchase.Draft {
	proposed := ledger.NewDate(time.Date(2026, time.March, 15, 9, 0, 0, 0, time.UTC))

	return purchase.Draft{
		BusinessID: bakery.ID,
		State:      purchase.StateProposed,
		Revision:   1,
		Utterance:  "Compré 25 kilos de harina",
		Lines: []purchase.Line{
			{
				Name:    "Harina",
				Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
				Price:   purchase.Price{Total: 12000},
			},
		},
		Missing:    []string{"the price of Pan"},
		Ambiguous:  []purchase.Ambiguity{{Name: "harina", Options: []string{"Harina panadera", "Harina de trigo"}}},
		Inferred:   []string{"the day of the purchase is the day of today"},
		Day:        proposed,
		Market:     "Plaza de Mercado",
		Invoice:    "FV-001",
		ProposedOn: proposed,
		ExpiresOn:  ledger.NewDate(proposed.Time().AddDate(0, 0, purchase.DraftLifetimeDays)),
	}
}

// TestTheDraftOfTheBusinessIsKeptAndReadBack proves the rules of the table
// against a real database: the draft of a business is written with its
// lines, with what it still needs, with what it was not sure about and
// with what PACA filled, and it is read back as it was written.
func TestTheDraftOfTheBusinessIsKeptAndReadBack(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)

	written, err := repo.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the draft has no identity, want the one the database generated")
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the draft is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.State != purchase.StateProposed {
		t.Errorf("the state is %q, want %q", written.State, purchase.StateProposed)
	}
	if written.Revision != 1 {
		t.Errorf("the revision is %d, want 1", written.Revision)
	}
	if written.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", written.Utterance, "Compré 25 kilos de harina")
	}
	if len(written.Lines) != 1 {
		t.Fatalf("the draft holds %d lines, want 1", len(written.Lines))
	}
	if written.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", written.Lines[0].Name, "Harina")
	}
	if written.Lines[0].Measure.Quantity != 25 || written.Lines[0].Measure.Unit != "kg" {
		t.Errorf("the measure of the line is %+v, want 25 kg", written.Lines[0].Measure)
	}
	if written.Lines[0].Price.Total != 12000 {
		t.Errorf("the money of the line is %d, want 12000", written.Lines[0].Price.Total)
	}
	if len(written.Missing) != 1 {
		t.Errorf("the draft needs %v, want the price of the bread", written.Missing)
	}
	if len(written.Ambiguous) != 1 {
		t.Errorf("the draft holds %v, want the ambiguity of the flour", written.Ambiguous)
	}
	if len(written.Inferred) != 1 {
		t.Errorf("PACA filled %v, want the day of the purchase", written.Inferred)
	}
	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day is %v, want 2026-03-15", written.Day)
	}
	if written.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", written.Market, "Plaza de Mercado")
	}
	if written.Invoice != "FV-001" {
		t.Errorf("the invoice is %q, want %q", written.Invoice, "FV-001")
	}
	if written.ExpiresOn.String() != "2026-03-22" {
		t.Errorf("the day of the expiry is %v, want 2026-03-22", written.ExpiresOn)
	}

	found, err := repo.Find(ctx, written.ID)
	if err != nil {
		t.Fatalf("the draft could not be read: %v", err)
	}

	if found.ID != written.ID {
		t.Errorf("the identity of the draft is %d, want %d", found.ID, written.ID)
	}
	if len(found.Lines) != 1 || found.Lines[0].Price.Total != 12000 {
		t.Errorf("the lines of the draft are %v, want the line of the flour", found.Lines)
	}
	if len(found.Ambiguous) != 1 || len(found.Ambiguous[0].Options) != 2 {
		t.Errorf("the ambiguities of the draft are %v, want the two options", found.Ambiguous)
	}
}

// TestTheAmendmentOfTheDraftIsTheSameDraft proves that the amendment of a
// draft is the same row with a revision further: the sentence the owner
// said the first time is kept, and the day of the expiry does not move.
func TestTheAmendmentOfTheDraftIsTheSameDraft(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)

	written, err := repo.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	amended := written
	amended.Revision = 2
	amended.Lines = []purchase.Line{
		{
			Name:    "Harina",
			Measure: purchase.Measure{Quantity: 30, Unit: "kg"},
			Price:   purchase.Price{Total: 14400},
		},
	}

	updated, err := repo.Save(ctx, amended)
	if err != nil {
		t.Fatalf("the amendment could not be written: %v", err)
	}

	if updated.ID != written.ID {
		t.Errorf("the amended draft is %d, want %d", updated.ID, written.ID)
	}
	if updated.Revision != 2 {
		t.Errorf("the revision is %d, want 2", updated.Revision)
	}
	if updated.Utterance != written.Utterance {
		t.Errorf("the sentence is %q, want %q", updated.Utterance, written.Utterance)
	}
	if updated.ExpiresOn != written.ExpiresOn {
		t.Errorf("the day of the expiry is %v, want %v", updated.ExpiresOn, written.ExpiresOn)
	}
	if updated.Lines[0].Price.Total != 14400 {
		t.Errorf("the money of the line is %d, want 14400", updated.Lines[0].Price.Total)
	}

	var count int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM purchase_drafts WHERE id = $1`, int64(written.ID),
	).Scan(&count); err != nil {
		t.Fatalf("the drafts could not be counted: %v", err)
	}
	if count != 1 {
		t.Errorf("the database holds %d drafts, want 1", count)
	}
}

// TestTheTableRefusesTheDraftWithoutASentence proves the rule of the domain
// against a real database: the sentence of the owner is not empty.
func TestTheTableRefusesTheDraftWithoutASentence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheTest(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO purchase_drafts (business_id, utterance, proposed_on, expires_on) VALUES ($1, $2, $3, $4)`,
		int64(bakery.ID), "", "2026-03-15", "2026-03-22",
	)
	if err == nil {
		t.Fatal("the database stored a draft without a sentence, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "purchase_drafts_utterance_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchase_drafts_utterance_not_empty_check")
	}
}

// TestTheTableRefusesTheDraftOfABusinessThatIsNotThere proves that the
// draft belongs to a business of the storage.
func TestTheTableRefusesTheDraftOfABusinessThatIsNotThere(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheTest(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO purchase_drafts (business_id, utterance, proposed_on, expires_on) VALUES ($1, $2, $3, $4)`,
		int64(bakery.ID)+1000, "Compré harina", "2026-03-15", "2026-03-22",
	)
	if err == nil {
		t.Fatal("the database stored a draft of a business that is not there, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != foreignKeyViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, foreignKeyViolation)
	}
	if pgErr.ConstraintName != "purchase_drafts_business_id_fkey" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchase_drafts_business_id_fkey")
	}
}

// TestTheTableRefusesTheDraftThatExpiresBeforeItWasProposed proves the
// rule of the domain against a real database: a draft expires seven days
// after the day it was proposed, and never before it.
func TestTheTableRefusesTheDraftThatExpiresBeforeItWasProposed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheTest(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO purchase_drafts (business_id, utterance, proposed_on, expires_on) VALUES ($1, $2, $3, $4)`,
		int64(bakery.ID), "Compré harina", "2026-03-15", "2026-03-14",
	)
	if err == nil {
		t.Fatal("the database stored a draft that expires before it was proposed, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "purchase_drafts_expiry_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchase_drafts_expiry_check")
	}
}

// TestTheDraftThatIsNotThereIsNotFound proves that a draft that does not
// exist is not found.
func TestTheDraftThatIsNotThereIsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := draftsRepository(t)

	if _, err := repo.Find(ctx, purchase.DraftID(999_999_999)); err == nil {
		t.Fatal("the draft was found, want the refusal of the storage")
	}
}

// purchasesRecorder returns the recorder of the purchases, bound to the
// database of the suite.
func purchasesRecorder(t *testing.T) *postgres.PurchaseRecorder {
	t.Helper()

	recorder, err := postgres.NewPurchaseRecorder(theSuite.pool)
	if err != nil {
		t.Fatalf("the recorder of the purchases could not be created: %v", err)
	}

	return recorder
}

// thePurchaseOfTheTest builds the purchase the tests of the table write.
func thePurchaseOfTheTest(draft purchase.Draft) purchase.Purchase {
	return purchase.Purchase{
		DraftID:       draft.ID,
		BusinessID:    draft.BusinessID,
		Day:           draft.Day,
		Market:        draft.Market,
		Invoice:       draft.Invoice,
		Utterance:     draft.Utterance,
		ConfirmedWith: "sí",
		Lines: []purchase.Line{
			{
				Name:    "Harina",
				Measure: purchase.Measure{Quantity: 25, Unit: "kg"},
				Price:   purchase.Price{Total: 12000, Unit: 480},
			},
		},
		Total: 12000,
	}
}

// TestThePurchaseOfTheDraftIsWrittenWithItsLines proves the rules of the
// table against a real database: the purchase is written with its lines,
// with the sentence of the owner and with the word of the confirmation, and
// it is read back as it was written.
func TestThePurchaseOfTheDraftIsWrittenWithItsLines(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	recorder := purchasesRecorder(t)
	bakery := theBakeryOfTheTest(t, ctx)

	draft, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	written, err := recorder.Record(ctx, thePurchaseOfTheTest(draft))
	if err != nil {
		t.Fatalf("the purchase could not be written: %v", err)
	}

	if written.ID == 0 {
		t.Error("the purchase has no identity, want the one the database generated")
	}
	if written.DraftID != draft.ID {
		t.Errorf("the draft of the purchase is %d, want %d", written.DraftID, draft.ID)
	}
	if written.BusinessID != bakery.ID {
		t.Errorf("the business of the purchase is %d, want %d", written.BusinessID, bakery.ID)
	}
	if written.Utterance != "Compré 25 kilos de harina" {
		t.Errorf("the sentence is %q, want %q", written.Utterance, "Compré 25 kilos de harina")
	}
	if written.ConfirmedWith != "sí" {
		t.Errorf("the word of the confirmation is %q, want %q", written.ConfirmedWith, "sí")
	}
	if written.Total != 12000 {
		t.Errorf("the money of the purchase is %d, want 12000", written.Total)
	}
	if written.Day.String() != "2026-03-15" {
		t.Errorf("the day is %v, want 2026-03-15", written.Day)
	}
	if written.Market != "Plaza de Mercado" {
		t.Errorf("the market is %q, want %q", written.Market, "Plaza de Mercado")
	}
	if written.Invoice != "FV-001" {
		t.Errorf("the invoice is %q, want %q", written.Invoice, "FV-001")
	}
	if len(written.Lines) != 1 {
		t.Fatalf("the purchase holds %d lines, want 1", len(written.Lines))
	}
	if written.Lines[0].Name != "Harina" {
		t.Errorf("the name of the line is %q, want %q", written.Lines[0].Name, "Harina")
	}
	if written.Lines[0].Measure.Quantity != 25 || written.Lines[0].Measure.Unit != "kg" {
		t.Errorf("the measure of the line is %+v, want 25 kg", written.Lines[0].Measure)
	}
	if written.Lines[0].Price.Total != 12000 {
		t.Errorf("the money of the line is %d, want 12000", written.Lines[0].Price.Total)
	}
	if written.Lines[0].Price.Unit != 480 {
		t.Errorf("the price of the unit is %d, want 480", written.Lines[0].Price.Unit)
	}

	var lines int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM purchase_lines WHERE purchase_id = $1`, int64(written.ID),
	).Scan(&lines); err != nil {
		t.Fatalf("the lines could not be counted: %v", err)
	}
	if lines != 1 {
		t.Errorf("the database holds %d lines, want 1", lines)
	}
}

// TestTheTableRefusesThePurchaseWithoutASentence proves the rule of the
// domain against a real database: the sentence of the owner is not empty.
func TestTheTableRefusesThePurchaseWithoutASentence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)

	draft, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO purchases (business_id, draft_id, day, utterance, confirmed_with, total)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		int64(bakery.ID), int64(draft.ID), "2026-03-15", "", "sí", int64(12000),
	)
	if err == nil {
		t.Fatal("the database stored a purchase without a sentence, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "purchases_utterance_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchases_utterance_not_empty_check")
	}
}

// TestTheTableRefusesThePurchaseWithoutTheWordOfTheConfirmation proves the
// rule of the domain against a real database: the word the owner used to
// confirm is kept with the record, and it is not empty.
func TestTheTableRefusesThePurchaseWithoutTheWordOfTheConfirmation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)

	draft, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO purchases (business_id, draft_id, day, utterance, confirmed_with, total)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		int64(bakery.ID), int64(draft.ID), "2026-03-15", "Compré harina", "", int64(12000),
	)
	if err == nil {
		t.Fatal("the database stored a purchase without a word, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.ConstraintName != "purchases_confirmed_with_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchases_confirmed_with_not_empty_check")
	}
}

// TestTheTableRefusesTheSecondPurchaseOfTheSameDraft proves the rule of the
// domain against a real database: a draft is confirmed once, and a second
// purchase of it is refused.
func TestTheTableRefusesTheSecondPurchaseOfTheSameDraft(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	recorder := purchasesRecorder(t)
	bakery := theBakeryOfTheTest(t, ctx)

	draft, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	if _, err := recorder.Record(ctx, thePurchaseOfTheTest(draft)); err != nil {
		t.Fatalf("the purchase could not be written: %v", err)
	}

	_, err = recorder.Record(ctx, thePurchaseOfTheTest(draft))
	if err == nil {
		t.Fatal("the database stored a second purchase of the same draft, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != uniqueViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, uniqueViolation)
	}
	if pgErr.ConstraintName != "purchases_one_per_draft" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchases_one_per_draft")
	}
}

// TestTheTableRefusesTheLineWithoutAName proves the rule of the domain
// against a real database: the name of a line is the name the person read.
func TestTheTableRefusesTheLineWithoutAName(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	recorder := purchasesRecorder(t)
	bakery := theBakeryOfTheTest(t, ctx)

	draft, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	written, err := recorder.Record(ctx, thePurchaseOfTheTest(draft))
	if err != nil {
		t.Fatalf("the purchase could not be written: %v", err)
	}

	_, err = theSuite.db.ExecContext(ctx,
		`INSERT INTO purchase_lines (purchase_id, position, name, quantity, unit, total, unit_price)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		int64(written.ID), 2, "", 3.0, "unidad", int64(1500), int64(500),
	)
	if err == nil {
		t.Fatal("the database stored a line without a name, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.ConstraintName != "purchase_lines_name_not_empty_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchase_lines_name_not_empty_check")
	}
}

// TestTheDraftIsConfirmedAndItNeverWaitsAgain proves the rule of the domain
// against a real database: the confirmation moves the draft to the state of
// the confirmed, and a draft that changed under the eyes of the owner is
// refused.
func TestTheDraftIsConfirmedAndItNeverWaitsAgain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)

	written, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	confirmed, err := drafts.Confirm(ctx, written.ID, written.Revision)
	if err != nil {
		t.Fatalf("the draft could not be confirmed: %v", err)
	}
	if confirmed.State != purchase.StateConfirmed {
		t.Errorf("the state of the draft is %q, want %q", confirmed.State, purchase.StateConfirmed)
	}

	found, err := drafts.Find(ctx, written.ID)
	if err != nil {
		t.Fatalf("the draft could not be read: %v", err)
	}
	if found.State != purchase.StateConfirmed {
		t.Errorf("the state of the draft is %q, want %q", found.State, purchase.StateConfirmed)
	}

	_, err = drafts.Confirm(ctx, written.ID, written.Revision)
	if !errors.Is(err, purchase.ErrStaleRevision) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrStaleRevision)
	}
}

// TestTheDraftIsCancelledAndItNeverWaitsAgain proves the rule of the state
// machine against a real database: the draft moves to the state of the
// cancelled, and a draft that does not wait any more is not cancelled
// again.
func TestTheDraftIsCancelledAndItNeverWaitsAgain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)

	written, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	cancelled, err := drafts.Cancel(ctx, written.ID)
	if err != nil {
		t.Fatalf("the draft could not be cancelled: %v", err)
	}
	if cancelled.State != purchase.StateCancelled {
		t.Errorf("the state of the draft is %q, want %q", cancelled.State, purchase.StateCancelled)
	}
	if cancelled.ID != written.ID {
		t.Errorf("the identity of the draft is %d, want %d", cancelled.ID, written.ID)
	}

	found, err := drafts.Find(ctx, written.ID)
	if err != nil {
		t.Fatalf("the draft could not be read: %v", err)
	}
	if found.State != purchase.StateCancelled {
		t.Errorf("the state of the draft is %q, want %q", found.State, purchase.StateCancelled)
	}

	_, err = drafts.Cancel(ctx, written.ID)
	if !errors.Is(err, purchase.ErrNotCancellable) {
		t.Errorf("the error is %v, want %v", err, purchase.ErrNotCancellable)
	}
}

// TestThePendingDraftsOfTheBusinessAreRead proves the rule of the domain
// against a real database: the drafts of a business are read back in the
// order of their identity, and the drafts of another business and the
// drafts that ended are not among them.
func TestThePendingDraftsOfTheBusinessAreRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	drafts := draftsRepository(t)
	bakery := theBakeryOfTheTest(t, ctx)
	other := theBakeryOfTheTest(t, ctx)

	first, err := drafts.Save(ctx, theDraftOfTheTest(bakery))
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	second := theDraftOfTheTest(bakery)
	second.Utterance = "Compré 3 kilos de pan"
	written, err := drafts.Save(ctx, second)
	if err != nil {
		t.Fatalf("the draft could not be written: %v", err)
	}

	otherDraft := theDraftOfTheTest(other)
	if _, err := drafts.Save(ctx, otherDraft); err != nil {
		t.Fatalf("the draft of the other business could not be written: %v", err)
	}

	if _, err := drafts.Cancel(ctx, written.ID); err != nil {
		t.Fatalf("the draft could not be cancelled: %v", err)
	}

	held, err := drafts.List(ctx, bakery.ID)
	if err != nil {
		t.Fatalf("the drafts could not be read: %v", err)
	}

	if len(held) != 1 {
		t.Fatalf("the business holds %d pending drafts, want 1", len(held))
	}
	if held[0].ID != first.ID {
		t.Errorf("the pending draft is %d, want %d", held[0].ID, first.ID)
	}
	if held[0].BusinessID != bakery.ID {
		t.Errorf("the business of the draft is %d, want %d", held[0].BusinessID, bakery.ID)
	}
}

// TestTheTableRefusesTheStateThatIsNotOneOfTheDraft proves the rule of the
// domain against a real database: the states of a draft are the ones the
// domain knows, and the table holds them as a defense in depth.
func TestTheTableRefusesTheStateThatIsNotOneOfTheDraft(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheTest(t, ctx)

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO purchase_drafts (business_id, state, utterance, proposed_on, expires_on)
		 VALUES ($1, $2, $3, $4, $5)`,
		int64(bakery.ID), "dropped", "Compré harina", "2026-03-15", "2026-03-22",
	)
	if err == nil {
		t.Fatal("the database stored a draft in a state that is not one of the domain, want the row refused")
	}

	pgErr := postgresError(t, err)
	if pgErr.Code != checkViolation {
		t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
	}
	if pgErr.ConstraintName != "purchase_drafts_state_check" {
		t.Errorf("the constraint of the error is %q, want %q",
			pgErr.ConstraintName, "purchase_drafts_state_check")
	}
}

// TestTheTableHoldsTheStatesOfTheDraft proves the rule of the domain
// against a real database: the table holds the whole vocabulary of the
// states of a draft.
func TestTheTableHoldsTheStatesOfTheDraft(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bakery := theBakeryOfTheTest(t, ctx)

	for _, state := range purchase.States() {
		_, err := theSuite.db.ExecContext(ctx,
			`INSERT INTO purchase_drafts (business_id, state, utterance, proposed_on, expires_on)
			 VALUES ($1, $2, $3, $4, $5)`,
			int64(bakery.ID), state.String(), "Compré harina", "2026-03-15", "2026-03-22",
		)
		if err != nil {
			t.Errorf("the database refused the state %q: %v", state.String(), err)
		}
	}
}
