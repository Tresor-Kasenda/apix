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
	interopopenapi "github.com/Tresor-Kasend/apix/internal/interop/openapi"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/project"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var allowedAuthTypes = map[string]bool{
	"none":    true,
	"bearer":  true,
	"basic":   true,
	"api_key": true,
	"custom":  true,
}

type initOptions struct {
	name     string
	baseURL  string
	auth     string
	spec     string
	yes      bool
	noDetect bool
}

func newInitCmd() *cobra.Command {
	opts := initOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new apix project",
		Long: `Create apix.yaml, environment files, and request directories in the current folder.

apix inspects the project to pre-fill sensible defaults:
  - the backend framework (also inside monorepo folders such as backend/ or api/)
  - the base URL, from .env (API_URL, APP_URL, PORT, ...) or framework defaults
  - routes, from an OpenAPI/Swagger document when one exists, otherwise from source code

Prompts are skipped with --yes or when stdin is not a terminal (CI, scripts).`,
		Example: `  apix init
  apix init --yes
  apix init --name shop --base-url http://localhost:3000 --auth none -y
  apix init --spec http://localhost:8000/openapi.json -y`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "Project name (default: current directory name)")
	cmd.Flags().StringVar(&opts.baseURL, "base-url", "", "Base URL for requests (default: detected)")
	cmd.Flags().StringVar(&opts.auth, "auth", "", "Auth type: none, bearer, basic, api_key, custom (default: bearer)")
	cmd.Flags().StringVar(&opts.spec, "spec", "", "OpenAPI/Swagger file or URL to import requests from")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Accept detected defaults without prompting")
	cmd.Flags().BoolVar(&opts.noDetect, "no-detect", false, "Skip framework, route and spec detection")
	return cmd
}

func runInit(opts initOptions) error {
	if _, err := os.Stat(project.ConfigFileName); err == nil {
		return fmt.Errorf("apix.yaml already exists in this directory")
	}
	if opts.auth != "" && !allowedAuthTypes[strings.ToLower(opts.auth)] {
		return fmt.Errorf("invalid auth type %q (allowed: none, bearer, basic, api_key, custom)", opts.auth)
	}
	if parent := project.Root(); parent != "." {
		output.PrintInfo(fmt.Sprintf("Creating a nested project (parent project at %s)", parent))
	}

	result := detect.Result{BaseURL: "http://localhost:8000"}
	if !opts.noDetect {
		result = detect.Detect(".")
		reportDetection(result)
	}

	interactive := !opts.yes && stdinIsTerminal()
	ask := func(label, flagValue, defaultValue string) string {
		if flagValue != "" {
			return flagValue
		}
		if !interactive {
			return defaultValue
		}
		return promptInput(label, defaultValue)
	}

	projectName := ask("Project name", opts.name, detectProjectName())
	baseURL := ask("Base URL", opts.baseURL, result.BaseURL)
	authType := strings.ToLower(opts.auth)
	if authType == "" {
		authType = "bearer"
		if interactive {
			authType = promptAuthType("Auth type", "bearer")
		}
	}

	// Write the config first: from then on, project paths resolve to this folder.
	if err := config.WriteDefault(projectName, baseURL, authType); err != nil {
		return err
	}
	output.PrintSuccess("Created apix.yaml")

	for _, d := range []string{project.RequestsDir(), project.EnvDir(), project.StateDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %q: %w", d, err)
		}
	}
	if err := env.EnsureGitignoreEntry(".apix/"); err != nil {
		return err
	}

	if err := env.Create("dev"); err != nil {
		return err
	}
	output.PrintSuccess("Created env/dev.yaml")

	specSource := opts.spec
	if specSource == "" {
		specSource = result.SpecPath
	}
	switch {
	case specSource != "":
		if err := importSpecOnInit(specSource, baseURL); err != nil {
			output.PrintInfo(fmt.Sprintf("Could not import OpenAPI spec %s: %v", specSource, err))
			saveDetectedRoutes(result.Routes)
		}
	default:
		saveDetectedRoutes(result.Routes)
	}

	output.PrintSuccess("Project initialized! Try 'apix list' or 'apix get /'.")
	return nil
}

func reportDetection(result detect.Result) {
	if result.Framework != nil {
		location := ""
		if result.Dir != "" {
			location = " in " + result.Dir + "/"
		}
		output.PrintInfo(fmt.Sprintf("Detected: %s (%s)%s", result.Framework.Name, result.Framework.Language, location))
	} else {
		output.PrintInfo("No known framework detected — using generic defaults")
	}
	if result.SpecPath != "" {
		output.PrintInfo("Found OpenAPI spec: " + result.SpecPath)
	}
}

// importSpecOnInit imports every operation of an OpenAPI document. Paths are
// prefixed with the spec base path (e.g. /api/v1) unless base_url already
// ends with it.
func importSpecOnInit(source, baseURL string) error {
	probe, err := interopopenapi.ParseSource(source, interopopenapi.Options{})
	if err != nil {
		return err
	}
	res := probe
	if probe.BasePath != "" && !strings.HasSuffix(strings.TrimRight(baseURL, "/"), probe.BasePath) {
		if res, err = interopopenapi.ParseSource(source, interopopenapi.Options{IncludeBasePath: true}); err != nil {
			return err
		}
	}
	count, err := saveImportedRequests(res.Requests)
	if err != nil {
		return err
	}
	output.PrintSuccess(fmt.Sprintf("Imported %d request(s) from OpenAPI spec", count))
	return nil
}

func saveDetectedRoutes(routes []detect.DetectedRoute) {
	if len(routes) == 0 {
		return
	}
	saved := 0
	for _, route := range routes {
		name := detect.SanitizeName(route.Method, route.Path)
		req := request.SavedRequest{
			Method: route.Method,
			Path:   detect.NormalizePathParams(route.Path),
		}
		if err := request.Save(name, req); err != nil {
			output.PrintError(fmt.Errorf("saving route %s: %w", name, err))
			continue
		}
		saved++
	}
	output.PrintSuccess(fmt.Sprintf("Found %d routes, saved to %s/", saved, filepath.Base(project.RequestsDir())))
}

func stdinIsTerminal() bool {
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
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
