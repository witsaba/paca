// Command paca is the command line interface of PACA.
//
// This file is the composition root: it builds the command tree and runs it.
// It holds no business rule and no I/O of its own.
package main

import (
	"os"

	"github.com/witsaba/paca/apps/cli/internal/adapters/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
