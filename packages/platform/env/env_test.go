package env_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/packages/platform/env"
)

func TestTheVersionOfTheServiceHasOneName(t *testing.T) {
	t.Parallel()

	if env.ServiceVersion != "PACA_SERVICE_VERSION" {
		t.Fatalf("service version = %q, want %q", env.ServiceVersion, "PACA_SERVICE_VERSION")
	}
}

func TestTheEnvironmentOfTheDatabaseHasOneNameInTheMonorepository(t *testing.T) {
	t.Parallel()

	names := map[string]string{
		"the connection string of the database": env.DatabaseURL,
		"the host of the database":              env.DatabaseHost,
		"the port of the database":              env.DatabasePort,
		"the role of the database":              env.PostgresUser,
		"the password of the role":              env.PostgresPassword,
		"the name of the database":              env.PostgresDB,
	}

	want := map[string]string{
		"the connection string of the database": "PACA_DATABASE_URL",
		"the host of the database":              "PACA_DATABASE_HOST",
		"the port of the database":              "PACA_DATABASE_PORT",
		"the role of the database":              "POSTGRES_USER",
		"the password of the role":              "POSTGRES_PASSWORD",
		"the name of the database":              "POSTGRES_DB",
	}

	for what, name := range names {
		if name != want[what] {
			t.Errorf("%s = %q, want %q", what, name, want[what])
		}
	}
}

func TestTheDatabaseOfTheMonorepositoryHasOneHostAndOnePort(t *testing.T) {
	t.Parallel()

	if env.DefaultDatabaseHost != "postgres" {
		t.Errorf("default host = %q, want %q", env.DefaultDatabaseHost, "postgres")
	}
	if env.DefaultDatabasePort != "5432" {
		t.Errorf("default port = %q, want %q", env.DefaultDatabasePort, "5432")
	}
}

func TestValueReadsTheValueOfAVariable(t *testing.T) {
	t.Parallel()

	value, ok := env.Value(func(string) (string, bool) {
		return "postgres", true
	}, env.DatabaseHost)

	if !ok {
		t.Fatalf("Value() found no value, want one")
	}
	if value != "postgres" {
		t.Fatalf("value = %q, want %q", value, "postgres")
	}
}

func TestValueTrimsTheSpacesOfTheValue(t *testing.T) {
	t.Parallel()

	value, ok := env.Value(func(string) (string, bool) {
		return "  postgres  ", true
	}, env.DatabaseHost)

	if !ok {
		t.Fatalf("Value() found no value, want one")
	}
	if value != "postgres" {
		t.Fatalf("value = %q, want %q", value, "postgres")
	}
}

func TestValueTreatsAnEmptyValueAsNoValue(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "   "} {
		value, ok := env.Value(func(string) (string, bool) {
			return raw, true
		}, env.DatabaseHost)

		if ok {
			t.Errorf("Value() found the value %q of %q, want no value", value, raw)
		}
	}
}

func TestValueReportsAVariableThatIsNotSet(t *testing.T) {
	t.Parallel()

	value, ok := env.Value(func(string) (string, bool) {
		return "", false
	}, env.DatabaseHost)

	if ok {
		t.Fatalf("Value() found the value %q, want no value", value)
	}
}

func TestSecretKeepsTheSpacesOfThePassword(t *testing.T) {
	t.Parallel()

	password, ok := env.Secret(func(string) (string, bool) {
		return " pa ss word ", true
	}, env.PostgresPassword)

	if !ok {
		t.Fatalf("Secret() found no password, want one")
	}
	if password != " pa ss word " {
		t.Fatalf("password = %q, want %q", password, " pa ss word ")
	}
}

func TestSecretTreatsAnEmptyPasswordAsNoPassword(t *testing.T) {
	t.Parallel()

	password, ok := env.Secret(func(string) (string, bool) {
		return "", true
	}, env.PostgresPassword)

	if ok {
		t.Fatalf("Secret() found the password %q, want no password", password)
	}
}

func TestDurationReadsADuration(t *testing.T) {
	t.Parallel()

	duration, err := env.Duration("PACA_TEST_TIMEOUT", "30s")
	if err != nil {
		t.Fatalf("Duration() = %v", err)
	}
	if duration != 30*time.Second {
		t.Fatalf("duration = %s, want %s", duration, 30*time.Second)
	}
}

func TestDurationTrimsTheSpacesOfTheValue(t *testing.T) {
	t.Parallel()

	duration, err := env.Duration("PACA_TEST_TIMEOUT", " 5s ")
	if err != nil {
		t.Fatalf("Duration() = %v", err)
	}
	if duration != 5*time.Second {
		t.Fatalf("duration = %s, want %s", duration, 5*time.Second)
	}
}

func TestDurationRejectsAValueThatIsNotADuration(t *testing.T) {
	t.Parallel()

	duration, err := env.Duration("PACA_TEST_TIMEOUT", "soon")
	if err == nil {
		t.Fatalf("Duration() = %s, want an error", duration)
	}

	want := `PACA_TEST_TIMEOUT: "soon" is not a duration`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestBoolReadsABoolean(t *testing.T) {
	t.Parallel()

	value, err := env.Bool(func(string) (string, bool) {
		return "true", true
	}, "PACA_TEST_SWITCH")

	if err != nil {
		t.Fatalf("Bool() = %v", err)
	}
	if !value {
		t.Fatal("Bool() = false, want true")
	}
}

func TestBoolReadsAFalse(t *testing.T) {
	t.Parallel()

	value, err := env.Bool(func(string) (string, bool) {
		return "FALSE", true
	}, "PACA_TEST_SWITCH")

	if err != nil {
		t.Fatalf("Bool() = %v", err)
	}
	if value {
		t.Fatal("Bool() = true, want false")
	}
}

func TestBoolReadsAVariableThatIsNotSetAsFalse(t *testing.T) {
	t.Parallel()

	value, err := env.Bool(func(string) (string, bool) {
		return "", false
	}, "PACA_TEST_SWITCH")

	if err != nil {
		t.Fatalf("Bool() = %v", err)
	}
	if value {
		t.Fatal("Bool() = true, want false")
	}
}

func TestBoolRejectsAValueThatIsNotABoolean(t *testing.T) {
	t.Parallel()

	_, err := env.Bool(func(string) (string, bool) {
		return "maybe", true
	}, "PACA_TEST_SWITCH")
	if err == nil {
		t.Fatal("Bool() = nil, want an error")
	}

	want := `PACA_TEST_SWITCH: "maybe" is not true or false`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
