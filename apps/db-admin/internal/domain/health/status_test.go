package health_test

import (
	"testing"

	"github.com/witsaba/paca/apps/db-admin/internal/domain/health"
)

func TestTheWorstStatusIsTheSickestOne(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		left  health.Status
		right health.Status
		want  health.Status
	}{
		{"ok and ok", health.StatusOK, health.StatusOK, health.StatusOK},
		{"ok and degraded", health.StatusOK, health.StatusDegraded, health.StatusDegraded},
		{"ok and unhealthy", health.StatusOK, health.StatusUnhealthy, health.StatusUnhealthy},
		{"degraded and ok", health.StatusDegraded, health.StatusOK, health.StatusDegraded},
		{"degraded and unhealthy", health.StatusDegraded, health.StatusUnhealthy, health.StatusUnhealthy},
		{"unhealthy and degraded", health.StatusUnhealthy, health.StatusDegraded, health.StatusUnhealthy},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.left.Worst(testCase.right); got != testCase.want {
				t.Fatalf("Worst() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAnUnknownStatusIsNeverTrusted(t *testing.T) {
	t.Parallel()

	if got := health.StatusOK.Worst("something else"); got != health.StatusUnhealthy {
		t.Fatalf("Worst() = %q, want %q", got, health.StatusUnhealthy)
	}
	if got := health.Status("something else").Worst(health.StatusOK); got != health.StatusUnhealthy {
		t.Fatalf("Worst() = %q, want %q", got, health.StatusUnhealthy)
	}
}

func TestTheStatusSaysWhetherTheServiceServes(t *testing.T) {
	t.Parallel()

	if !health.StatusOK.IsServing() {
		t.Fatal("an ok service does not serve, want it to serve")
	}
	if !health.StatusDegraded.IsServing() {
		t.Fatal("a degraded service does not serve, want it to serve")
	}
	if health.StatusUnhealthy.IsServing() {
		t.Fatal("an unhealthy service serves, want it not to serve")
	}
}

func TestOnlyAnOKServiceIsHealthy(t *testing.T) {
	t.Parallel()

	if !health.StatusOK.IsHealthy() {
		t.Fatal("an ok service is not healthy, want it to be healthy")
	}
	if health.StatusDegraded.IsHealthy() {
		t.Fatal("a degraded service is healthy, want it not to be healthy")
	}
	if health.StatusUnhealthy.IsHealthy() {
		t.Fatal("an unhealthy service is healthy, want it not to be healthy")
	}
}

func TestTheStatusIsWrittenAsTheTextOfTheAPI(t *testing.T) {
	t.Parallel()

	if got := health.StatusDegraded.String(); got != "degraded" {
		t.Fatalf("String() = %q, want %q", got, "degraded")
	}
}
