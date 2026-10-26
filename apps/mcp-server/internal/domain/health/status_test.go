package health_test

import (
	"testing"

	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
)

func TestStatusWorstKeepsTheSickestStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		first health.Status
		other health.Status
		want  health.Status
	}{
		{name: "ok and ok", first: health.StatusOK, other: health.StatusOK, want: health.StatusOK},
		{name: "ok and degraded", first: health.StatusOK, other: health.StatusDegraded, want: health.StatusDegraded},
		{name: "ok and unhealthy", first: health.StatusOK, other: health.StatusUnhealthy, want: health.StatusUnhealthy},
		{name: "degraded and ok", first: health.StatusDegraded, other: health.StatusOK, want: health.StatusDegraded},
		{name: "degraded and unhealthy", first: health.StatusDegraded, other: health.StatusUnhealthy, want: health.StatusUnhealthy},
		{name: "unhealthy and degraded", first: health.StatusUnhealthy, other: health.StatusDegraded, want: health.StatusUnhealthy},
		{name: "an unknown status is never trusted", first: health.StatusOK, other: health.Status("what"), want: health.StatusUnhealthy},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.first.Worst(test.other); got != test.want {
				t.Fatalf("Worst() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestStatusIsHealthyOnlyWhenEverythingIsFine(t *testing.T) {
	t.Parallel()

	if !health.StatusOK.IsHealthy() {
		t.Fatal("a healthy service must report itself as healthy")
	}
	if health.StatusDegraded.IsHealthy() {
		t.Fatal("a degraded service is not completely healthy")
	}
	if health.StatusUnhealthy.IsHealthy() {
		t.Fatal("an unhealthy service is not healthy")
	}
}

func TestStatusIsServingWhileItIsNotUnhealthy(t *testing.T) {
	t.Parallel()

	if !health.StatusOK.IsServing() {
		t.Fatal("a healthy service serves")
	}
	if !health.StatusDegraded.IsServing() {
		t.Fatal("a degraded service still serves")
	}
	if health.StatusUnhealthy.IsServing() {
		t.Fatal("an unhealthy service does not serve")
	}
}

func TestStatusStringIsTheTextTheAPIExposes(t *testing.T) {
	t.Parallel()

	if got := health.StatusUnhealthy.String(); got != "unhealthy" {
		t.Fatalf("String() = %q, want %q", got, "unhealthy")
	}
}
