// Command paca is the command line interface of PACA.
//
// This file is the composition root: it builds the command tree and runs it.
// It holds no business rule and no I/O of its own.
package main

import (
	"io"
	"os"

	"github.com/witsaba/paca/apps/cli/internal/adapters/cli"
)

func main() {
	os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr))
}

// execute is the body of the process: it builds the command tree and runs
// it with the arguments it is given.
//
// The status of the process is the return value: 0 when the command
// succeeded, and 1 when it failed. The body of the process lives here and
// not in main, because main cannot be called from a test and the root of
// the CLI has to be covered like any other file.
func execute(args []string, stdout, stderr io.Writer) int {
	cmd := cli.NewRootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		return 1
	}

	return 0
}
