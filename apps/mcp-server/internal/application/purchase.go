package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/store"
)

// PurchaseService is the use case of the drafts of the utterance.
//
// The agent reads the sentence of the owner and proposes a candidate; the
// domain decides what may enter the book, and the use case keeps the
// candidate as a draft of the business.
//
// The use case never writes the book. A draft is a proposal, and it waits
// for the yes of the owner: a later increment turns a confirmed draft into
// the purchase.
//
// The use case also drops the drafts the owner does not want and reads the
// pending ones back, so that a lost conversation can be recovered.
type PurchaseService struct {
	drafts      purchase.Repository
	ingredients ingredient.Repository
	clock       ports.Clock
}

// NewPurchaseService builds the use case of the drafts.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The catalog of the
// names is one of them: the unit of a line the owner did not name is the
// unit of the ingredient. The clock is the port of the time: the day of a
// draft that names none is the day of today, and the service does not read
// the clock of the machine by itself.
func NewPurchaseService(
	drafts purchase.Repository,
	ingredients ingredient.Repository,
	clock ports.Clock,
) *PurchaseService {
	return &PurchaseService{drafts: drafts, ingredients: ingredients, clock: clock}
}

// Draft keeps the candidate the agent read as a draft of the business.
//
// The sentence of the owner travels verbatim, and the lines travel as the
// agent read them: the domain decides what may enter the book, and the
// use case never decides for it.
//
// The day of the purchase is the one the agent names, and the day of today
// when it names none: the day is one of the three things Paca may fill by
// herself. The money is never one of them, and the domain refuses a price
// that the agent filled.
//
// The candidate amends the draft the agent names, and it is a new draft
// when it names none. The amendment grows the revision of the draft and
// keeps the sentence the owner said the first time.
func (s *PurchaseService) Draft(ctx context.Context, request ports.DraftPurchase) (purchase.Draft, error) {
	if s == nil || s.drafts == nil || s.ingredients == nil || s.clock == nil {
		return purchase.Draft{}, ports.ErrNoDatabase
	}

	catalog, err := s.catalogOf(ctx, request.BusinessID)
	if err != nil {
		return purchase.Draft{}, err
	}

	candidate, err := s.candidateOf(request, catalog)
	if err != nil {
		return purchase.Draft{}, err
	}

	if request.DraftID != 0 {
		return s.amend(ctx, request, candidate)
	}

	draft, err := purchase.NewDraft(request.BusinessID, request.Utterance, candidate, s.today())
	if err != nil {
		return purchase.Draft{}, err
	}

	return s.drafts.Save(ctx, draft)
}

// amend applies the candidate to the draft the agent names.
//
// The draft belongs to its business: the amendment of a draft of another
// business is refused, and nothing is written.
func (s *PurchaseService) amend(
	ctx context.Context,
	request ports.DraftPurchase,
	candidate purchase.Candidate,
) (purchase.Draft, error) {
	existing, err := s.drafts.Find(ctx, request.DraftID)
	if err != nil {
		return purchase.Draft{}, err
	}

	if existing.BusinessID != request.BusinessID {
		return purchase.Draft{}, fmt.Errorf("%w: the draft %d is of the business %d",
			purchase.ErrWrongBusiness, existing.ID, existing.BusinessID)
	}

	amended, err := purchase.Amend(existing, candidate, s.today())
	if err != nil {
		return purchase.Draft{}, err
	}

	return s.drafts.Save(ctx, amended)
}

// Cancel drops the draft the owner does not want.
//
// The draft belongs to its business: the cancellation of a draft of another
// business is refused, and the draft stays as it was.
//
// The rules of the state machine are the ones of the domain: only a draft
// that waits is cancelled. A draft that is confirmed already, a draft that
// is cancelled already and a draft that expired are refused by the domain,
// and nothing is written.
func (s *PurchaseService) Cancel(ctx context.Context, request ports.CancelDraft) (purchase.Draft, error) {
	if s == nil || s.drafts == nil || s.clock == nil {
		return purchase.Draft{}, ports.ErrNoDatabase
	}

	existing, err := s.drafts.Find(ctx, request.DraftID)
	if err != nil {
		return purchase.Draft{}, err
	}

	if existing.BusinessID != request.BusinessID {
		return purchase.Draft{}, fmt.Errorf("%w: the draft %d is of the business %d",
			purchase.ErrWrongBusiness, existing.ID, existing.BusinessID)
	}

	cancelled, err := purchase.Cancel(existing, s.today())
	if err != nil {
		return purchase.Draft{}, err
	}

	return s.drafts.Cancel(ctx, cancelled.ID)
}

