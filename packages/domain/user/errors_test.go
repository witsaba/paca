package user_test

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/user"
)

// TestTheCatalogueCarriesEveryErrorOfThePort proves that the catalogue of
// the errors names every error the port of the storage can return.
//
// A caller proves its mapping against the catalogue: an error that the
// catalogue does not name is an error that no caller knows how to say.
func TestTheCatalogueCarriesEveryErrorOfThePort(t *testing.T) {
	t.Parallel()

	every := []error{
		user.ErrNotFound,
		user.ErrAlreadyExists,
		user.ErrEmailTaken,
		user.ErrPhoneTaken,
		user.ErrConcurrentChange,
		user.ErrInvalidValue,
		user.ErrNoStorage,
		user.ErrInvalidTransition,
		user.ErrObservationRequired,
		user.ErrUnexpectedObservation,
		user.ErrInvalidPhone,
		user.ErrInvalidEmail,
		user.ErrInvalidID,
		user.ErrUnknownStatus,
	}

	catalogue := user.Errors()

	if len(catalogue) != len(every) {
		t.Fatalf("the catalogue carries %d error(s), want %d", len(catalogue), len(every))
	}

	for _, err := range every {
		found := false
		for _, listed := range catalogue {
			if errors.Is(listed, err) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the catalogue does not carry %q", err)
		}
	}
}
