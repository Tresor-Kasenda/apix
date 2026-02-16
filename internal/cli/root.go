// Package cli defines all cobra commands for the apix CLI.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Tresor-Kasend/apix/internal/config"
	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/output"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/spf13/cobra"
)

var version = "dev"

// Execute runs the root command with the given version string.
func Execute(v string) error {
	version = v
	return rootCmd().Execute()
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apix",
		Short:   "A modern CLI API tester",
		Long:    "apix is a modern, framework-agnostic API testing tool for terminal-first developers.",
		Version: version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newGetCmd(),
		newPostCmd(),
		newPutCmd(),
		newPatchCmd(),
		newDeleteCmd(),
		newInitCmd(),
		newEnvCmd(),
		newSaveCmd(),
		newRunCmd(),
	)

	return cmd
}

// ExecuteOptions holds the parameters for executing an HTTP request.
type ExecuteOptions struct {
	Headers  map[string]string
	Query    map[string]string
	Vars     map[string]string
	Body     string
	BodyFile string
	Raw      bool
	Verbose  bool
}

// executeFromOptions is the shared core that all HTTP commands use.
func executeFromOptions(method, path string, opts ExecuteOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Build full URL.
	url := buildURL(cfg.BaseURL, path)

	// Merge headers: config defaults + request-specific.
	headers := make(map[string]string)
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	for k, v := range opts.Headers {
		headers[k] = v
	}

	// Add auth header.
	if cfg.Auth.Type == "bearer" && cfg.Auth.Token != "" {
		format := cfg.Auth.HeaderFormat
		if format == "" {
			format = "Bearer ${TOKEN}"
		}
		headers["Authorization"] = strings.ReplaceAll(format, "${TOKEN}", cfg.Auth.Token)
	}

	// Build variable map and resolve variables.
	vars := request.BuildVariableMap(cfg.Variables, cfg.Auth.Token, opts.Vars)

	url = request.ResolveVariables(url, vars)
	for k, v := range headers {
		headers[k] = request.ResolveVariables(v, vars)
	}

	// Resolve query params.
	query := make(map[string]string)
	for k, v := range opts.Query {
		query[k] = request.ResolveVariables(v, vars)
	}

	// Build body.
	var bodyReader io.Reader
	var bodyStr string
	if opts.BodyFile != "" {
		data, err := os.ReadFile(opts.BodyFile)
		if err != nil {
			return fmt.Errorf("reading body file: %w", err)
		}
		bodyStr = request.ResolveVariables(string(data), vars)
		bodyReader = strings.NewReader(bodyStr)
	} else if opts.Body != "" {
		bodyStr = request.ResolveVariables(opts.Body, vars)
		bodyReader = strings.NewReader(bodyStr)
	}

	// Send request.
	client := apixhttp.NewClient(time.Duration(cfg.Timeout) * time.Second)
	resp, err := client.Send(apixhttp.RequestOptions{
		Method:  method,
		URL:     url,
		Headers: headers,
		Query:   query,
		Body:    bodyReader,
	})
	if err != nil {
		return err
	}

	// Print response.
	output.PrintStatus(method, path, resp.StatusCode, resp.Status, resp.Duration)
	if opts.Verbose {
		output.PrintHeaders(resp.Headers)
	}
	output.PrintBody(resp.Body, opts.Raw)

	// Auto-capture token.
	if cfg.Auth.TokenPath != "" {
		if token, err := resp.ExtractField(cfg.Auth.TokenPath); err == nil && token != "" {
			if saveErr := config.SaveToken(token); saveErr == nil {
				output.PrintTokenCaptured()
			}
		}
	}

	// Save as last request for `apix save`.
	_ = request.SaveLast(request.SavedRequest{
		Method:  method,
		Path:    path,
		Headers: opts.Headers,
		Query:   opts.Query,
		Body:    bodyStr,
	})

	return nil
}

// executeRequest parses cobra flags and delegates to executeFromOptions.
func executeRequest(cmd *cobra.Command, method string, args []string) error {
	path := args[0]

	headerFlags, _ := cmd.Flags().GetStringSlice("header")
	queryFlags, _ := cmd.Flags().GetStringSlice("query")
	varFlags, _ := cmd.Flags().GetStringSlice("var")
	raw, _ := cmd.Flags().GetBool("raw")
	verbose, _ := cmd.Flags().GetBool("verbose")

	opts := ExecuteOptions{
		Headers: parseKeyValueSlice(headerFlags, ":"),
		Query:   parseKeyValueSlice(queryFlags, "="),
		Vars:    parseKeyValueSlice(varFlags, "="),
		Raw:     raw,
		Verbose: verbose,
	}

	// Body flags (only present on POST/PUT/PATCH).
	if cmd.Flags().Lookup("data") != nil {
		opts.Body, _ = cmd.Flags().GetString("data")
	}
	if cmd.Flags().Lookup("file") != nil {
		opts.BodyFile, _ = cmd.Flags().GetString("file")
	}

	return executeFromOptions(method, path, opts)
}

// addCommonFlags adds flags shared by all HTTP method commands.
func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceP("header", "H", nil, "Additional headers (key:value)")
	cmd.Flags().StringSliceP("query", "q", nil, "Query parameters (key=value)")
	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	cmd.Flags().Bool("raw", false, "Print raw response body without formatting")
	cmd.Flags().BoolP("verbose", "v", false, "Show response headers")
}

// addBodyFlags adds body-related flags for POST, PUT, and PATCH commands.
func addBodyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "Request body as JSON string")
	cmd.Flags().StringP("file", "f", "", "Read request body from file")
}

// buildURL joins the base URL and path, handling slash deduplication.
func buildURL(base, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// parseKeyValueSlice parses a slice of "key<sep>value" strings into a map.
func parseKeyValueSlice(items []string, sep string) map[string]string {
	result := make(map[string]string)
	for _, item := range items {
		parts := strings.SplitN(item, sep, 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