// Read answers the pending drafts of a business.
//
// The pending drafts are the ones that wait for the yes of the owner: the
// domain computes the expiry when the draft is read, and an expired draft
// does not wait any more.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *PurchaseService) Read(ctx context.Context, request ports.ReadDrafts) ([]purchase.Draft, error) {
	if s == nil || s.drafts == nil || s.clock == nil {
		return nil, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return nil, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	drafts, err := s.drafts.List(ctx, request.BusinessID)
	if err != nil {
		return nil, err
	}

	return purchase.Pending(drafts, day), nil
}

// catalogOf answers the catalog of the names the business buys.
//
// The catalog is what the use case asks when the owner names no unit for a
// line: the unit of the line is then the unit of the ingredient, and the
// domain still refuses a line that ends without a unit.
func (s *PurchaseService) catalogOf(ctx context.Context, own business.ID) (ingredient.Catalog, error) {
	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return ingredient.Catalog{}, err
	}

	return ingredient.NewCatalog(own, items)
}

// candidateOf turns what the agent sent into the candidate of the domain.
//
// The rules are the ones of the domain: the use case asks the domain to
// build the measure, the line and the ambiguity, and it never builds them
// by itself.
func (s *PurchaseService) candidateOf(
	request ports.DraftPurchase,
	catalog ingredient.Catalog,
) (purchase.Candidate, error) {
	lines := make([]purchase.Line, 0, len(request.Lines))
	for _, one := range request.Lines {
		measure, err := purchase.NewMeasure(one.Quantity, s.unitOfTheLine(one, catalog))
		if err != nil {
			return purchase.Candidate{}, err
		}

		line, err := purchase.NewLine(one.Name, measure, purchase.Price{
			Total:    ledger.Amount(one.Total),
			Unit:     ledger.Amount(one.UnitPrice),
			Inferred: one.Inferred,
		})
		if err != nil {
			return purchase.Candidate{}, err
		}

		// The identity of the ingredient is the one the agent chose with
		// the tool of the catalog: the line keeps it, and the name is kept
		// as the owner said it either way.
		lines = append(lines, line.WithIngredient(ingredient.ID(one.IngredientID)))
	}

	ambiguities := make([]purchase.Ambiguity, 0, len(request.Ambiguities))
	for _, one := range request.Ambiguities {
		ambiguity, err := purchase.NewAmbiguity(one.Name, one.Options)
		if err != nil {
			return purchase.Candidate{}, err
		}

		ambiguities = append(ambiguities, ambiguity)
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return purchase.Candidate{}, err
	}

	return purchase.Candidate{
		Lines:       lines,
		Day:         day,
		Market:      request.Market,
		Invoice:     request.Invoice,
		Ambiguities: ambiguities,
	}, nil
}

// unitOfTheLine answers the unit of a line of the purchase.
//
// The rule of the use case: the unit of the line is the unit the owner
// named, and the unit of the ingredient of the catalog when she named
// none. The unit is empty when neither of them carries one, and the
// domain refuses the line that ends without a unit.
func (s *PurchaseService) unitOfTheLine(line ports.PurchaseLine, catalog ingredient.Catalog) string {
	if named := strings.TrimSpace(line.Unit); named != "" {
		return named
	}

	resolved, err := catalog.Resolve(ingredient.ID(line.IngredientID), line.Name)
	if err != nil {
		return ""
	}

	return resolved.Ingredient.Unit.String()
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *PurchaseService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *PurchaseService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}

// ConfirmationService is the use case that writes the purchase of a
// confirmed draft.
//
// The agent reads the sentence of the owner, the draft waits for her yes,
// and the confirmation writes the fact: the purchase, its lines and the
// movement of the money of the payment.
//
// The confirmation writes the purchase of the confirmed draft, and it
// grows the catalog of the names the business buys: a name the catalog
// does not know becomes an ingredient of the business, recorded as the
// owner said it.
//
// The three are written in one transaction: a book that holds a purchase
// and no movement of its money, or a movement and no purchase, lies about
// what the owner bought.
type ConfirmationService struct {
	drafts      purchase.Repository
	purchases   purchase.Recorder
	accounts    account.Repository
	book        ledger.Repository
	ingredients ingredient.Repository
	work        store.UnitOfWork
	clock       ports.Clock
}

// NewConfirmationService builds the use case of the confirmation.
//
// The repositories are the ports of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: a draft that expired is refused, and the expiry is
// measured against the day of today.
func NewConfirmationService(
	drafts purchase.Repository,
	purchases purchase.Recorder,
	accounts account.Repository,
	book ledger.Repository,
	ingredients ingredient.Repository,
	work store.UnitOfWork,
	clock ports.Clock,
) *ConfirmationService {
	return &ConfirmationService{
		drafts:      drafts,
		purchases:   purchases,
		accounts:    accounts,
		book:        book,
		ingredients: ingredients,
		work:        work,
		clock:       clock,
	}
}

