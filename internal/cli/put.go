// put.go defines the "apix put" command.
package cli

import "github.com/spf13/cobra"

func newPutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put <path>",
		Short: "Send a PUT request",
		Long:  "Send a PUT request with an optional JSON body to the specified path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "PUT", args)
		},
	}
	addCommonFlags(cmd)
	addBodyFlags(cmd)
	return cmd
}
