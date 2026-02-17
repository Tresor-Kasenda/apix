package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/watch"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch <name>",
		Short: "Watch and re-run a saved request on file changes",
		Long:  "Watch requests/<name>.yaml and rerun it when the file changes, including pre_request/post_request hooks.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("request name is required")
			}

			watchPath := filepath.Join("requests", name+".yaml")
			if _, err := os.Stat(watchPath); err != nil {
				return fmt.Errorf("watch target %q not found: %w", watchPath, err)
			}

			interval, _ := cmd.Flags().GetDuration("interval")
			envOverride, _ := cmd.Flags().GetString("env")
			varFlags, _ := cmd.Flags().GetStringSlice("var")
			flagVars := parseKeyValueSlice(varFlags, "=")

			baseOpts := ExecuteOptions{EnvOverride: envOverride}
			if err := applyAdvancedNetworkFlags(cmd, &baseOpts); err != nil {
				return err
			}

			executor, err := watch.NewExecutor(func(requestName string, vars map[string]string, env string) (*apixhttp.Response, error) {
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
				return executeSavedRequestWithResponse(requestName, opts)
			}, watch.ExecutorOptions{})
			if err != nil {
				return err
			}

			watchMode := "fs events"
			if interval > 0 {
				watchMode = fmt.Sprintf("polling every %s", interval)
			}
			output.PrintInfo(fmt.Sprintf("Watching %s (%s). Press Ctrl+C to stop.", watchPath, watchMode))

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			iteration := 0
			err = watch.Run(ctx, watch.WatcherOptions{
				Path:     watchPath,
				Interval: interval,
			}, func(trigger watch.Trigger) {
				iteration++

				result, runErr := executor.Run(name, flagVars, envOverride)
				if runErr != nil {
					output.PrintInfo(fmt.Sprintf("[%s] #%d %s -> error: %v", trigger.Time.Format("15:04:05"), iteration, trigger.Reason, runErr))
					return
				}

				resp := result.Response
				statusText := resp.Status
				if parts := strings.SplitN(resp.Status, " ", 2); len(parts) == 2 {
					statusText = parts[1]
				}

				output.PrintInfo(fmt.Sprintf("[%s] #%d %s -> %d %s (%d req, %d vars)",
					trigger.Time.Format("15:04:05"),
					iteration,
					trigger.Reason,
					resp.StatusCode,
					statusText,
					result.Executed,
					result.Captured,
				))
			})
			if err != nil {
				return err
			}

			output.PrintInfo("Watch stopped.")
			return nil
		},
	}

	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	cmd.Flags().String("env", "", "Use a specific environment for this watch session")
	cmd.Flags().Duration("interval", 0, "Polling interval for file changes (e.g. 5s)")
	addAdvancedNetworkFlags(cmd)
	return cmd
}
