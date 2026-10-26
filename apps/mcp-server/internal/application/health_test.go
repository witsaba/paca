package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// fakeClock is the clock of the test: it only moves when the test moves it.
type fakeClock struct {
	now time.Time
}

// Now returns the time of the test.
func (c *fakeClock) Now() time.Time {
	return c.now
}

// fakeChecker is a checker whose outcome the test decides.
type fakeChecker struct {
	name  string
	err   error
	calls int
}

// Name identifies the component the checker watches.
func (c *fakeChecker) Name() string {
	return c.name
}

// Check returns the outcome the test decided.
func (c *fakeChecker) Check(context.Context) error {
	c.calls++
	return c.err
}

func TestHealthOfAServiceWithoutChecksIsHealthy(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.October, 27, 10, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: startedAt.Add(90 * time.Second)}

	service, err := application.NewHealthService(
		health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
		nil,
		clock,
		startedAt,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewHealthService() = %v", err)
	}

	report, err := service.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() = %v", err)
	}

	if report.Status != health.StatusOK {
		t.Fatalf("status = %q, want %q", report.Status, health.StatusOK)
	}
	if report.Uptime != 90*time.Second {
		t.Fatalf("uptime = %s, want %s", report.Uptime, 90*time.Second)
	}
	if report.Identity.Name != "paca-mcp-server" || report.Identity.Version != "0.1.0" {
		t.Fatalf("identity = %+v, want the name and the version of the service", report.Identity)
	}
	if len(report.Checks) != 0 {
		t.Fatalf("checks = %d, want none", len(report.Checks))
	}
}

func TestHealthAggregatesTheWorstStatusOfTheChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		want   health.Status
		detail bool
	}{
		{name: "a healthy component", err: nil, want: health.StatusOK},
		{name: "a degraded component", err: health.ErrDegraded, want: health.StatusDegraded, detail: true},
		{name: "a failing component", err: errors.New("the component is down"), want: health.StatusUnhealthy, detail: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			checker := &fakeChecker{name: "mcp-transport", err: test.err}
			clock := &fakeClock{now: time.Now()}

			service, err := application.NewHealthService(
				health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
				[]ports.Checker{checker},
				clock,
				clock.now,
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewHealthService() = %v", err)
			}

			report, err := service.Health(context.Background())
			if err != nil {
				t.Fatalf("Health() = %v", err)
			}

			if report.Status != test.want {
				t.Fatalf("status = %q, want %q", report.Status, test.want)
			}
			if len(report.Checks) != 1 {
				t.Fatalf("checks = %d, want 1", len(report.Checks))
			}
			if report.Checks[0].Name != "mcp-transport" {
				t.Fatalf("check name = %q, want %q", report.Checks[0].Name, "mcp-transport")
			}
			if test.detail && report.Checks[0].Detail == "" {
				t.Fatal("a check that is not fine must explain why")
			}
			if !test.detail && report.Checks[0].Detail != "" {
				t.Fatalf("detail = %q, want none", report.Checks[0].Detail)
			}
			if checker.calls != 1 {
				t.Fatalf("the checker ran %d times, want 1", checker.calls)
			}
		})
	}
}

func TestHealthWithACancelledContextReturnsTheError(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Now()}
	service, err := application.NewHealthService(
		health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
		nil,
		clock,
		clock.now,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewHealthService() = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.Health(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Health() = %v, want %v", err, context.Canceled)
	}
}

func TestNewHealthServiceRejectsAnInvalidConstruction(t *testing.T) {
	t.Parallel()

	clock := &fakeClock{now: time.Now()}

	tests := []struct {
		name     string
		identity health.Identity
		clock    ports.Clock
		timeout  time.Duration
	}{
		{name: "a service without a name", identity: health.Identity{}, clock: clock, timeout: time.Second},
		{name: "a service without a clock", identity: health.Identity{Name: "paca-mcp-server"}, clock: nil, timeout: time.Second},
		{name: "a service without a timeout", identity: health.Identity{Name: "paca-mcp-server"}, clock: clock, timeout: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := application.NewHealthService(test.identity, nil, test.clock, clock.now, test.timeout); err == nil {
				t.Fatal("NewHealthService() = nil, want an error")
			}
		})
	}
}
