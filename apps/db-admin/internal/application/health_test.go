package application_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/application"
	"github.com/witsaba/paca/apps/db-admin/internal/domain/health"
	"github.com/witsaba/paca/apps/db-admin/internal/ports"
)

// fakeClock is the clock of the tests: it answers the time it holds, and it
// moves forward on every question, so a check has a latency.
type fakeClock struct {
	mutex sync.Mutex
	now   time.Time
	step  time.Duration
}

// Now returns the time of the clock and moves it forward.
func (c *fakeClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := c.now
	c.now = c.now.Add(c.step)
	return now
}

// fakeChecker is a checker the test decides the outcome of.
type fakeChecker struct {
	name string
	err  error
}

// Name identifies the component the checker watches.
func (c *fakeChecker) Name() string {
	return c.name
}

// Check returns the error of the checker.
func (c *fakeChecker) Check(context.Context) error {
	return c.err
}

// newHealthService builds the health use case of the tests.
func newHealthService(t *testing.T, checkers ...*fakeChecker) *application.HealthService {
	t.Helper()

	startedAt := time.Date(2026, time.October, 27, 10, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: startedAt, step: 5 * time.Millisecond}

	ported := make([]ports.Checker, 0, len(checkers))
	for _, checker := range checkers {
		ported = append(ported, checker)
	}

	service, err := application.NewHealthService(
		health.Identity{Name: "paca-db-admin", Version: "1.2.3"},
		ported,
		clock,
		startedAt,
		2*time.Second,
	)
	if err != nil {
		t.Fatalf("NewHealthService() = %v", err)
	}

	return service
}

func TestTheHealthServiceNeedsTheNameOfTheService(t *testing.T) {
	t.Parallel()

	_, err := application.NewHealthService(health.Identity{}, nil, &fakeClock{}, time.Now(), time.Second)
	if err == nil {
		t.Fatal("NewHealthService() = nil, want an error about the name")
	}
}

func TestTheHealthServiceNeedsAClock(t *testing.T) {
	t.Parallel()

	_, err := application.NewHealthService(
		health.Identity{Name: "paca-db-admin"},
		nil,
		nil,
		time.Now(),
		time.Second,
	)
	if err == nil {
		t.Fatal("NewHealthService() = nil, want an error about the clock")
	}
}

func TestTheHealthServiceNeedsAPositiveTimeout(t *testing.T) {
	t.Parallel()

	_, err := application.NewHealthService(
		health.Identity{Name: "paca-db-admin"},
		nil,
		&fakeClock{},
		time.Now(),
		0,
	)
	if err == nil {
		t.Fatal("NewHealthService() = nil, want an error about the timeout")
	}
}

func TestTheHealthOfAServiceWithoutChecksIsHealthy(t *testing.T) {
	t.Parallel()

	report, err := newHealthService(t).Health(context.Background())
	if err != nil {
		t.Fatalf("Health() = %v", err)
	}

	if report.Status != health.StatusOK {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusOK)
	}
	if len(report.Checks) != 0 {
		t.Fatalf("checks = %+v, want none", report.Checks)
	}
}

func TestTheHealthReportCarriesTheIdentityAndTheUptime(t *testing.T) {
	t.Parallel()

	report, err := newHealthService(t).Health(context.Background())
	if err != nil {
		t.Fatalf("Health() = %v", err)
	}

	if report.Identity.Name != "paca-db-admin" || report.Identity.Version != "1.2.3" {
		t.Fatalf("identity = %+v, want the name and the version of the service", report.Identity)
	}
	if report.Uptime != 0 {
		t.Fatalf("uptime = %s, want none", report.Uptime)
	}
	if report.CheckedAt.IsZero() {
		t.Fatal("the report carries no moment, want the one of the check")
	}
}

func TestOneFailingCheckMakesTheWholeServiceUnhealthy(t *testing.T) {
	t.Parallel()

	service := newHealthService(t,
		&fakeChecker{name: "database", err: nil},
		&fakeChecker{name: "broken", err: errors.New("the database did not answer")},
	)

	report, err := service.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() = %v", err)
	}

	if report.Status != health.StatusUnhealthy {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusUnhealthy)
	}
	if len(report.Checks) != 2 {
		t.Fatalf("checks = %+v, want two of them", report.Checks)
	}
	if report.Checks[1].Status != health.StatusUnhealthy {
		t.Fatalf("the failing check is %q, want %q", report.Checks[1].Status, health.StatusUnhealthy)
	}
	if report.Checks[1].Detail != "the database did not answer" {
		t.Fatalf("detail = %q, want the failure of the check", report.Checks[1].Detail)
	}
}

func TestADegradedCheckMakesTheWholeServiceDegraded(t *testing.T) {
	t.Parallel()

	service := newHealthService(t,
		&fakeChecker{name: "database", err: health.ErrDegraded},
	)

	report, err := service.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() = %v", err)
	}

	if report.Status != health.StatusDegraded {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusDegraded)
	}
	if report.Checks[0].Status != health.StatusDegraded {
		t.Fatalf("the check is %q, want %q", report.Checks[0].Status, health.StatusDegraded)
	}
}

func TestTheHealthOfACancelledRequestFails(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := newHealthService(t).Health(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Health() = %v, want %v", err, context.Canceled)
	}
}

func TestTheCheckIsBoundByTheTimeoutOfTheService(t *testing.T) {
	t.Parallel()

	startedAt := time.Now()
	service, err := application.NewHealthService(
		health.Identity{Name: "paca-db-admin"},
		[]ports.Checker{&slowChecker{}},
		&fakeClock{now: startedAt},
		startedAt,
		50*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("NewHealthService() = %v", err)
	}

	report, err := service.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() = %v", err)
	}

	if report.Checks[0].Status != health.StatusUnhealthy {
		t.Fatalf("the check is %q, want %q", report.Checks[0].Status, health.StatusUnhealthy)
	}
	if !strings.Contains(report.Checks[0].Detail, context.DeadlineExceeded.Error()) {
		t.Fatalf("detail = %q, want the failure of the check", report.Checks[0].Detail)
	}
}

// slowChecker waits for the context of the check, so the timeout of the
// service is what ends it.
type slowChecker struct{}

// Name identifies the component the checker watches.
func (c *slowChecker) Name() string {
	return "slow"
}

// Check waits for the context to be done.
func (c *slowChecker) Check(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
