package user

import (
	"errors"
	"testing"
)

func TestNewRegistrationBuildsTheDataOfAPerson(t *testing.T) {
	t.Parallel()

	registration, err := NewRegistration("  Ana María Torres  ", "+57 321-253-9984", "  Ana.Torres@Example.com  ")
	if err != nil {
		t.Fatalf("the registration was refused: %v", err)
	}

	if registration.FullName != "Ana María Torres" {
		t.Errorf("the name is %q, want %q", registration.FullName, "Ana María Torres")
	}
	if registration.Phone != "573212539984" {
		t.Errorf("the phone is %q, want %q", registration.Phone, "573212539984")
	}
	if registration.Email != "ana.torres@example.com" {
		t.Errorf("the address is %q, want %q", registration.Email, "ana.torres@example.com")
	}
}

func TestNewRegistrationAcceptsAPersonWithoutAPhoneAndWithoutAnEmail(t *testing.T) {
	t.Parallel()

	registration, err := NewRegistration("Ana María Torres", "", "")
	if err != nil {
		t.Fatalf("the registration was refused: %v", err)
	}

	if registration.Phone != "" {
		t.Errorf("the phone is %q, want it empty", registration.Phone)
	}
	if registration.Email != "" {
		t.Errorf("the address is %q, want it empty", registration.Email)
	}
}

func TestNewRegistrationRefusesWhatTheDomainRefuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fullName string
		phone    string
		email    string
		want     error
	}{
		{
			name:     "a registration without a name",
			fullName: "   ",
			want:     ErrNameRequired,
		},
		{
			name:     "a phone that is not international",
			fullName: "Ana",
			phone:    "3212539984",
			want:     ErrInvalidPhone,
		},
		{
			name:     "an address that is not an address",
			fullName: "Ana",
			email:    "not an address",
			want:     ErrInvalidEmail,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewRegistration(testCase.fullName, testCase.phone, testCase.email)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}
