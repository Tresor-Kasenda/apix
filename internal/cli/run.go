package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/request"
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

			saved, err := request.Load(name)
			if err != nil {
				return fmt.Errorf("loading saved request %q: %w", name, err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			raw, _ := cmd.Flags().GetBool("raw")

			varFlags, _ := cmd.Flags().GetStringSlice("var")
			flagVars := parseKeyValueSlice(varFlags, "=")

			opts := ExecuteOptions{
				Headers: saved.Headers,
				Query:   saved.Query,
				Body:    saved.Body,
				Vars:    flagVars,
				Raw:     raw,
				Verbose: verbose,
			}

			return executeFromOptions(saved.Method, saved.Path, opts)
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show response headers")
	cmd.Flags().Bool("raw", false, "Print raw response body without formatting")
	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")

	return cmd
}
