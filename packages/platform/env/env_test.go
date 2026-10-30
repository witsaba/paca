package env_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/packages/platform/env"
)

func TestTheVersionOfTheServiceHasOneName(t *testing.T) {
	t.Parallel()

	if env.ServiceVersion != "PACA_SERVICE_VERSION" {
		t.Fatalf("service version = %q, want %q", env.ServiceVersion, "PACA_SERVICE_VERSION")
	}
}

func TestDurationReadsADuration(t *testing.T) {
	t.Parallel()

	duration, err := env.Duration("PACA_TEST_TIMEOUT", "30s")
	if err != nil {
		t.Fatalf("Duration() = %v", err)
	}
	if duration != 30*time.Second {
		t.Fatalf("duration = %s, want %s", duration, 30*time.Second)
	}
}

func TestDurationTrimsTheSpacesOfTheValue(t *testing.T) {
	t.Parallel()

	duration, err := env.Duration("PACA_TEST_TIMEOUT", " 5s ")
	if err != nil {
		t.Fatalf("Duration() = %v", err)
	}
	if duration != 5*time.Second {
		t.Fatalf("duration = %s, want %s", duration, 5*time.Second)
	}
}

func TestDurationRejectsAValueThatIsNotADuration(t *testing.T) {
	t.Parallel()

	duration, err := env.Duration("PACA_TEST_TIMEOUT", "soon")
	if err == nil {
		t.Fatalf("Duration() = %s, want an error", duration)
	}

	want := `PACA_TEST_TIMEOUT: "soon" is not a duration`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
