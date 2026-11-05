package ledger

import (
	"context"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/business"
)

// fakeRepository is a book that is written in memory: it proves that the
// port of the book can be implemented, and it holds the calls it received.
type fakeRepository struct {
	summary Summary
	from    Date
	to      Date
}

func (f *fakeRepository) Record(_ context.Context, entry Entry) (Entry, error) {
	return entry, nil
}

func (f *fakeRepository) List(_ context.Context, _ business.ID, _, _ Date) ([]Entry, error) {
	return nil, nil
}

func (f *fakeRepository) Balances(_ context.Context, _ business.ID, _ Date) ([]Balance, error) {
	return nil, nil
}

func (f *fakeRepository) Summary(_ context.Context, _ business.ID, from, to Date) (Summary, error) {
	f.from = from
	f.to = to

	return f.summary, nil
}

// The port of the book is what a use case needs: an implementation of it
// must answer with the money of a period.
var _ Repository = (*fakeRepository)(nil)

func TestThePortOfTheBookAsksForTheMoneyOfAPeriod(t *testing.T) {
	t.Parallel()

	from := NewDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC))
	to := NewDate(time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC))

	book := &fakeRepository{summary: NewSummary(2_000_000, 1_500_000)}

	var repository Repository = book

	summary, err := repository.Summary(context.Background(), business.ID(77), from, to)
	if err != nil {
		t.Fatalf("the money of the period was refused: %v", err)
	}
	if summary.Net() != 500_000 {
		t.Errorf("the net is %d, want 500000", summary.Net())
	}
	if book.from != from || book.to != to {
		t.Errorf("the period asked was %v..%v, want %v..%v", book.from, book.to, from, to)
	}
}
