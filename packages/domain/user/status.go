package user

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidTransition is the error of a status that may not move to the
// status it was asked to move to.
var ErrInvalidTransition = errors.New("the status cannot move to that state")

// ErrObservationRequired is the error of a status that must carry an
// observation and does not.
var ErrObservationRequired = errors.New("the status needs an observation")

// ErrUnexpectedObservation is the error of an observation offered to a status
// that carries none.
var ErrUnexpectedObservation = errors.New("the status does not take an observation")

// transitions is the map of the state machine of the user.
//
// It holds every move the user accepts. A status that is not a key of the map
// is a status the domain does not know, and it moves nowhere.
//
// No move leads back to the onboarding: that state is the entry of the
// machine, the state of a person that has just been created, and it is never
// a destination.
var transitions = map[Status][]Status{
	StatusOnboarding: {StatusActive, StatusInactive, StatusBlocked},
	StatusActive:     {StatusInactive, StatusBlocked},
	StatusInactive:   {StatusActive, StatusBlocked},
	StatusBlocked:    {StatusActive, StatusInactive},
}

// AllowedTargets returns the statuses the status may move to.
//
// The caller receives a copy: the map never changes under it.
func (s Status) AllowedTargets() []Status {
	targets, ok := transitions[s]
	if !ok {
		return nil
	}

	allowed := make([]Status, len(targets))
	copy(allowed, targets)
	return allowed
}

// CanMoveTo reports whether the status may move to the target.
//
// A status never moves to itself: a move is a change.
func (s Status) CanMoveTo(target Status) bool {
	for _, candidate := range transitions[s] {
		if candidate == target {
			return true
		}
	}
	return false
}

// RequiresObservations reports whether a status must carry an observation.
//
// The two closed states carry the reason why the person is not using the
// product. The two open states carry none: there is nothing to explain.
func RequiresObservations(status Status) bool {
	return status == StatusInactive || status == StatusBlocked
}

// ValidateObservation checks an observation against the status it belongs to.
//
// The observation of a closed status is mandatory, and the observation of an
// open status is refused: the two rules are the same rule seen from both
// sides.
func ValidateObservation(status Status, observation string) error {
	trimmed := strings.TrimSpace(observation)

	if RequiresObservations(status) {
		if trimmed == "" {
			return fmt.Errorf("%w: the status %q needs one", ErrObservationRequired, status)
		}
		return nil
	}

	if trimmed != "" {
		return fmt.Errorf("%w: the status %q takes none", ErrUnexpectedObservation, status)
	}
	return nil
}
