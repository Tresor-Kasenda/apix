package cli

import (
	"fmt"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/runner"
	"github.com/spf13/cobra"
)

func newChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chain <req1> <req2> [req3...]",
		Short: "Run saved requests in sequence",
		Long:  "Execute multiple saved requests sequentially, propagating captured variables between steps.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			varFlags, _ := cmd.Flags().GetStringSlice("var")
			flagVars := parseKeyValueSlice(varFlags, "=")
			envOverride, _ := cmd.Flags().GetString("env")
			baseOpts := ExecuteOptions{
				EnvOverride: envOverride,
			}
			if err := applyAdvancedNetworkFlags(cmd, &baseOpts); err != nil {
				return err
			}

			result, err := runner.RunChain(args, flagVars, envOverride, func(name string, vars map[string]string, env string) (*apixhttp.Response, error) {
				opts := ExecuteOptions{
					Vars:        vars,
					RequestName: name,
					EnvOverride: env,
					Retry:       baseOpts.Retry,
					RetryDelay:  baseOpts.RetryDelay,
					Proxy:       baseOpts.Proxy,
					Insecure:    baseOpts.Insecure,
					CertFile:    baseOpts.CertFile,
					KeyFile:     baseOpts.KeyFile,
					NoCookies:   baseOpts.NoCookies,
				}
				return executeSavedRequestWithResponse(name, opts)
			})
			if err != nil {
				if result != nil {
					output.PrintInfo(fmt.Sprintf("Chain stopped after %d/%d requests", result.Executed, result.Total))
				}
				return err
			}

			output.PrintSuccess(fmt.Sprintf("Chain completed: %d/%d requests succeeded (%d variables captured)", result.Executed, result.Total, result.Captured))
			return nil
		},
	}

	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	cmd.Flags().String("env", "", "Use a specific environment for this chain only")
	addAdvancedNetworkFlags(cmd)
	return cmd
}
