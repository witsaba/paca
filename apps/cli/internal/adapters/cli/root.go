package cli

import (
	"github.com/spf13/cobra"

	"github.com/witsaba/paca/packages/core"
)

const (
	rootName  = "paca"
	rootShort = core.ProductName + ", " + core.ProductDescription
	rootLong  = `paca is the command line interface of ` + core.ProductName + `, a personal finance manager
for the Colombian market.

This build ships the command header only: the command tree is not implemented
yet, so --help is the only supported command.`
)

// NewRootCommand builds the root command of the paca CLI.
//
// The root command owns the name and the description of the application. The
// commands that will drive PACA (accounts, movements, loans, reports) are
// attached to it as subcommands, each one in its own file with its own test.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   rootName,
		Short: rootShort,
		Long:  rootLong,
		// The header has no behaviour of its own: an unexpected argument is an
		// error, not a reason to print the help a second time.
		Args: cobra.NoArgs,
		// Runnable with no arguments means the command shows its own help
		// instead of failing. It is also what makes cobra validate the
		// arguments, so `paca accounts` is refused.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		// A failure must report the cause, not the usage of a command that
		// does not exist yet.
		SilenceUsage: true,
	}

	return cmd
}
