package cli

import (
	"fmt"

	"github.com/Tresor-Kasend/apix/internal/config"
	"github.com/Tresor-Kasend/apix/internal/env"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
		Long:  "List, show, create, and switch between environments.",
	}

	cmd.AddCommand(
		newEnvUseCmd(),
		newEnvListCmd(),
		newEnvShowCmd(),
		newEnvCreateCmd(),
	)

	return cmd
}

func newEnvUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := env.SetActive(name); err != nil {
				return err
			}
			output.PrintSuccess(fmt.Sprintf("Switched to environment %q", name))
			return nil
		},
	}
}

func newEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available environments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := env.List()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				output.PrintInfo("No environments found. Run 'apix env create <name>' to create one.")
				return nil
			}

			// Get current active env.
			cfg, _ := config.Load()
			active := ""
			if cfg != nil {
				active = cfg.CurrentEnv
			}

			fmt.Println()
			for _, name := range names {
				if name == active {
					fmt.Printf("  * %s (active)\n", name)
				} else {
					fmt.Printf("    %s\n", name)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

func newEnvShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show environment configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := env.Show(args[0])
			if err != nil {
				return err
			}
			fmt.Println()
			fmt.Println(content)
			return nil
		},
	}
}

func newEnvCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := env.Create(name); err != nil {
				return err
			}
			output.PrintSuccess(fmt.Sprintf("Created environment %q (env/%s.yaml)", name, name))
			return nil
		},
	}
}
