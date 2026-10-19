package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/domain"
)

// Every failure the agent can observe carries one of the stable codes of the
// C7 taxonomy, and the code is the one the ADR fixes.
func TestErrorCarriesItsStableCode(t *testing.T) {
	err := domain.New(domain.CodeNotFound, "account 7 does not exist")

	if got := err.Code(); got != domain.CodeNotFound {
		t.Errorf("the code of the error is %q, want %q", got, domain.CodeNotFound)
	}

	if got := err.Detail(); got != "account 7 does not exist" {
		t.Errorf("the detail of the error is %q, want %q", got, "account 7 does not exist")
	}
}

// The taxonomy is exactly the seven codes of the ADR. A new code is a change
// of the contract, so the set is pinned here.
func TestTaxonomyHoldsTheSevenStableCodes(t *testing.T) {
	codes := []domain.Code{
		domain.CodeValidationFailed,
		domain.CodeNotFound,
		domain.CodeDuplicateRequest,
		domain.CodeCurrencyNotSupported,
		domain.CodeAccountInactive,
		domain.CodeLoopRequiredForUnsafeCall,
		domain.CodeInternal,
	}

	want := []string{
		"validation_failed",
		"not_found",
		"duplicate_request",
		"currency_not_supported_in_mvp1",
		"account_inactive",
		"loop_required_for_unsafe_call",
		"internal_error",
	}

	for i, code := range codes {
		if string(code) != want[i] {
			t.Errorf("the code %d is %q, want %q", i, code, want[i])
		}
	}
}

// The detail of an error never carries the internals of the system that
// produced it: no driver message, no file path, no stack frame.
func TestDetailNeverCarriesTheCause(t *testing.T) {
	cause := errors.New(`SQLITE_CONSTRAINT: UNIQUE constraint failed: accounts.phone (/tmp/paca/paca.db)`)

	err := domain.Wrap(domain.CodeValidationFailed, "the phone number is already registered", cause)

	if strings.Contains(err.Detail(), "SQLITE_CONSTRAINT") {
		t.Errorf("the detail leaks the driver message: %q", err.Detail())
	}

	if strings.Contains(err.Detail(), "/tmp/paca") {
		t.Errorf("the detail leaks a file path: %q", err.Detail())
	}

	if !errors.Is(err, cause) {
		t.Error("the wrapped error does not keep its cause reachable")
	}
}

// An error of the system is not a failure the agent can act on, so it is
// reported as internal_error and never as one of the business codes.
func TestUntypedErrorIsAnInternalError(t *testing.T) {
	err := errors.New("the disk is on fire")

	if got := domain.CodeOf(err); got != domain.CodeInternal {
		t.Errorf("the code of an untyped error is %q, want %q", got, domain.CodeInternal)
	}
}

// The code of a wrapped domain error survives the wrapping, so a transport
// can report the business code and not the code of the wrapper.
func TestCodeOfFindsTheCodeThroughWrapping(t *testing.T) {
	err := domain.New(domain.CodeDuplicateRequest, "the request was already processed")
	wrapped := errors.Join(errors.New("context lost"), err)

	if got := domain.CodeOf(wrapped); got != domain.CodeDuplicateRequest {
		t.Errorf("the code of the wrapped error is %q, want %q", got, domain.CodeDuplicateRequest)
	}
}

// A nil error has no code: a caller that has nothing to report reports
// nothing.
func TestCodeOfNilIsEmpty(t *testing.T) {
	if got := domain.CodeOf(nil); got != "" {
		t.Errorf("the code of a nil error is %q, want an empty code", got)
	}
}

// The message of an error is the code and the detail, because the detail is
// the only part that is safe to show.
func TestErrorMessageIsTheCodeAndTheDetail(t *testing.T) {
	err := domain.New(domain.CodeAccountInactive, "account 3 is closed")

	want := "account_inactive: account 3 is closed"
	if got := err.Error(); got != want {
		t.Errorf("the message of the error is %q, want %q", got, want)
	}
}

// An error without a detail has nothing to add to its code.
func TestErrorMessageWithoutDetailIsTheCode(t *testing.T) {
	err := domain.New(domain.CodeInternal, "")

	if got := err.Error(); got != "internal_error" {
		t.Errorf("the message of the error is %q, want %q", got, "internal_error")
	}
}
