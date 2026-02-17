package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/config"
	"github.com/Tresor-Kasend/apix/internal/detect"
	"github.com/Tresor-Kasend/apix/internal/env"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

var allowedAuthTypes = map[string]bool{
	"none":    true,
	"bearer":  true,
	"basic":   true,
	"api_key": true,
	"custom":  true,
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new apix project",
		Long:  "Create apix.yaml, environment files, and request directories in the current folder.",
		Args:  cobra.NoArgs,
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	if config.Exists() {
		return fmt.Errorf("apix.yaml already exists in this directory")
	}

	// Detect framework and routes.
	result := detect.Detect(".")

	// Propose base URL based on detected framework.
	defaultURL := "http://localhost:8000/api"
	if result.Framework != nil {
		output.PrintInfo(fmt.Sprintf("Detected: %s (%s)", result.Framework.Name, result.Framework.Language))
		defaultURL = fmt.Sprintf("http://localhost:%d/api", result.Framework.DefaultPort)
	}

	defaultProject := detectProjectName()
	projectName := promptInput("Project name", defaultProject)
	baseURL := promptInput("Base URL", defaultURL)
	authType := promptAuthType("Auth type", "bearer")

	// Create directories.
	dirs := []string{"requests", "env", ".apix"}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %q: %w", d, err)
		}
	}

	if err := env.EnsureGitignoreEntry(".apix/"); err != nil {
		return err
	}

	// Write apix.yaml.
	if err := config.WriteDefault(projectName, baseURL, authType); err != nil {
		return err
	}
	output.PrintSuccess("Created apix.yaml")

	// Create default dev environment.
	if err := env.Create("dev"); err != nil {
		return err
	}
	output.PrintSuccess("Created env/dev.yaml")

	// Save detected routes as request files.
	if len(result.Routes) > 0 {
		for _, route := range result.Routes {
			name := detect.SanitizeName(route.Method, route.Path)
			req := request.SavedRequest{
				Method: route.Method,
				Path:   route.Path,
			}
			if err := request.Save(name, req); err != nil {
				output.PrintError(fmt.Errorf("saving route %s: %w", name, err))
				continue
			}
		}
		output.PrintSuccess(fmt.Sprintf("Found %d routes, saved to requests/", len(result.Routes)))
	}

	output.PrintSuccess("Project initialized! Run 'apix get /health' to test.")
	return nil
}

func detectProjectName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "my-api"
	}
	name := strings.TrimSpace(filepath.Base(wd))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "my-api"
	}
	return name
}

func promptInput(label, defaultVal string) string {
	fmt.Printf("  %s [%s]: ", label, defaultVal)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptAuthType(label, defaultVal string) string {
	for {
		value := strings.ToLower(promptInput(label+" (none|bearer|basic|api_key|custom)", defaultVal))
		if allowedAuthTypes[value] {
			return value
		}
		output.PrintInfo("Invalid auth type. Allowed: none, bearer, basic, api_key, custom")
	}
}
