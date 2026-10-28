package user_test

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/user"
)

func TestStatusAllowedTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status user.Status
		want   []user.Status
	}{
		{
			name:   "an onboarding user may become active, inactive or blocked",
			status: user.StatusOnboarding,
			want:   []user.Status{user.StatusActive, user.StatusInactive, user.StatusBlocked},
		},
		{
			name:   "an active user may become inactive or blocked",
			status: user.StatusActive,
			want:   []user.Status{user.StatusInactive, user.StatusBlocked},
		},
		{
			name:   "an inactive user may become active or blocked",
			status: user.StatusInactive,
			want:   []user.Status{user.StatusActive, user.StatusBlocked},
		},
		{
			name:   "a blocked user may become active or inactive",
			status: user.StatusBlocked,
			want:   []user.Status{user.StatusActive, user.StatusInactive},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.status.AllowedTargets()

			if len(got) != len(tc.want) {
				t.Fatalf("the targets are %v, want %v", got, tc.want)
			}
			for i, target := range tc.want {
				if got[i] != target {
					t.Errorf("the target %d is %q, want %q", i, got[i], target)
				}
			}
		})
	}
}

func TestStatusAllowedTargetsReturnsACopy(t *testing.T) {
	t.Parallel()

	first := user.StatusOnboarding.AllowedTargets()
	first[0] = user.StatusBlocked

	second := user.StatusOnboarding.AllowedTargets()

	if second[0] != user.StatusActive {
		t.Errorf("the targets were tampered through the copy: %q", second[0])
	}
}

func TestStatusCanMoveTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from user.Status
		to   user.Status
		want bool
	}{
		{name: "onboarding to active", from: user.StatusOnboarding, to: user.StatusActive, want: true},
		{name: "onboarding to inactive", from: user.StatusOnboarding, to: user.StatusInactive, want: true},
		{name: "onboarding to blocked", from: user.StatusOnboarding, to: user.StatusBlocked, want: true},
		{name: "active to inactive", from: user.StatusActive, to: user.StatusInactive, want: true},
		{name: "active to blocked", from: user.StatusActive, to: user.StatusBlocked, want: true},
		{name: "inactive to active", from: user.StatusInactive, to: user.StatusActive, want: true},
		{name: "inactive to blocked", from: user.StatusInactive, to: user.StatusBlocked, want: true},
		{name: "blocked to active", from: user.StatusBlocked, to: user.StatusActive, want: true},
		{name: "blocked to inactive", from: user.StatusBlocked, to: user.StatusInactive, want: true},
		{name: "onboarding never goes back to onboarding", from: user.StatusOnboarding, to: user.StatusOnboarding, want: false},
		{name: "active never goes back to onboarding", from: user.StatusActive, to: user.StatusOnboarding, want: false},
		{name: "inactive never goes back to onboarding", from: user.StatusInactive, to: user.StatusOnboarding, want: false},
		{name: "blocked never goes back to onboarding", from: user.StatusBlocked, to: user.StatusOnboarding, want: false},
		{name: "a status never moves to itself", from: user.StatusActive, to: user.StatusActive, want: false},
		{name: "an unknown target is refused", from: user.StatusActive, to: user.Status("unknown"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.from.CanMoveTo(tc.to); got != tc.want {
				t.Errorf("the move from %q to %q is allowed: %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestStatusRequiresObservations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status user.Status
		want   bool
	}{
		{name: "an onboarding user carries no observation", status: user.StatusOnboarding, want: false},
		{name: "an active user carries no observation", status: user.StatusActive, want: false},
		{name: "an inactive user carries an observation", status: user.StatusInactive, want: true},
		{name: "a blocked user carries an observation", status: user.StatusBlocked, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := user.RequiresObservations(tc.status); got != tc.want {
				t.Errorf("the status %q requires an observation: %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestValidateObservation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      user.Status
		observation string
		wantErr     error
	}{
		{
			name:        "an inactive user needs an observation",
			status:      user.StatusInactive,
			observation: "",
			wantErr:     user.ErrObservationRequired,
		},
		{
			name:        "a blocked user needs an observation",
			status:      user.StatusBlocked,
			observation: "  ",
			wantErr:     user.ErrObservationRequired,
		},
		{
			name:        "an inactive user with an observation is fine",
			status:      user.StatusInactive,
			observation: "the person asked to leave",
			wantErr:     nil,
		},
		{
			name:        "an active user takes no observation",
			status:      user.StatusActive,
			observation: "",
			wantErr:     nil,
		},
		{
			name:        "an active user with an observation is refused",
			status:      user.StatusActive,
			observation: "the payment was confirmed",
			wantErr:     user.ErrUnexpectedObservation,
		},
		{
			name:        "an onboarding user takes no observation",
			status:      user.StatusOnboarding,
			observation: "",
			wantErr:     nil,
		},
		{
			name:        "an onboarding user with an observation is refused",
			status:      user.StatusOnboarding,
			observation: "the person just arrived",
			wantErr:     user.ErrUnexpectedObservation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := user.ValidateObservation(tc.status, tc.observation)

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("the observation was refused: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("the error is %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestAllTargetsReturnsTheStatesAUserCanReach(t *testing.T) {
	t.Parallel()

	targets := user.AllTargets()

	want := []user.Status{user.StatusActive, user.StatusInactive, user.StatusBlocked}
	if len(targets) != len(want) {
		t.Fatalf("the targets are %v, want %v", targets, want)
	}
	for i, target := range targets {
		if target != want[i] {
			t.Fatalf("the targets are %v, want %v", targets, want)
		}
	}
}

func TestAllTargetsNeverLeadsBackToTheOnboarding(t *testing.T) {
	t.Parallel()

	for _, target := range user.AllTargets() {
		if target == user.StatusOnboarding {
			t.Fatalf("the onboarding is a target of the map: %v", user.AllTargets())
		}
	}
}

func TestAllTargetsReturnsACopy(t *testing.T) {
	t.Parallel()

	targets := user.AllTargets()
	if len(targets) == 0 {
		t.Fatal("there are no targets")
	}

	targets[0] = user.Status("changed")

	if user.AllTargets()[0] == user.Status("changed") {
		t.Fatal("the list of the targets changed under the caller")
	}
}
