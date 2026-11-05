package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
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
type PurchaseService struct {
	drafts purchase.Repository
	clock  ports.Clock
}

// NewPurchaseService builds the use case of the drafts.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: the day of a draft that names none is the day of
// today, and the service does not read the clock of the machine by itself.
func NewPurchaseService(drafts purchase.Repository, clock ports.Clock) *PurchaseService {
	return &PurchaseService{drafts: drafts, clock: clock}
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
	if s == nil || s.drafts == nil || s.clock == nil {
		return purchase.Draft{}, ports.ErrNoDatabase
	}

	candidate, err := s.candidateOf(request)
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

	amended, err := purchase.Amend(existing, candidate)
	if err != nil {
		return purchase.Draft{}, err
	}

	return s.drafts.Save(ctx, amended)
}

// candidateOf turns what the agent sent into the candidate of the domain.
//
// The rules are the ones of the domain: the use case asks the domain to
// build the measure, the line and the ambiguity, and it never builds them
// by itself.
func (s *PurchaseService) candidateOf(request ports.DraftPurchase) (purchase.Candidate, error) {
	lines := make([]purchase.Line, 0, len(request.Lines))
	for _, one := range request.Lines {
		measure, err := purchase.NewMeasure(one.Quantity, one.Unit)
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

		lines = append(lines, line)
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
