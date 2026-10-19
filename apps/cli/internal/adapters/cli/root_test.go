package cli_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/witsaba/paca/apps/cli/internal/adapters/cli"
)

// The paca command is a header: --help is the only supported command.
func TestRootCommandHelp(t *testing.T) {
	var out bytes.Buffer

	cmd := cli.NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("paca --help returned an error, want exit code 0: %v", err)
	}

	help := out.String()
	for _, want := range []string{
		"paca",
		"Usage:",
		"personal finance manager",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("paca --help output does not contain %q\noutput:\n%s", want, help)
		}
	}
}

// Running paca with no arguments shows the help instead of failing.
func TestRootCommandWithoutArgumentsShowsHelp(t *testing.T) {
	var out bytes.Buffer

	cmd := cli.NewRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("paca with no arguments returned an error, want exit code 0: %v", err)
	}

	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("paca with no arguments did not print the help\noutput:\n%s", out.String())
	}
}

// The header exposes no subcommand: everything but --help is rejected.
func TestRootCommandRejectsAnyOtherCommand(t *testing.T) {
	cmd := cli.NewRootCommand()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"accounts"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("paca accounts succeeded, want an error: the header has no subcommands")
	}
}
