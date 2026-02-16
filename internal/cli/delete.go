// delete.go defines the "apix delete" command.
package cli

import "github.com/spf13/cobra"

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <path>",
		Short: "Send a DELETE request",
		Long:  "Send a DELETE request to the specified path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "DELETE", args)
		},
	}
	addCommonFlags(cmd)
	return cmd
}
