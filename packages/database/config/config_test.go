package config_test

import (
	"net/url"
	"testing"

	"github.com/witsaba/paca/packages/database/config"
)

func TestConnectionStringAssemblesTheURLOfTheCredentials(t *testing.T) {
	t.Parallel()

	connection := config.ConnectionString(config.Credentials{
		User:     "paca",
		Password: "s3cr3t",
		Database: "paca",
		Host:     "postgres",
		Port:     "5432",
	})

	want := "postgres://paca:s3cr3t@postgres:5432/paca?sslmode=disable"
	if connection != want {
		t.Fatalf("connection string = %q, want %q", connection, want)
	}
}

func TestConnectionStringEscapesTheCharactersTheURLReserves(t *testing.T) {
	t.Parallel()

	connection := config.ConnectionString(config.Credentials{
		User:     "paca",
		Password: "p@ss/wor:d",
		Database: "paca",
		Host:     "postgres",
		Port:     "5432",
	})

	want := "postgres://paca:p%40ss%2Fwor%3Ad@postgres:5432/paca?sslmode=disable"
	if connection != want {
		t.Fatalf("connection string = %q, want %q", connection, want)
	}

	parsed, err := url.Parse(connection)
	if err != nil {
		t.Fatalf("parse %q: %v", connection, err)
	}
	if parsed.User == nil {
		t.Fatalf("connection string %q carries no user", connection)
	}
	password, ok := parsed.User.Password()
	if !ok {
		t.Fatalf("connection string %q carries no password", connection)
	}
	if password != "p@ss/wor:d" {
		t.Fatalf("password = %q, want %q", password, "p@ss/wor:d")
	}
	if parsed.Host != "postgres:5432" {
		t.Fatalf("host = %q, want %q", parsed.Host, "postgres:5432")
	}
	if parsed.Path != "/paca" {
		t.Fatalf("path = %q, want %q", parsed.Path, "/paca")
	}
	if parsed.RawQuery != "sslmode=disable" {
		t.Fatalf("query = %q, want %q", parsed.RawQuery, "sslmode=disable")
	}
}

func TestConnectionStringKeepsTheSpaceOfThePassword(t *testing.T) {
	t.Parallel()

	connection := config.ConnectionString(config.Credentials{
		User:     "paca",
		Password: "p@ss/wor d",
		Database: "paca",
		Host:     "postgres",
		Port:     "5432",
	})

	want := "postgres://paca:p%40ss%2Fwor%20d@postgres:5432/paca?sslmode=disable"
	if connection != want {
		t.Fatalf("connection string = %q, want %q", connection, want)
	}

	parsed, err := url.Parse(connection)
	if err != nil {
		t.Fatalf("parse %q: %v", connection, err)
	}
	if parsed.User == nil {
		t.Fatalf("connection string %q carries no user", connection)
	}
	password, ok := parsed.User.Password()
	if !ok {
		t.Fatalf("connection string %q carries no password", connection)
	}
	if password != "p@ss/wor d" {
		t.Fatalf("password = %q, want %q", password, "p@ss/wor d")
	}
}
