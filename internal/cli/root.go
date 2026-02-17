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
		Use:           "apix",
		Short:         "A modern CLI API tester",
		Long:          "apix is a modern, framework-agnostic API testing tool for terminal-first developers.",
		Version:       version,
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

type ExecuteOptions struct {
	Headers  map[string]string
	Query    map[string]string
	Vars     map[string]string
	Body     string
	BodyFile string
	Raw      bool
	Verbose  bool
}

func executeFromOptions(method, path string, opts ExecuteOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	url := buildURL(cfg.BaseURL, path)

	headers := make(map[string]string)
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	for k, v := range opts.Headers {
		headers[k] = v
	}

	if cfg.Auth.Type == "bearer" && cfg.Auth.Token != "" {
		format := cfg.Auth.HeaderFormat
		if format == "" {
			format = "Bearer ${TOKEN}"
		}
		headers["Authorization"] = strings.ReplaceAll(format, "${TOKEN}", cfg.Auth.Token)
	}

	vars := request.BuildVariableMap(cfg.Variables, cfg.Auth.Token, opts.Vars)

	url = request.ResolveVariables(url, vars)
	for k, v := range headers {
		headers[k] = request.ResolveVariables(v, vars)
	}

	query := make(map[string]string)
	for k, v := range opts.Query {
		query[k] = request.ResolveVariables(v, vars)
	}

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

	output.PrintStatus(method, path, resp.StatusCode, resp.Status, resp.Duration)
	if opts.Verbose {
		output.PrintHeaders(resp.Headers)
	}
	output.PrintBody(resp.Body, opts.Raw)

	if cfg.Auth.TokenPath != "" {
		if token, err := resp.ExtractField(cfg.Auth.TokenPath); err == nil && token != "" {
			if saveErr := config.SaveToken(token); saveErr == nil {
				output.PrintTokenCaptured()
			}
		}
	}

	_ = request.SaveLast(request.SavedRequest{
		Method:  method,
		Path:    path,
		Headers: opts.Headers,
		Query:   opts.Query,
		Body:    bodyStr,
	})

	return nil
}

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

	if cmd.Flags().Lookup("data") != nil {
		opts.Body, _ = cmd.Flags().GetString("data")
	}
	if cmd.Flags().Lookup("file") != nil {
		opts.BodyFile, _ = cmd.Flags().GetString("file")
	}

	return executeFromOptions(method, path, opts)
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceP("header", "H", nil, "Additional headers (key:value)")
	cmd.Flags().StringSliceP("query", "q", nil, "Query parameters (key=value)")
	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	cmd.Flags().Bool("raw", false, "Print raw response body without formatting")
	cmd.Flags().BoolP("verbose", "v", false, "Show response headers")
}

func addBodyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "Request body as JSON string")
	cmd.Flags().StringP("file", "f", "", "Read request body from file")
}

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
