package party

import (
	"errors"
	"strings"
	"testing"
)

// TestTheDomainBuildsTheLegalHumanOfTheKind proves the rule of the domain:
// the kind of the party decides the legal human that represents it, and the
// domain builds it from the raw facts.
//
// The rule is the one of the owner: a person is never represented by a legal
// human, a store requires one, and a farm carries one only when it has one.
// A legal figure without the name of the human it belongs to is a figure of
// nobody.
func TestTheDomainBuildsTheLegalHumanOfTheKind(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		kind   Kind
		human  string
		figure string
		want   error
	}{
		{
			name: "a person with no legal human",
			kind: KindPerson,
		},
		{
			name:   "a person with a legal figure",
			kind:   KindPerson,
			figure: "S.A.S.",
			want:   ErrUnexpectedLegalHuman,
		},
		{
			name:  "a person with a legal human",
			kind:  KindPerson,
			human: "Rosalba",
			want:  ErrUnexpectedLegalHuman,
		},
		{
			name: "a store without a legal human",
			kind: KindStore,
			want: ErrMissingLegalHuman,
		},
		{
			name:  "a store with a legal human",
			kind:  KindStore,
			human: " Jaime Ortiz ",
		},
		{
			name: "a farm with no legal human",
			kind: KindFarm,
		},
		{
			name:   "a farm with a legal figure and no name",
			kind:   KindFarm,
			figure: "S.A.S.",
			want:   ErrMissingLegalHuman,
		},
		{
			name:   "a farm with a legal human and its figure",
			kind:   KindFarm,
			human:  "María Gómez",
			figure: " Ltda. ",
		},
		{
			name: "a kind the domain does not know",
			kind: Kind("bakery"),
			want: ErrUnknownKind,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			built, err := LegalHumanOf(testCase.kind, testCase.human, testCase.figure)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("the error is %v, want %v", err, testCase.want)
			}
			if testCase.want != nil {
				return
			}

			if built.Name != strings.TrimSpace(testCase.human) {
				t.Errorf("the name of the legal human is %q, want %q", built.Name, strings.TrimSpace(testCase.human))
			}
			if built.Figure != strings.TrimSpace(testCase.figure) {
				t.Errorf("the legal figure is %q, want %q", built.Figure, strings.TrimSpace(testCase.figure))
			}
		})
	}
}
