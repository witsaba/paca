package application

import (
	"context"
	"strings"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/ledger"
)

// BookService is the use case of the book of a business.
//
// The book is the money of a person: the movements that fill it and the
// balance that comes out of it. The service speaks in the model of the
// domain and it writes through the port of the storage, and it knows
// nothing about the protocol that reaches it.
type BookService struct {
	book  ledger.Repository
	clock ports.Clock
}

// NewBookService builds the use case of the book.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else. The clock is the
// port of the time: a movement of today needs the day of today, and the
// service does not read the clock of the machine by itself.
func NewBookService(book ledger.Repository, clock ports.Clock) *BookService {
	return &BookService{book: book, clock: clock}
}

// Record writes a movement of the book.
//
// The money lands in the account of the destination and comes from the
// account of the source: the debit of the movement is the destination and
// the credit is the source.
//
// The day of the movement is the one the caller names, and the day of today
// when the caller names none.
func (s *BookService) Record(
	ctx context.Context,
	businessID business.ID,
	fromAccountID, toAccountID account.ID,
	amount ledger.Amount,
	date, description string,
) (ledger.Entry, error) {
	if s == nil || s.book == nil || s.clock == nil {
		return ledger.Entry{}, ports.ErrNoDatabase
	}

	day, err := s.dayOf(date)
	if err != nil {
		return ledger.Entry{}, err
	}

	entry, err := ledger.NewEntry(businessID, toAccountID, fromAccountID, amount, day, description)
	if err != nil {
		return ledger.Entry{}, err
	}

	return s.book.Record(ctx, entry)
}

// Movements returns the movements of the book between two days.
//
// The days are the two ends of the period, and the movement of the first
// day and the movement of the last one are both in the answer.
//
// The period is built by the domain, and a period that ends before it
// begins is refused: the book of a person answers the movements of a
// stretch of time, and a stretch that ends before it starts is not a
// stretch of time.
func (s *BookService) Movements(
	ctx context.Context,
	businessID business.ID,
	from, to string,
) ([]ledger.Entry, error) {
	if s == nil || s.book == nil || s.clock == nil {
		return nil, ports.ErrNoDatabase
	}

	first, err := s.dayOf(from)
	if err != nil {
		return nil, err
	}

	last, err := s.dayOf(to)
	if err != nil {
		return nil, err
	}

	period, err := ledger.NewPeriod(first, last)
	if err != nil {
		return nil, err
	}

	return s.book.List(ctx, businessID, period.From, period.To)
}

// Balances returns the balance of every account of the book, and the money
// of the business as a whole.
//
// The day is the day the balance is read as of, and the day of today when
// the caller names none.
func (s *BookService) Balances(ctx context.Context, businessID business.ID, asOf string) (ledger.Book, []ledger.Balance, error) {
	if s == nil || s.book == nil || s.clock == nil {
		return ledger.Book{}, nil, ports.ErrNoDatabase
	}

	day, err := s.dayOf(asOf)
	if err != nil {
		return ledger.Book{}, nil, err
	}

	balances, err := s.book.Balances(ctx, businessID, day)
	if err != nil {
		return ledger.Book{}, nil, err
	}

	return ledger.BookOf(businessID, balances), balances, nil
}

// dayOf returns the day of the book that the caller names, and the day of
// today when the caller names none.
//
// The book of a person is written the day it happens, and a caller that
// does not name the day means the day of today: the agent of Paca speaks
// to a person in the present tense.
func (s *BookService) dayOf(date string) (ledger.Date, error) {
	if strings.TrimSpace(date) == "" {
		return ledger.NewDate(s.clock.Now()), nil
	}

	return ledger.ParseDate(date)
}
