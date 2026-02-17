package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <path>",
		Short: "Send a DELETE request or delete a saved request",
		Long:  "By default sends an HTTP DELETE request. Use --saved to delete requests/<name>.yaml.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deleteSaved, _ := cmd.Flags().GetBool("saved")
			if deleteSaved {
				name := args[0]
				if err := request.Delete(name); err != nil {
					return err
				}
				output.PrintSuccess(fmt.Sprintf("Deleted saved request %q", name))
				return nil
			}
			return executeRequest(cmd, "DELETE", args)
		},
	}
	addCommonFlags(cmd)
	cmd.Flags().Bool("saved", false, "Delete a saved request by name instead of sending an HTTP DELETE")
	return cmd
}
