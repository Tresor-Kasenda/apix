package cli

import (
	"encoding/json"
	"fmt"

	cfgpkg "github.com/Tresor-Kasend/apix/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect effective configuration",
		Long:  "Commands related to the merged runtime configuration.",
	}

	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show merged active configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cfgpkg.Load()
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("encoding config: %w", err)
			}

			fmt.Println()
			fmt.Println(string(data))
			return nil
		},
	}
}
