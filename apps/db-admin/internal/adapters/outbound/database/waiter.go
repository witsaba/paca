package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
)

// The defaults of the wait for the database.
const (
	// WaitInterval is how long the writer pauses between two attempts to
	// reach the database.
	//
	// The pause is short: the database of the monorepository accepts the
	// connections a couple of seconds after its container starts, and the
	// writer of the schema waits for it.
	WaitInterval = time.Second

	// attemptTimeout bounds a single attempt to reach the database.
	//
	// One attempt that hangs must not eat the whole budget of the wait:
	// the writer asks again, and it asks again, until the budget is
	// spent.
	attemptTimeout = 10 * time.Second
)

// Sleeper pauses between two attempts.
//
// It returns nil when the pause is over, and the reason why it was cut
// short when it was.
type Sleeper func(context.Context, time.Duration) error

// Waiter waits for the database to answer.
//
// The writer of the schema and the database of the monorepository start at
// the same moment, and the database accepts the connections a couple of
// seconds after its container starts. The writer waits for the database
// instead of asking it once and exiting: a container that exits is
// restarted by the runtime of Docker, and the health check of the container
// declares it unhealthy before the restart succeeds. The stack then comes
// up by a race, and the race is lost more often than not.
//
// The wait is bounded by the context it is given: the budget of the wait is
// the deadline of that context.
type Waiter struct {
	pinger   ports.Pinger
	logger   *slog.Logger
	interval time.Duration
	sleep    Sleeper
}

// NewWaiter builds the waiter of the database.
//
// The pinger asks the database whether it answers, the logger says what the
// wait is doing, the interval is the pause between two attempts, and the
// sleeper is the pause itself.
func NewWaiter(pinger ports.Pinger, logger *slog.Logger, interval time.Duration, sleep Sleeper) *Waiter {
	return &Waiter{
		pinger:   pinger,
		logger:   logger,
		interval: interval,
		sleep:    sleep,
	}
}

// Wait asks the database until it answers, or until the budget of the wait
// is spent.
//
// The refusal of the database is the cause of the error it returns: the
// operator sees why the database did not answer, and not only that it did
// not.
func (w *Waiter) Wait(ctx context.Context) error {
	var last error

	for {
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		err := w.pinger.Ping(attemptCtx)
		cancel()

		if err == nil {
			return nil
		}

		// The refusal is kept while the budget is alive: when the budget
		// runs out, the last refusal of the database is the reason the
		// operator needs, and not the deadline of the wait.
		if ctx.Err() == nil || last == nil {
			last = err
		}

		// The budget of the wait is spent: there is nothing left to wait
		// for.
		if ctx.Err() != nil {
			return waitEnded(ctx.Err(), last)
		}

		w.logger.Info("the database did not answer, the writer waits for it",
			slog.Any("error", err),
			slog.Duration("pause", w.interval),
		)

		if err := w.sleep(ctx, w.interval); err != nil {
			if ctx.Err() != nil {
				return waitEnded(ctx.Err(), last)
			}
			return fmt.Errorf("the wait for the database was interrupted: %w", errors.Join(last, err))
		}
	}
}

// waitEnded says why the wait ended.
//
// A budget that ran out and a wait that was cancelled are two different
// stories: the first one is the database that never answered, and the
// second one is somebody who stopped the service.
func waitEnded(cause, last error) error {
	if errors.Is(cause, context.Canceled) {
		return fmt.Errorf("the wait for the database was interrupted: %w", last)
	}
	return fmt.Errorf("the budget of the wait was spent: %w", last)
}

// Sleep pauses for the duration, or until the context is done.
//
// The pause of the writer is bounded by the context of the wait: a wait
// that was cut short must not keep the writer sleeping.
func Sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
