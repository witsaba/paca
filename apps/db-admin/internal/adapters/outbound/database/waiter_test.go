package database_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/adapters/outbound/database"
)

// errNoAnswer is the refusal of the database of the tests: it does not
// answer yet.
//
// It is what the real database answers when its container has started and
// its server does not accept the connections yet: the connection is
// refused, and the refusal is not an error of the context.
var errNoAnswer = errors.New("the database did not answer")

// refusingDatabase is a database that refuses to answer a number of times
// before it answers.
//
// The writer of the schema and the database of the monorepository start at
// the same moment, and the database accepts the connections a couple of
// seconds after its container starts: the writer asks once, it is refused,
// and it has to ask again.
type refusingDatabase struct {
	// refusals is the number of refusals left before the database
	// answers.
	refusals int
	// asked counts the questions the database received.
	asked int
}

// Ping answers the question of the writer. It is the pinger the waiter
// asks.
func (d *refusingDatabase) Ping(_ context.Context) error {
	d.asked++

	if d.refusals > 0 {
		d.refusals--
		return errNoAnswer
	}

	return nil
}

// recordingSleeper records the pauses and never waits.
type recordingSleeper struct {
	// waits counts the pauses.
	waits int
	// interrupted stops the wait, the way a broken pause does.
	interrupted bool
}

// sleep records the pause of the writer.
func (s *recordingSleeper) sleep(_ context.Context, _ time.Duration) error {
	s.waits++
	if s.interrupted {
		return errors.New("the pause was cut short")
	}
	return nil
}

// theLogger is the logger of the tests: it says nothing.
func theLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestTheWriterWaitsForTheDatabaseUntilItAnswers(t *testing.T) {
	t.Parallel()

	refusing := &refusingDatabase{refusals: 2}
	sleeper := &recordingSleeper{}

	waiter := database.NewWaiter(refusing, theLogger(), database.WaitInterval, sleeper.sleep)

	if err := waiter.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() = %v, want the database to answer", err)
	}

	if refusing.asked != 3 {
		t.Fatalf("the database was asked %d times, want 3", refusing.asked)
	}
	if sleeper.waits != 2 {
		t.Fatalf("the writer waited %d times, want 2", sleeper.waits)
	}
}

func TestTheWriterGivesUpWhenTheBudgetOfTheWaitIsSpent(t *testing.T) {
	t.Parallel()

	// The budget of the wait is the deadline of the context, and it is
	// spent before the writer asks.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	refusing := &refusingDatabase{refusals: 10}
	sleeper := &recordingSleeper{}

	waiter := database.NewWaiter(refusing, theLogger(), database.WaitInterval, sleeper.sleep)

	err := waiter.Wait(ctx)
	if err == nil {
		t.Fatal("Wait() = nil, want the wait to end when the budget is spent")
	}
	if !errors.Is(err, errNoAnswer) {
		t.Fatalf("Wait() = %v, want the refusal of the database to be the cause", err)
	}
	if refusing.asked != 1 {
		t.Fatalf("the database was asked %d times, want 1", refusing.asked)
	}
}

func TestTheWriterStopsWaitingWhenTheServiceIsCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	refusing := &refusingDatabase{refusals: 10}
	sleeper := &recordingSleeper{}

	waiter := database.NewWaiter(refusing, theLogger(), database.WaitInterval, sleeper.sleep)

	err := waiter.Wait(ctx)
	if err == nil {
		t.Fatal("Wait() = nil, want the wait to end when the service is cancelled")
	}
	if !errors.Is(err, errNoAnswer) {
		t.Fatalf("Wait() = %v, want the refusal of the database to be the cause", err)
	}
	if refusing.asked != 1 {
		t.Fatalf("the database was asked %d times, want 1", refusing.asked)
	}
}

func TestTheWriterStopsWaitingWhenThePauseIsInterrupted(t *testing.T) {
	t.Parallel()

	refusing := &refusingDatabase{refusals: 10}
	sleeper := &recordingSleeper{interrupted: true}

	waiter := database.NewWaiter(refusing, theLogger(), database.WaitInterval, sleeper.sleep)

	err := waiter.Wait(context.Background())
	if err == nil {
		t.Fatal("Wait() = nil, want the wait to end when the pause is interrupted")
	}
	if !errors.Is(err, errNoAnswer) {
		t.Fatalf("Wait() = %v, want the refusal of the database to be the cause", err)
	}
	if refusing.asked != 1 {
		t.Fatalf("the database was asked %d times, want 1", refusing.asked)
	}
}
