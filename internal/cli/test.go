package cli

import (
	"fmt"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/Tresor-Kasend/apix/internal/tester"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [name]",
		Short: "Run request assertions",
		Long:  "Run tests from saved requests that define an expect block.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}

			dir, _ := cmd.Flags().GetString("dir")
			envOverride, _ := cmd.Flags().GetString("env")
			varFlags, _ := cmd.Flags().GetStringSlice("var")
			flagVars := parseKeyValueSlice(varFlags, "=")
			baseOpts := ExecuteOptions{
				EnvOverride: envOverride,
			}
			if err := applyAdvancedNetworkFlags(cmd, &baseOpts); err != nil {
				return err
			}

			suite, err := tester.Run(tester.RunnerOptions{
				Name:        name,
				Dir:         dir,
				Vars:        flagVars,
				EnvOverride: envOverride,
			}, func(requestName string, saved *request.SavedRequest, vars map[string]string, env string) (*apixhttp.Response, error) {
				opts := ExecuteOptions{
					Vars:           vars,
					EnvOverride:    env,
					RequestName:    requestName,
					Retry:          baseOpts.Retry,
					RetryDelay:     baseOpts.RetryDelay,
					Proxy:          baseOpts.Proxy,
					Insecure:       baseOpts.Insecure,
					CertFile:       baseOpts.CertFile,
					KeyFile:        baseOpts.KeyFile,
					NoCookies:      baseOpts.NoCookies,
					SkipSaveLast:   true,
					SuppressOutput: true,
					Silent:         true,
				}
				return executeSavedDefinitionWithResponse(requestName, saved, opts)
			})
			if err != nil {
				return err
			}

			if suite.Total == 0 {
				output.PrintInfo("No testable requests found (missing expect block).")
				return nil
			}

			for _, result := range suite.Results {
				output.PrintTestResult(result)
			}
			output.PrintTestSummary(*suite)

			if suite.ExitCode() != 0 {
				return fmt.Errorf("%d test(s) failed", suite.Failed)
			}
			return nil
		},
	}

	cmd.Flags().String("dir", "", "Directory containing request YAML files to test")
	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	cmd.Flags().String("env", "", "Use a specific environment for this test run only")
	addAdvancedNetworkFlags(cmd)
	return cmd
}
