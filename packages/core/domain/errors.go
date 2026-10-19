package domain

import "errors"

// Code is the stable, machine-readable identifier of a failure.
//
// The set of codes is a contract with the clients of PACA: an agent reads the
// code and decides what to do, so a code never changes meaning and never
// disappears without a new version of the contract.
type Code string

const (
	// CodeValidationFailed means the request was understood and is not
	// acceptable: a missing field, a malformed value, a rule of the business
	// that the request breaks.
	CodeValidationFailed Code = "validation_failed"

	// CodeNotFound means the entity the request names does not exist.
	CodeNotFound Code = "not_found"

	// CodeDuplicateRequest means the request was already processed under the
	// same idempotency key. It is never used for a uniqueness violation: a
	// uniqueness violation is a validation failure.
	CodeDuplicateRequest Code = "duplicate_request"

	// CodeCurrencyNotSupported means the amount is in a currency that MVP1
	// does not handle. MVP1 is Colombian pesos only.
	CodeCurrencyNotSupported Code = "currency_not_supported_in_mvp1"

	// CodeAccountInactive means the account named by the request is closed,
	// so no money can move through it.
	CodeAccountInactive Code = "account_inactive"

	// CodeLoopRequiredForUnsafeCall means the call needs a loop, and a loop
	// is a request the agent has to make: an operation that would move money
	// to a destination that can be reached from the origin is refused, and
	// the agent has to break the cycle itself.
	CodeLoopRequiredForUnsafeCall Code = "loop_required_for_unsafe_call"

	// CodeInternal means something unexpected happened. Its detail never
	// carries the internals of the system: no driver message, no file path,
	// no stack frame.
	CodeInternal Code = "internal_error"
)

// Error is a failure of PACA that the outside world can observe.
//
// It carries a stable code and a human-readable detail. The cause that
// produced it, when there is one, stays reachable through Unwrap and never
// reaches the detail: the detail travels to an agent, and an agent must not
// see the internals of the system.
type Error struct {
	code   Code
	detail string
	cause  error
}

// New returns an error with the given code and detail.
func New(code Code, detail string) *Error {
	return &Error{code: code, detail: detail}
}

// Wrap returns an error with the given code and detail that keeps the cause
// reachable through errors.Is and errors.As.
//
// The cause is never part of the detail and never part of the message: the
// message is what a caller may show, and the detail is what an agent reads.
func Wrap(code Code, detail string, cause error) *Error {
	return &Error{code: code, detail: detail, cause: cause}
}

// Code returns the stable code of the error.
func (e *Error) Code() Code {
	return e.code
}

// Detail returns the human-readable detail of the error.
func (e *Error) Detail() string {
	return e.detail
}

// Unwrap returns the cause of the error, or nil when there is none.
func (e *Error) Unwrap() error {
	return e.cause
}

// Error returns the message of the error: the code and the detail, and
// nothing else.
func (e *Error) Error() string {
	return string(e.code) + ": " + e.detail
}

// CodeOf returns the code of the error, looking through the wrapping.
//
// An error that is not a domain error has no code of its own: the caller
// reports it as an internal error, because it is a failure the system did not
// classify. A nil error has no code at all.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}

	var domainErr *Error
	if errors.As(err, &domainErr) {
		return domainErr.code
	}

	return CodeInternal
}

// DetailOf returns the detail of the error, looking through the wrapping.
//
// The detail of an error that is not a domain error is empty: an unclassified
// failure has nothing safe to show.
func DetailOf(err error) string {
	if err == nil {
		return ""
	}

	var domainErr *Error
	if errors.As(err, &domainErr) {
		return domainErr.detail
	}

	return ""
}
