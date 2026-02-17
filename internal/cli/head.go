package cli

import "github.com/spf13/cobra"

func newHeadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "head <path>",
		Short: "Send a HEAD request",
		Long:  "Send a HEAD request and print response headers.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "HEAD", args)
		},
	}
	addCommonFlags(cmd)
	return cmd
}