// Confirm writes the purchase of the draft the owner confirmed.
//
// The draft belongs to its business: the confirmation of a draft of another
// business is refused, and nothing is written.
//
// The money of the payment comes from the account the owner names, and
// from the cash of the business when she names none: the money lands in the
// inventory, because the business bought something and the inventory is
// what it has.
//
// The rules of the domain hold the confirmation: an expired draft is
// refused, a draft that changed under the eyes of the owner is refused, and
// a draft that still needs the money of one of its lines is refused. A
// refused confirmation leaves nothing behind: the draft stays as it was and
// no purchase is written.
func (s *ConfirmationService) Confirm(ctx context.Context, request ports.ConfirmPurchase) (purchase.Confirmation, error) {
	if s == nil || s.drafts == nil || s.purchases == nil || s.accounts == nil || s.book == nil ||
		s.ingredients == nil || s.work == nil || s.clock == nil {
		return purchase.Confirmation{}, ports.ErrNoDatabase
	}

	draft, err := s.drafts.Find(ctx, request.DraftID)
	if err != nil {
		return purchase.Confirmation{}, err
	}

	if draft.BusinessID != request.BusinessID {
		return purchase.Confirmation{}, fmt.Errorf("%w: the draft %d is of the business %d",
			purchase.ErrWrongBusiness, draft.ID, draft.BusinessID)
	}

	confirmation, err := purchase.Confirm(draft, purchase.Yes{
		Revision: request.Revision,
		Word:     request.ConfirmedWith,
	}, s.today())
	if err != nil {
		return purchase.Confirmation{}, err
	}

	accounts, err := s.accounts.List(ctx, request.BusinessID)
	if err != nil {
		return purchase.Confirmation{}, err
	}

	paying, err := purchase.ThePayingAccount(accounts, request.FromAccountID)
	if err != nil {
		return purchase.Confirmation{}, err
	}

	inventory, err := purchase.TheInventoryAccount(accounts)
	if err != nil {
		return purchase.Confirmation{}, err
	}

	movement, err := confirmation.Purchase.Movement(paying.ID, inventory.ID)
	if err != nil {
		return purchase.Confirmation{}, err
	}

	var written purchase.Purchase
	err = s.work.Do(ctx, func(ctx context.Context) error {
		lines, newNames, err := s.resolveLines(ctx, request.BusinessID, confirmation.Purchase.Lines)
		if err != nil {
			return err
		}

		toWrite := confirmation.Purchase
		toWrite.Lines = lines

		recorded, err := s.purchases.Record(ctx, toWrite)
		if err != nil {
			return err
		}

		if _, err := s.book.Record(ctx, movement); err != nil {
			return err
		}

		if _, err := s.drafts.Confirm(ctx, draft.ID, request.Revision); err != nil {
			return err
		}

		written = recorded
		confirmation.NewNames = newNames

		return nil
	})
	if err != nil {
		return purchase.Confirmation{}, err
	}

	confirmation.Purchase = written
	confirmation.Recorded = written.Lines

	return confirmation, nil
}

// resolveLines answers the ingredient of every line of the purchase, and the
// names the catalog did not know.
//
// The catalog of the business grows with what the owner buys: a name the
// catalog does not know becomes an ingredient of the business, recorded as
// she said it. The name of a line the catalog knows is the ingredient of
// that line, and a name the catalog cannot answer leaves the line with the
// name as she said it.
//
// The names the catalog did not know are the ones the answer carries: a
// name the catalog does not know is not an error, and the owner has to be
// told that it is new.
//
// The rules are the ones of the domain: the use case reads the catalog and
// asks the domain what it knows about every name.
func (s *ConfirmationService) resolveLines(
	ctx context.Context,
	own business.ID,
	lines []purchase.Line,
) ([]purchase.Line, []string, error) {
	items, err := s.ingredients.List(ctx, own)
	if err != nil {
		return nil, nil, err
	}

	catalog, err := ingredient.NewCatalog(own, items)
	if err != nil {
		return nil, nil, err
	}

	newNames := []string{}
	resolved := make([]purchase.Line, 0, len(lines))
	for _, one := range lines {
		resolution, err := catalog.Resolve(one.IngredientID, one.Name)
		if err != nil {
			return nil, nil, err
		}

		if resolution.New {
			created, err := ingredient.New(own, one.Name, nil)
			if err != nil {
				return nil, nil, err
			}

			written, err := s.ingredients.Create(ctx, created)
			if err != nil {
				return nil, nil, err
			}

			resolution.Ingredient = written
			newNames = append(newNames, written.Name)

			catalog, err = catalog.With(written)
			if err != nil {
				return nil, nil, err
			}
		}

		resolved = append(resolved, one.WithIngredient(resolution.Ingredient.ID))
	}

	return resolved, newNames, nil
}

// today answers the day of today.
func (s *ConfirmationService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
