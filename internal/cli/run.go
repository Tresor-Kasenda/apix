package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a saved request",
		Long:  "Load and execute a previously saved request from requests/<name>.yaml.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			timeoutSeconds, _ := cmd.Flags().GetInt("timeout")
			noFollow, _ := cmd.Flags().GetBool("no-follow")
			outputFile, _ := cmd.Flags().GetString("output")
			raw, _ := cmd.Flags().GetBool("raw")
			verbose, _ := cmd.Flags().GetBool("verbose")
			headersOnly, _ := cmd.Flags().GetBool("headers-only")
			bodyOnly, _ := cmd.Flags().GetBool("body-only")
			silent, _ := cmd.Flags().GetBool("silent")

			varFlags, _ := cmd.Flags().GetStringSlice("var")
			flagVars := parseKeyValueSlice(varFlags, "=")

			opts := ExecuteOptions{
				Vars:        flagVars,
				Raw:         raw,
				Verbose:     verbose,
				HeadersOnly: headersOnly,
				BodyOnly:    bodyOnly,
				Silent:      silent,
				OutputFile:  outputFile,
				NoFollow:    noFollow,
				Timeout:     time.Duration(timeoutSeconds) * time.Second,
			}

			return executeSavedRequest(name, opts)
		},
	}

	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	addExecutionFlags(cmd)

	return cmd
}
