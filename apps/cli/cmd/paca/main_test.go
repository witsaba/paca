package main

import (
	"bytes"
	"strings"
	"testing"
)

// The command tree runs and the process ends with the status of the
// command.
func TestExecuteShowsTheHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	status := execute([]string{"--help"}, &out, &errOut)

	if status != 0 {
		t.Errorf("the status is %d, want 0", status)
	}

	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("the output does not carry the help\noutput:\n%s", out.String())
	}
}

// A command that does not exist is a failure of the process.
func TestExecuteFailsOnAnUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	status := execute([]string{"accounts"}, &out, &errOut)

	if status != 1 {
		t.Errorf("the status is %d, want 1", status)
	}
}
