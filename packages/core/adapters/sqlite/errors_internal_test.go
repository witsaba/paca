package sqlite

import "testing"

// The mapping of the result codes of SQLite to the details of the domain is
// a table, and a table is worth pinning: a code that falls through to the
// default would say the wrong thing about every constraint it does not name.
//
// The test is inside the package because the mapping is not exported: it is
// an implementation of the error contract, and the contract itself is tested
// from the outside.
func TestConstraintDetailNamesEveryResultCode(t *testing.T) {
	cases := map[int]string{
		codeUnique:     detailDuplicate,
		codePrimaryKey: detailDuplicate,
		codeNotNull:    detailRequired,
		codeCheck:      detailCheck,
		codeForeignKey: detailForeignKey,
		codeTrigger:    detailConstraint,
		// A constraint that SQLite reports without naming is a rule of the
		// schema that the adapter does not know.
		codeConstraint: detailConstraint,
		// A result code that is not a constraint at all falls through to the
		// same rule, and never to a detail of its own.
		9999: detailConstraint,
	}

	for code, want := range cases {
		if got := constraintDetail(code); got != want {
			t.Errorf("the detail of the result code %d is %q, want %q", code, got, want)
		}
	}
}
