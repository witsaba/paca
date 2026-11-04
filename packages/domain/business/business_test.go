package business

import (
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/domain/user"
)

func TestNewBuildsABusiness(t *testing.T) {
	t.Parallel()

	built, err := New(1234, KindPersonal, "  Mi Negocio  ", " USD ")
	if err != nil {
		t.Fatalf("the business was refused: %v", err)
	}

	if built.OwnerID != user.ID(1234) {
		t.Errorf("the owner is %d, want 1234", built.OwnerID)
	}
	if built.Kind != KindPersonal {
		t.Errorf("the kind is %q, want %q", built.Kind, KindPersonal)
	}
	if built.Name != "Mi Negocio" {
		t.Errorf("the name is %q, want %q", built.Name, "Mi Negocio")
	}
	if built.Currency != "USD" {
		t.Errorf("the currency is %q, want %q", built.Currency, "USD")
	}
	if built.ID != 0 {
		t.Errorf("the identity is %d, want 0: the storage generates it", built.ID)
	}
}

func TestNewRefusesWhatTheDomainRefuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     Kind
		business string
		currency string
		want     error
	}{
		{
			name:     "an unknown kind",
			kind:     Kind("bakery"),
			business: "Mi Negocio",
			currency: "COP",
			want:     ErrUnknownKind,
		},
		{
			name:     "an empty name",
			kind:     KindPersonal,
			business: "   ",
			currency: "COP",
			want:     ErrInvalidName,
		},
		{
			name:     "an empty currency",
			kind:     KindPersonal,
			business: "Mi Negocio",
			currency: "",
			want:     ErrInvalidCurrency,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(1234, testCase.kind, testCase.business, testCase.currency)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestPersonalBuildsThePersonalBusiness(t *testing.T) {
	t.Parallel()

	built, err := Personal(1234)
	if err != nil {
		t.Fatalf("the personal business was refused: %v", err)
	}

	if built.Kind != KindPersonal {
		t.Errorf("the kind is %q, want %q", built.Kind, KindPersonal)
	}
	if built.Name != DefaultPersonalName {
		t.Errorf("the name is %q, want %q", built.Name, DefaultPersonalName)
	}
	if built.Currency != DefaultCurrency {
		t.Errorf("the currency is %q, want %q", built.Currency, DefaultCurrency)
	}
	if built.OwnerID != user.ID(1234) {
		t.Errorf("the owner is %d, want 1234", built.OwnerID)
	}
}

func TestParseKind(t *testing.T) {
	t.Parallel()

	kind, err := ParseKind("business")
	if err != nil {
		t.Fatalf("the kind was refused: %v", err)
	}
	if kind != KindBusiness {
		t.Errorf("the kind is %q, want %q", kind, KindBusiness)
	}

	if _, err := ParseKind("bakery"); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownKind)
	}
}

func TestKindValid(t *testing.T) {
	t.Parallel()

	if !KindPersonal.Valid() {
		t.Error("the personal kind is not valid")
	}
	if !KindBusiness.Valid() {
		t.Error("the business kind is not valid")
	}
	if Kind("bakery").Valid() {
		t.Error("an unknown kind is valid")
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()

	if KindPersonal.String() != "personal" {
		t.Errorf("the name of the kind is %q, want %q", KindPersonal.String(), "personal")
	}
}

func TestAllKindsReturnsACopy(t *testing.T) {
	t.Parallel()

	all := AllKinds()
	if len(all) != 2 {
		t.Fatalf("the kinds are %v, want 2", all)
	}

	all[0] = KindBusiness

	if AllKinds()[0] != KindPersonal {
		t.Errorf("the first kind is %q, want %q", AllKinds()[0], KindPersonal)
	}
}

func TestErrorsNamesEveryError(t *testing.T) {
	t.Parallel()

	all := Errors()
	if len(all) != 7 {
		t.Fatalf("the catalogue holds %d errors, want 7", len(all))
	}

	for _, err := range all {
		if err == nil {
			t.Error("the catalogue holds an empty error")
		}
	}
}
