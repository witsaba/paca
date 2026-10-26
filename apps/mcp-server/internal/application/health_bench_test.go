package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
)

// maxHealthAllocations is the budget of the health use case. It is a budget
// and not a benchmark result: the number of allocations of a report does not
// depend on the machine, so the check is the same everywhere.
const maxHealthAllocations = 40

// benchmarkService builds the service the benchmark and the budget check use.
func benchmarkService(tb testing.TB) *application.HealthService {
	tb.Helper()

	clock := &fakeClock{now: time.Now()}
	service, err := application.NewHealthService(
		health.Identity{Name: "paca-mcp-server", Version: "0.1.0"},
		[]ports.Checker{&fakeChecker{name: "mcp-transport"}},
		clock,
		clock.now,
		time.Second,
	)
	if err != nil {
		tb.Fatalf("NewHealthService() = %v", err)
	}
	return service
}

// BenchmarkHealthReport measures the cost of a health report.
func BenchmarkHealthReport(b *testing.B) {
	service := benchmarkService(b)
	ctx := context.Background()

	b.ReportAllocs()
	for b.Loop() {
		if _, err := service.Health(ctx); err != nil {
			b.Fatalf("Health() = %v", err)
		}
	}
}

// TestTheHealthUseCaseStaysWithinItsAllocationBudget fails when the health
// report starts to allocate more than it should.
func TestTheHealthUseCaseStaysWithinItsAllocationBudget(t *testing.T) {
	service := benchmarkService(t)
	ctx := context.Background()

	allocations := testing.AllocsPerRun(100, func() {
		if _, err := service.Health(ctx); err != nil {
			t.Fatalf("Health() = %v", err)
		}
	})

	if allocations > maxHealthAllocations {
		t.Fatalf("the health report allocates %v times, want at most %d", allocations, maxHealthAllocations)
	}
}
