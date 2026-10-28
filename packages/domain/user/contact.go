package user

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalidPhone is the error of a phone number that is not an
// international number.
var ErrInvalidPhone = errors.New("the phone number is not an international number")

// ErrInvalidEmail is the error of an address that is not an address.
var ErrInvalidEmail = errors.New("the email address is not valid")

// ErrInvalidID is the error of a document ID that is not a number.
var ErrInvalidID = errors.New("the document ID is not a number")

// The bounds of a phone number, in digits.
//
// Eleven digits is the shortest international number: a country code of at
// least one digit and a national number of ten. Fifteen is the longest the
// ITU allows. A number inside these bounds carries a country code, and a
// national number without one is refused: ten digits are exactly the length
// of a Colombian number, and that is the number that must be written with
// its country code.
const (
	minPhoneDigits = 11
	maxPhoneDigits = 15
)

// phoneSeparators are the characters a phone number may carry between its
// digits: the plus of the country code, the spaces, the dashes, the dots and
// the parentheses.
const phoneSeparators = " -.()"

// idSeparators are the characters a document ID may carry between its digits.
const idSeparators = " .-"

// emailPattern is the shape of an address.
//
// It is the same rule the database holds: a name, an at sign, a domain with a
// dot in it, and no space anywhere.
var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// CleanPhone returns the digits of a phone number.
//
// The number is cleaned the way a person writes it: the plus of the country
// code, the spaces, the dashes, the dots and the parentheses go away, and
// what is left are the digits. The number of a person who writes it the way
// they say it is the number that is stored.
//
// An empty number is not a number: it is the absence of one, and it is not
// an error. A number that carries anything else, or that is too short or too
// long to be an international number, is refused.
func CleanPhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	trimmed = strings.TrimPrefix(trimmed, "+")

	var digits strings.Builder
	for _, character := range trimmed {
		switch {
		case character >= '0' && character <= '9':
			digits.WriteRune(character)
		case strings.ContainsRune(phoneSeparators, character):
		default:
			return "", fmt.Errorf("%w: %q holds %q", ErrInvalidPhone, raw, character)
		}
	}

	cleaned := digits.String()
	if len(cleaned) < minPhoneDigits || len(cleaned) > maxPhoneDigits {
		return "", fmt.Errorf("%w: %q holds %d digits, want between %d and %d",
			ErrInvalidPhone, raw, len(cleaned), minPhoneDigits, maxPhoneDigits)
	}

	return cleaned, nil
}

// CleanEmail returns an address without the spaces around it.
//
// An empty address is the absence of one, and it is not an error. An address
// that does not look like an address is refused.
func CleanEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	if !emailPattern.MatchString(trimmed) {
		return "", fmt.Errorf("%w: %q", ErrInvalidEmail, raw)
	}

	return trimmed, nil
}

// ParseID returns the ID of a document written as text.
//
// A person writes the document with dots, with spaces or with a dash, and
// all of them are the same number. The characters that are not digits are
// ignored, and what is left must be a number.
func ParseID(raw string) (ID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("%w: the document is empty", ErrInvalidID)
	}

	var digits strings.Builder
	for _, character := range trimmed {
		switch {
		case character >= '0' && character <= '9':
			digits.WriteRune(character)
		case strings.ContainsRune(idSeparators, character):
		default:
			return 0, fmt.Errorf("%w: %q holds %q", ErrInvalidID, raw, character)
		}
	}

	cleaned := digits.String()
	if cleaned == "" {
		return 0, fmt.Errorf("%w: %q holds no digits", ErrInvalidID, raw)
	}

	value, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not a number that fits", ErrInvalidID, raw)
	}

	return ID(value), nil
}

// Details holds the optional data of a person.
//
// A field that is empty is a field that is not set. The storage never writes
// an empty string to the database: it writes NULL.
type Details struct {
	// FullName is the complete name of the person.
	FullName string

	// Phone is the phone number of the person, as digits with the country
	// code.
	Phone string

	// Email is the email address of the person.
	Email string
}

// NewDetails cleans and validates the data of a person.
//
// The name loses the spaces around it, the phone loses the punctuation of the
// way a person writes it, and the email is checked. The data that is not
// given stays empty.
func NewDetails(fullName, phone, email string) (Details, error) {
	cleanPhone, err := CleanPhone(phone)
	if err != nil {
		return Details{}, err
	}

	cleanEmail, err := CleanEmail(email)
	if err != nil {
		return Details{}, err
	}

	return Details{
		FullName: strings.TrimSpace(fullName),
		Phone:    cleanPhone,
		Email:    cleanEmail,
	}, nil
}
