package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/config"
	interopcurl "github.com/Tresor-Kasend/apix/internal/interop/curl"
	interoppostman "github.com/Tresor-Kasend/apix/internal/interop/postman"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export requests to external formats",
		Long:  "Export saved requests as curl commands or a Postman collection.",
	}

	cmd.AddCommand(
		newExportCurlCmd(),
		newExportPostmanCmd(),
	)
	return cmd
}

func newExportCurlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "curl <name>",
		Short: "Export one saved request as a curl command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			saved, err := request.Load(name)
			if err != nil {
				return err
			}

			command, err := interopcurl.ToCommand(*saved)
			if err != nil {
				return err
			}

			fmt.Println(command)
			return nil
		},
	}
}

func newExportPostmanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postman",
		Short: "Export all saved requests as a Postman collection",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := request.ListSaved()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				return fmt.Errorf("no saved requests found")
			}

			collectionRequests := make([]request.SavedRequest, 0, len(names))
			for _, name := range names {
				saved, err := request.Load(name)
				if err != nil {
					return err
				}
				if strings.TrimSpace(saved.Name) == "" {
					saved.Name = name
				}
				collectionRequests = append(collectionRequests, *saved)
			}

			collectionName, _ := cmd.Flags().GetString("name")
			if strings.TrimSpace(collectionName) == "" {
				cfg, err := config.Load()
				if err == nil && strings.TrimSpace(cfg.Project) != "" {
					collectionName = cfg.Project
				} else {
					collectionName = "apix export"
				}
			}

			data, err := interoppostman.ExportCollection(collectionRequests, collectionName)
			if err != nil {
				return err
			}

			outputPath, _ := cmd.Flags().GetString("output")
			if strings.TrimSpace(outputPath) == "" {
				fmt.Println(string(data))
				return nil
			}

			if err := os.WriteFile(outputPath, data, 0o644); err != nil {
				return fmt.Errorf("writing postman export %q: %w", outputPath, err)
			}
			output.PrintSuccess(fmt.Sprintf("Postman collection exported to %s", outputPath))
			return nil
		},
	}

	cmd.Flags().String("name", "", "Collection name override")
	cmd.Flags().String("output", "", "Write JSON to file instead of stdout")
	return cmd
}
