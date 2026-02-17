package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	interopcurl "github.com/Tresor-Kasend/apix/internal/interop/curl"
	interopinsomnia "github.com/Tresor-Kasend/apix/internal/interop/insomnia"
	interoppostman "github.com/Tresor-Kasend/apix/internal/interop/postman"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

var nameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import requests from external formats",
		Long:  "Import request definitions from Postman, Insomnia, or a curl command.",
	}

	cmd.AddCommand(
		newImportPostmanCmd(),
		newImportInsomniaCmd(),
		newImportCurlCmd(),
	)
	return cmd
}

func newImportPostmanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "postman <collection.json>",
		Short: "Import requests from a Postman collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requests, err := interoppostman.ParseCollectionFile(args[0])
			if err != nil {
				return err
			}
			count, err := saveImportedRequests(requests)
			if err != nil {
				return err
			}
			output.PrintSuccess(fmt.Sprintf("Imported %d request(s) from Postman", count))
			return nil
		},
	}
}

func newImportInsomniaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "insomnia <export.json>",
		Short: "Import requests from an Insomnia export",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requests, err := interopinsomnia.ParseExportFile(args[0])
			if err != nil {
				return err
			}
			count, err := saveImportedRequests(requests)
			if err != nil {
				return err
			}
			output.PrintSuccess(fmt.Sprintf("Imported %d request(s) from Insomnia", count))
			return nil
		},
	}
}

func newImportCurlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "curl <command>",
		Short: "Import a curl command as a saved request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := interopcurl.ParseCommand(args[0])
			if err != nil {
				return err
			}

			overrideName, _ := cmd.Flags().GetString("name")
			if strings.TrimSpace(overrideName) != "" {
				req.Name = overrideName
			}

			count, err := saveImportedRequests([]request.SavedRequest{*req})
			if err != nil {
				return err
			}
			output.PrintSuccess(fmt.Sprintf("Imported %d curl request", count))
			return nil
		},
	}

	cmd.Flags().String("name", "", "Override the saved request name")
	return cmd
}

func saveImportedRequests(imported []request.SavedRequest) (int, error) {
	if len(imported) == 0 {
		return 0, fmt.Errorf("no importable requests found")
	}

	if err := os.MkdirAll("requests", 0o755); err != nil {
		return 0, fmt.Errorf("creating requests directory: %w", err)
	}

	used := make(map[string]int)
	savedCount := 0
	for i, reqDef := range imported {
		baseName := sanitizeRequestName(reqDef.Name)
		if baseName == "" {
			baseName = deriveRequestName(reqDef, i+1)
		}
		name := uniqueRequestName(baseName, used)

		reqDef.Name = name
		if err := request.Save(name, reqDef); err != nil {
			return savedCount, err
		}
		savedCount++
	}

	return savedCount, nil
}

func sanitizeRequestName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = nameSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-._")
	return strings.ToLower(value)
}

func deriveRequestName(reqDef request.SavedRequest, index int) string {
	method := strings.ToLower(strings.TrimSpace(reqDef.Method))
	if method == "" {
		method = "request"
	}

	pathValue := strings.TrimSpace(reqDef.Path)
	if pathValue == "" {
		return fmt.Sprintf("%s-%d", method, index)
	}
	pathValue = strings.Trim(pathValue, "/")
	if pathValue == "" {
		return method
	}
	pathValue = strings.ReplaceAll(pathValue, "://", "-")
	pathValue = strings.ReplaceAll(pathValue, "/", "-")
	pathValue = strings.ReplaceAll(pathValue, "?", "-")
	pathValue = strings.ReplaceAll(pathValue, "&", "-")
	pathValue = strings.ReplaceAll(pathValue, "=", "-")
	pathValue = strings.ReplaceAll(pathValue, "{", "")
	pathValue = strings.ReplaceAll(pathValue, "}", "")
	pathValue = strings.ReplaceAll(pathValue, "$", "")
	pathValue = sanitizeRequestName(filepath.Base(pathValue))
	if pathValue == "" {
		pathValue = fmt.Sprintf("request-%d", index)
	}

	return fmt.Sprintf("%s-%s", method, pathValue)
}

func uniqueRequestName(base string, used map[string]int) string {
	base = sanitizeRequestName(base)
	if base == "" {
		base = "request"
	}

	next := used[base]
	for {
		var candidate string
		if next == 0 {
			candidate = base
		} else {
			candidate = fmt.Sprintf("%s-%d", base, next+1)
		}

		if _, err := os.Stat(filepath.Join("requests", candidate+".yaml")); os.IsNotExist(err) {
			used[base] = next + 1
			return candidate
		}
		next++
	}
}
