package clock_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/packages/platform/clock"
)

// TestTheSystemClockTellsTheTime proves that the clock of the host answers
// with the time of the host: it is after the moment the test took, and it
// does not run backwards.
func TestTheSystemClockTellsTheTime(t *testing.T) {
	t.Parallel()

	before := time.Now()
	got := clock.System{}.Now()
	after := time.Now()

	if got.Before(before) {
		t.Fatalf("Now() = %s, want a time at or after %s", got, before)
	}
	if got.After(after) {
		t.Fatalf("Now() = %s, want a time at or before %s", got, after)
	}
}

// TestTheSystemClockMovesForward proves that the clock of the host does not
// stand still and does not run backwards.
func TestTheSystemClockMovesForward(t *testing.T) {
	t.Parallel()

	first := clock.System{}.Now()
	second := clock.System{}.Now()

	if second.Before(first) {
		t.Fatalf("Now() = %s, want a time at or after %s", second, first)
	}
}
