// get.go defines the "apix get" command.
package cli

import "github.com/spf13/cobra"

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Send a GET request",
		Long:  "Send a GET request to the specified path, appended to the configured base URL.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "GET", args)
		},
	}
	addCommonFlags(cmd)
	return cmd
}
