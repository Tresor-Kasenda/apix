package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved requests",
		Long:  "List all requests saved under requests/*.yaml.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := request.ListSaved()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				output.PrintInfo("No saved requests found. Use 'apix save <name>' after running a request.")
				return nil
			}

			fmt.Println()
			for _, name := range names {
				fmt.Printf("  %s\n", name)
			}
			fmt.Println()
			return nil
		},
	}
}
