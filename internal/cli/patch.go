package cli

import "github.com/spf13/cobra"

func newPatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch <path>",
		Short: "Send a PATCH request",
		Long:  "Send a PATCH request with an optional JSON body to the specified path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "PATCH", args)
		},
	}
	addCommonFlags(cmd)
	addBodyFlags(cmd)
	return cmd
}
