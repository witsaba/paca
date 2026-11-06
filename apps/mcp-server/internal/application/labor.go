package application

import (
	"context"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/recipe"
)

// LaborService is the use case that writes the labor of the owner.
//
// The agent reads the hours the owner worked and the value of the hour she
// declared, and the owner confirms it; the domain decides what a labor is,
// and the use case writes it in the book of the business.
//
// The hours and the value of the hour are never inferred: the use case
// writes what the owner said, and the domain refuses a labor that is not
// whole.
//
// The labor of the PAC is the labor of the owner alone: the payroll of the
// employees is out of scope for now, and the model of the labor of an
// employee does not exist yet.
type LaborService struct {
	labors recipe.LaborRepository
	clock  ports.Clock
}

// NewLaborService builds the use case of the labor.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: the day a labor names none is the day of today, and
// the service does not read the clock of the machine by itself.
func NewLaborService(labors recipe.LaborRepository, clock ports.Clock) *LaborService {
	return &LaborService{labors: labors, clock: clock}
}

// Record writes the labor the owner confirmed.
//
// The rules of the domain hold the recording: a labor without hours, a
// labor without the value of the hour, a labor without the day it was
// worked and a labor without the word of the owner are refused, and a
// refused labor leaves nothing behind.
//
// The day is the one the agent names, and the day of today when it names
// none: the day is one of the three things Paca may fill by herself.
func (s *LaborService) Record(ctx context.Context, request ports.RecordLabor) (recipe.Labor, error) {
	if s == nil || s.labors == nil || s.clock == nil {
		return recipe.Labor{}, ports.ErrNoDatabase
	}

	hours, err := recipe.NewHours(request.HandsOnHours, request.OvenHours, request.WaitingHours)
	if err != nil {
		return recipe.Labor{}, err
	}

	day, err := s.dayOf(request.Day)
	if err != nil {
		return recipe.Labor{}, err
	}

	if day.Time().IsZero() {
		day = s.today()
	}

	labour, err := recipe.NewLabor(request.BusinessID, recipe.LaborDefinition{
		Hours:         hours,
		HourValue:     ledger.Amount(request.HourValue),
		Day:           day,
		Utterance:     request.Utterance,
		ConfirmedWith: request.ConfirmedWith,
	})
	if err != nil {
		return recipe.Labor{}, err
	}

	return s.labors.Record(ctx, labour)
}

// dayOf answers the day the agent named.
//
// The day is written the way a person writes it, and it is the day of today
// when the agent names none.
func (s *LaborService) dayOf(day string) (ledger.Date, error) {
	if strings.TrimSpace(day) == "" {
		return ledger.Date{}, nil
	}

	return ledger.ParseDate(day)
}

// today answers the day of today.
func (s *LaborService) today() ledger.Date {
	return ledger.NewDate(s.clock.Now())
}
