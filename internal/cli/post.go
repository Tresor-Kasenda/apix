// post.go defines the "apix post" command.
package cli

import "github.com/spf13/cobra"

func newPostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post <path>",
		Short: "Send a POST request",
		Long:  "Send a POST request with an optional JSON body to the specified path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeRequest(cmd, "POST", args)
		},
	}
	addCommonFlags(cmd)
	addBodyFlags(cmd)
	return cmd
}
