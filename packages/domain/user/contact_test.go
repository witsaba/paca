package user_test

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/user"
)

func TestCleanPhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{name: "the example of the specification", raw: "+57 321-253-9984", want: "573212539984"},
		{name: "the parentheses are removed", raw: "+57 (321) 253-9984", want: "573212539984"},
		{name: "the dots are removed", raw: "+57.321.253.9984", want: "573212539984"},
		{name: "a number that is already clean stays clean", raw: "573212539984", want: "573212539984"},
		{name: "the spaces around the number are removed", raw: "  573212539984  ", want: "573212539984"},
		{name: "an empty number is not a number", raw: "", want: ""},
		{name: "a number without a country code is refused", raw: "3212539984", wantErr: user.ErrInvalidPhone},
		{name: "a number that is too short is refused", raw: "+57 321-253-99", wantErr: user.ErrInvalidPhone},
		{name: "a number that is too long is refused", raw: "5732125399841234", wantErr: user.ErrInvalidPhone},
		{name: "a number with letters is refused", raw: "+57 321-253-CASA", wantErr: user.ErrInvalidPhone},
		{name: "a number with a symbol is refused", raw: "+57 321#253#9984", wantErr: user.ErrInvalidPhone},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := user.CleanPhone(tc.raw)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("the error is %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("the phone was refused: %v", err)
			}
			if got != tc.want {
				t.Errorf("the phone is %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCleanEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{name: "an address stays as it is", raw: "ana@example.com", want: "ana@example.com"},
		{name: "the spaces around the address are removed", raw: "  ana@example.com  ", want: "ana@example.com"},
		{name: "the case of the address is kept", raw: "Ana@Example.com", want: "Ana@Example.com"},
		{name: "an empty address is not an address", raw: "", want: ""},
		{name: "an address without a name is refused", raw: "@example.com", wantErr: user.ErrInvalidEmail},
		{name: "an address without a domain is refused", raw: "ana@", wantErr: user.ErrInvalidEmail},
		{name: "an address without a dot in the domain is refused", raw: "ana@example", wantErr: user.ErrInvalidEmail},
		{name: "an address with two names is refused", raw: "ana@example.com@other.com", wantErr: user.ErrInvalidEmail},
		{name: "an address with a space is refused", raw: "ana example@example.com", wantErr: user.ErrInvalidEmail},
		{name: "a sentence is refused", raw: "write me at ana@example.com", wantErr: user.ErrInvalidEmail},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := user.CleanEmail(tc.raw)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("the error is %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("the email was refused: %v", err)
			}
			if got != tc.want {
				t.Errorf("the email is %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewDetails(t *testing.T) {
	t.Parallel()

	t.Run("the data of a person is cleaned", func(t *testing.T) {
		t.Parallel()

		details, err := user.NewDetails("  Ana María Pérez ", "+57 321-253-9984", " ana@example.com ")
		if err != nil {
			t.Fatalf("the details were refused: %v", err)
		}

		if details.FullName != "Ana María Pérez" {
			t.Errorf("the name is %q, want %q", details.FullName, "Ana María Pérez")
		}
		if details.Phone != "573212539984" {
			t.Errorf("the phone is %q, want %q", details.Phone, "573212539984")
		}
		if details.Email != "ana@example.com" {
			t.Errorf("the email is %q, want %q", details.Email, "ana@example.com")
		}
	})

	t.Run("the optional data may be missing", func(t *testing.T) {
		t.Parallel()

		details, err := user.NewDetails("", "", "")
		if err != nil {
			t.Fatalf("the details were refused: %v", err)
		}

		if details.FullName != "" || details.Phone != "" || details.Email != "" {
			t.Errorf("the details are %+v, want them empty", details)
		}
	})

	t.Run("a bad phone is refused", func(t *testing.T) {
		t.Parallel()

		if _, err := user.NewDetails("Ana", "3212539984", "ana@example.com"); !errors.Is(err, user.ErrInvalidPhone) {
			t.Fatalf("the error is %v, want %v", err, user.ErrInvalidPhone)
		}
	})

	t.Run("a bad email is refused", func(t *testing.T) {
		t.Parallel()

		if _, err := user.NewDetails("Ana", "573212539984", "not an address"); !errors.Is(err, user.ErrInvalidEmail) {
			t.Fatalf("the error is %v, want %v", err, user.ErrInvalidEmail)
		}
	})
}

func TestParseID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    user.ID
		wantErr error
	}{
		{name: "the digits of a document", raw: "1234567", want: 1234567},
		{name: "a document written with dots", raw: "1.234.567", want: 1234567},
		{name: "a document written with spaces", raw: "1 234 567", want: 1234567},
		{name: "a document written with a dash", raw: "1-234-567", want: 1234567},
		{name: "the spaces around the document are removed", raw: "  1234567  ", want: 1234567},
		{name: "an empty document is refused", raw: "", wantErr: user.ErrInvalidID},
		{name: "a document with letters is refused", raw: "CC1234567", wantErr: user.ErrInvalidID},
		{name: "a document with a sign is refused", raw: "+1234567", wantErr: user.ErrInvalidID},
		{name: "a document that is too long is refused", raw: "12345678901234567890", wantErr: user.ErrInvalidID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := user.ParseID(tc.raw)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("the error is %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("the document was refused: %v", err)
			}
			if got != tc.want {
				t.Errorf("the document is %d, want %d", got, tc.want)
			}
		})
	}
}
