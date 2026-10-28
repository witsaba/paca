package clock_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/platform/clock"
)

func TestTheSystemClockTellsTheTimeOfTheHost(t *testing.T) {
	t.Parallel()

	before := time.Now()
	now := clock.System{}.Now()
	after := time.Now()

	if now.Before(before) || now.After(after) {
		t.Fatalf("now = %v, want a time between %v and %v", now, before, after)
	}
}
