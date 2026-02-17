package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/config"
	"github.com/Tresor-Kasend/apix/internal/env"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
		Long:  "List, show, create, copy, delete, and switch between environments.",
	}

	cmd.AddCommand(
		newEnvUseCmd(),
		newEnvListCmd(),
		newEnvShowCmd(),
		newEnvCreateCmd(),
		newEnvDeleteCmd(),
		newEnvCopyCmd(),
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
		Use:   "show [name]",
		Short: "Show environment configuration",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := resolveEnvNameForShow(args)
			if err != nil {
				return err
			}

			content, err := env.Show(name)
			if err != nil {
				return err
			}
			fmt.Println()
			fmt.Printf("# env/%s.yaml\n", name)
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

func newEnvDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			force, _ := cmd.Flags().GetBool("force")

			cfg, _ := config.Load()
			active := ""
			if cfg != nil {
				active = cfg.CurrentEnv
			}

			if !force {
				confirmed, err := confirmDeleteEnvironment(name)
				if err != nil {
					return err
				}
				if !confirmed {
					output.PrintInfo("Deletion canceled.")
					return nil
				}
			}

			if err := env.Delete(name, active); err != nil {
				return err
			}

			output.PrintSuccess(fmt.Sprintf("Deleted environment %q", name))
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Delete without confirmation")
	return cmd
}

func newEnvCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copy <source> <dest>",
		Short: "Copy an environment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			dest := args[1]

			if err := env.Copy(source, dest); err != nil {
				return err
			}

			output.PrintSuccess(fmt.Sprintf("Copied environment %q to %q", source, dest))
			return nil
		},
	}
}

func resolveEnvNameForShow(args []string) (string, error) {
	if len(args) == 1 {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return "", fmt.Errorf("environment name cannot be empty")
		}
		return name, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.CurrentEnv == "" {
		return "", fmt.Errorf("no active environment set (use 'apix env use <name>')")
	}
	return cfg.CurrentEnv, nil
}

func confirmDeleteEnvironment(name string) (bool, error) {
	fmt.Printf("  Delete environment %q? [y/N]: ", name)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
