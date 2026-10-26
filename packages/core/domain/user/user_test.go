package user_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/domain/user"
)

func TestAllStatuses(t *testing.T) {
	t.Parallel()

	want := []user.Status{
		user.StatusOnboarding,
		user.StatusActive,
		user.StatusInactive,
		user.StatusBlocked,
	}

	got := user.AllStatuses()

	if len(got) != len(want) {
		t.Fatalf("the list holds %d statuses, want %d: %v", len(got), len(want), got)
	}
	for i, status := range want {
		if got[i] != status {
			t.Errorf("the status %d is %q, want %q", i, got[i], status)
		}
	}
}

func TestAllStatusesReturnsACopy(t *testing.T) {
	t.Parallel()

	first := user.AllStatuses()
	first[0] = "tampered"

	second := user.AllStatuses()

	if second[0] != user.StatusOnboarding {
		t.Errorf("the list of the statuses was tampered through the copy: %q", second[0])
	}
}

func TestStatusValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status user.Status
		want   bool
	}{
		{name: "onboarding is a state", status: user.StatusOnboarding, want: true},
		{name: "active is a state", status: user.StatusActive, want: true},
		{name: "inactive is a state", status: user.StatusInactive, want: true},
		{name: "blocked is a state", status: user.StatusBlocked, want: true},
		{name: "the empty status is not a state", status: user.Status(""), want: false},
		{name: "an unknown status is not a state", status: user.Status("unknown"), want: false},
		{name: "the state is case sensitive", status: user.Status("Active"), want: false},
		{name: "a trailing space is not a state", status: user.Status("active "), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.status.Valid(); got != tc.want {
				t.Errorf("the status %q is valid: %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestStatusString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status user.Status
		want   string
	}{
		{status: user.StatusOnboarding, want: "onboarding"},
		{status: user.StatusActive, want: "active"},
		{status: user.StatusInactive, want: "inactive"},
		{status: user.StatusBlocked, want: "blocked"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			if got := tc.status.String(); got != tc.want {
				t.Errorf("the status is %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want user.Status
	}{
		{name: "onboarding", want: user.StatusOnboarding},
		{name: "active", want: user.StatusActive},
		{name: "inactive", want: user.StatusInactive},
		{name: "blocked", want: user.StatusBlocked},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := user.ParseStatus(tc.name)
			if err != nil {
				t.Fatalf("the status %q did not parse: %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("the status is %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseStatusRejectsAnUnknownState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state string
	}{
		{name: "the empty status", state: ""},
		{name: "an unknown status", state: "unknown"},
		{name: "a status with a trailing space", state: "active "},
		{name: "a status in another case", state: "ACTIVE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := user.ParseStatus(tc.state)
			if err == nil {
				t.Fatalf("the status %q parsed into %q, want an error", tc.state, got)
			}
			if !errors.Is(err, user.ErrUnknownStatus) {
				t.Errorf("the error is %v, want it to wrap ErrUnknownStatus", err)
			}
		})
	}
}

func TestParseStatusNamesTheStateInTheError(t *testing.T) {
	t.Parallel()

	const state = "unknown"

	_, err := user.ParseStatus(state)
	if err == nil {
		t.Fatalf("the status %q parsed, want an error", state)
	}
	if !strings.Contains(err.Error(), state) {
		t.Errorf("the error %q does not name the state %q", err, state)
	}
}
