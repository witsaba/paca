package party

import (
	"errors"
	"testing"
)

// TestAllRolesNamesEveryRoleOfAParty proves that the list of the roles
// names the two of them, in the order they are presented.
func TestAllRolesNamesEveryRoleOfAParty(t *testing.T) {
	t.Parallel()

	all := AllRoles()
	if len(all) != 2 {
		t.Fatalf("the list holds %d roles, want 2", len(all))
	}

	if all[0] != RoleClient {
		t.Errorf("the first role is %q, want %q", all[0], RoleClient)
	}
	if all[1] != RoleProvider {
		t.Errorf("the second role is %q, want %q", all[1], RoleProvider)
	}
}

// TestAllRolesReturnsACopy proves that the caller of the list never changes
// the vocabulary of the domain.
func TestAllRolesReturnsACopy(t *testing.T) {
	t.Parallel()

	all := AllRoles()
	all[0] = Role("changed")

	if AllRoles()[0] != RoleClient {
		t.Error("the list changed under the caller")
	}
}

// TestRoleValid proves that the roles of a party are the two of them.
func TestRoleValid(t *testing.T) {
	t.Parallel()

	for _, role := range AllRoles() {
		if !role.Valid() {
			t.Errorf("the role %q is not valid", role)
		}
	}

	if Role("owner").Valid() {
		t.Error("an unknown role is valid")
	}
}

// TestRoleString proves that the name of a role is the name the storage
// holds.
func TestRoleString(t *testing.T) {
	t.Parallel()

	if RoleClient.String() != "client" {
		t.Errorf("the name of the role is %q, want %q", RoleClient.String(), "client")
	}
}

// TestParseRole proves that the role of a party is read from the name the
// storage holds, and that an unknown name is refused.
func TestParseRole(t *testing.T) {
	t.Parallel()

	role, err := ParseRole("provider")
	if err != nil {
		t.Fatalf("the role was refused: %v", err)
	}
	if role != RoleProvider {
		t.Errorf("the role is %q, want %q", role, RoleProvider)
	}

	if _, err := ParseRole("owner"); !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownRole)
	}
}

// TestParseRoles proves that the roles of a party are read from the names
// the caller sends, and that an unknown name is refused.
func TestParseRoles(t *testing.T) {
	t.Parallel()

	roles, err := ParseRoles([]string{"client", "provider"})
	if err != nil {
		t.Fatalf("the roles were refused: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("the party holds %d roles, want 2", len(roles))
	}
	if roles[0] != RoleClient || roles[1] != RoleProvider {
		t.Errorf("the roles are %v, want the client and the provider", roles)
	}

	if _, err := ParseRoles([]string{"owner"}); !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("the error is %v, want %v", err, ErrUnknownRole)
	}
}

// TestParseRolesOfNoRole proves that the roles of a party are read from an
// empty list: the domain refuses the party that holds none, and it is the
// party that refuses it.
func TestParseRolesOfNoRole(t *testing.T) {
	t.Parallel()

	roles, err := ParseRoles(nil)
	if err != nil {
		t.Fatalf("the roles were refused: %v", err)
	}
	if len(roles) != 0 {
		t.Errorf("the party holds %d roles, want none", len(roles))
	}
}
