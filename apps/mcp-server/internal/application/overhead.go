package application

import (
	"context"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// OverheadService is the use case that writes the overhead of the house.
//
// The agent reads the electricity the owner paid, the hours the oven ran
// and the allocation she declared, and the owner confirms it; the domain
// decides what an overhead is, and the use case writes it in the book of
// the business.
//
// The electricity and the allocation are never inferred: the use case
// writes what the owner said, and the domain refuses an overhead that is
// not whole.
//
// The overhead of the house is a fact of the business: it moves no money of
// the book yet, and the account of the overhead exists in the seed of the
// business.
type OverheadService struct {
	overheads recipe.OverheadRepository
	clock     ports.Clock
}

// NewOverheadService builds the use case of the overhead.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: the day an overhead names none is the day of today,
// and the service does not read the clock of the machine by itself.
func NewOverheadService(overheads recipe.OverheadRepository, clock ports.Clock) *OverheadService {
	return &OverheadService{overheads: overheads, clock: clock}
}

// Record writes the overhead the owner confirmed.
//
// The rules of the domain hold the recording: an overhead without the
// electricity of the house, without the hours of the oven, without an
// allocation that is a share, without the day it was declared and without
// the word of the owner are refused, and a refused overhead leaves nothing
// behind.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *OverheadService) Record(ctx context.Context, request ports.RecordOverhead) (recipe.Overhead, error) {
	if s == nil || s.overheads == nil || s.clock == nil {
		return recipe.Overhead{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return recipe.Overhead{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	overhead, err := recipe.NewOverhead(request.BusinessID, recipe.OverheadDefinition{
		Electricity:   ledger.Amount(request.Electricity),
		OvenHours:     request.OvenHours,
		Allocation:    request.Allocation,
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return recipe.Overhead{}, err
	}

	return s.overheads.Record(ctx, overhead)
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *OverheadService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *OverheadService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
