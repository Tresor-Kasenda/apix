package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a saved request",
		Long:  "Rename requests/<old>.yaml to requests/<new>.yaml.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			if err := request.Rename(oldName, newName); err != nil {
				return err
			}

			output.PrintSuccess(fmt.Sprintf("Renamed request %q to %q", oldName, newName))
			return nil
		},
	}
}
