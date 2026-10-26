package serving_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/serving"
)

func TestTheCheckerReportsThatTheTransportIsNotServingYet(t *testing.T) {
	t.Parallel()

	checker := serving.NewChecker()

	if err := checker.Check(context.Background()); !errors.Is(err, serving.ErrNotServing) {
		t.Fatalf("Check() = %v, want %v", err, serving.ErrNotServing)
	}
}

func TestTheCheckerReportsThatTheTransportIsServing(t *testing.T) {
	t.Parallel()

	checker := serving.NewChecker()
	checker.SetServing(true)

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("Check() = %v, want nil", err)
	}
}

func TestTheCheckerStopsReportingWhenTheTransportStops(t *testing.T) {
	t.Parallel()

	checker := serving.NewChecker()
	checker.SetServing(true)
	checker.SetServing(false)

	if err := checker.Check(context.Background()); !errors.Is(err, serving.ErrNotServing) {
		t.Fatalf("Check() = %v, want %v", err, serving.ErrNotServing)
	}
}

func TestTheCheckerNamesTheComponentItWatches(t *testing.T) {
	t.Parallel()

	if got := serving.NewChecker().Name(); got != "mcp-transport" {
		t.Fatalf("Name() = %q, want %q", got, "mcp-transport")
	}
}
