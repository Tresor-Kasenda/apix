package cli

import "github.com/spf13/cobra"

func newOptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "options <path>",
		Short: "Send an OPTIONS request",
		Long:  "Send an OPTIONS request to inspect allowed methods and CORS behavior.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "OPTIONS", args)
		},
	}
	addCommonFlags(cmd)
	return cmd
}
