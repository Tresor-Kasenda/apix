package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	apixauth "github.com/Tresor-Kasend/apix/internal/auth"
	"github.com/Tresor-Kasend/apix/internal/config"
	"github.com/Tresor-Kasend/apix/internal/history"
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
		newHeadCmd(),
		newOptionsCmd(),
		newInitCmd(),
		newEnvCmd(),
		newHistoryCmd(),
		newConfigCmd(),
		newSaveCmd(),
		newRunCmd(),
		newChainCmd(),
		newTestCmd(),
		newListCmd(),
		newShowCmd(),
		newRenameCmd(),
	)

	return cmd
}

type ExecuteOptions struct {
	Headers    map[string]string
	Query      map[string]string
	Vars       map[string]string
	Body       string
	BodyFile   string
	Form       []apixhttp.FormField
	URLEncoded []apixhttp.FormField

	Raw         bool
	Verbose     bool
	HeadersOnly bool
	BodyOnly    bool
	Silent      bool
	OutputFile  string

	Timeout     time.Duration
	NoFollow    bool
	EnvOverride string

	RequestName     string
	SkipAutoRefresh bool
	SkipSaveLast    bool
	FailOnHTTPError bool
	SuppressOutput  bool
}

func executeFromOptions(method, path string, opts ExecuteOptions) error {
	_, err := executeFromOptionsInternal(method, path, opts, false)
	return err
}

func executeFromOptionsInternal(method, path string, opts ExecuteOptions, alreadyRetried bool) (*apixhttp.Response, error) {
	if err := validateDisplayModes(opts); err != nil {
		return nil, err
	}
	if err := validateBodyModes(opts); err != nil {
		return nil, err
	}

	cfg, err := config.LoadWithEnvOverride(opts.EnvOverride)
	if err != nil {
		return nil, err
	}

	urlStr := buildURL(cfg.BaseURL, path)

	headers := make(map[string]string)
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	for k, v := range opts.Headers {
		headers[k] = v
	}

	vars := request.BuildVariableMap(cfg.Variables, cfg.Auth.Token, opts.Vars)

	urlStr = request.ResolveVariables(urlStr, vars)
	for k, v := range headers {
		headers[k] = request.ResolveVariables(v, vars)
	}
	if err := apixauth.Apply(headers, cfg, vars); err != nil {
		return nil, err
	}

	query := make(map[string]string)
	for k, v := range opts.Query {
		query[k] = request.ResolveVariables(v, vars)
	}

	bodyReader, bodyStr, contentType, err := buildRequestBody(opts, vars)
	if err != nil {
		return nil, err
	}
	if contentType != "" && !hasHeader(opts.Headers, "Content-Type") {
		headers["Content-Type"] = contentType
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	client := apixhttp.NewClientWithConfig(apixhttp.ClientConfig{
		Timeout:         timeout,
		FollowRedirects: !opts.NoFollow,
	})

	requestStart := time.Now()
	resp, err := client.Send(apixhttp.RequestOptions{
		Method:  method,
		URL:     urlStr,
		Headers: headers,
		Query:   query,
		Body:    bodyReader,
	})
	if err != nil {
		_ = history.Append(history.Entry{
			Method:     strings.ToUpper(method),
			Path:       urlStr,
			Status:     0,
			DurationMS: time.Since(requestStart).Milliseconds(),
		})
		return nil, err
	}
	_ = history.Append(history.Entry{
		Method:       strings.ToUpper(method),
		Path:         urlStr,
		Status:       resp.StatusCode,
		DurationMS:   resp.Duration.Milliseconds(),
		ResponseSize: len(resp.Body),
	})

	shouldRetry, refreshErr := apixauth.RefreshIfNeeded(
		cfg,
		opts.RequestName,
		resp.StatusCode,
		alreadyRetried,
		opts.SkipAutoRefresh,
		func(loginRequest string) error {
			return executeSavedRequest(loginRequest, ExecuteOptions{
				Vars:            opts.Vars,
				Timeout:         opts.Timeout,
				NoFollow:        opts.NoFollow,
				EnvOverride:     opts.EnvOverride,
				RequestName:     loginRequest,
				SkipAutoRefresh: true,
				SkipSaveLast:    true,
				Silent:          true,
				FailOnHTTPError: true,
				SuppressOutput:  true,
			})
		},
	)
	if refreshErr != nil {
		return nil, refreshErr
	}
	if shouldRetry {
		if !opts.Silent && !opts.BodyOnly && !opts.HeadersOnly {
			output.PrintSuccess("Token expired, re-authenticated automatically")
		}
		return executeFromOptionsInternal(method, path, opts, true)
	}
	if alreadyRetried && resp.StatusCode == 401 && cfg.Auth.LoginRequest != "" && !opts.SkipAutoRefresh {
		return nil, fmt.Errorf("request is still unauthorized after automatic re-authentication")
	}
	if opts.FailOnHTTPError && resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d %s", resp.StatusCode, resp.Status)
	}

	if err := writeOutputFile(opts.OutputFile, resp.Body); err != nil {
		return nil, err
	}

	shouldPrintStatus := !opts.SuppressOutput && !opts.Silent && !opts.BodyOnly
	shouldPrintHeaders := !opts.SuppressOutput && !opts.Silent && (opts.HeadersOnly || opts.Verbose || strings.EqualFold(method, "HEAD"))
	shouldPrintBody := !opts.SuppressOutput && !opts.HeadersOnly && !strings.EqualFold(method, "HEAD") && opts.OutputFile == ""

	if shouldPrintStatus {
		output.PrintStatus(method, path, resp.StatusCode, resp.Status, resp.Duration, len(resp.Body))
	}
	if shouldPrintHeaders {
		output.PrintHeaders(resp.Headers)
	}
	if shouldPrintBody {
		if opts.Silent || opts.BodyOnly {
			output.PrintBodyRaw(resp.Body)
		} else {
			output.PrintBody(resp.Body, opts.Raw)
		}
	}

	if cfg.Auth.TokenPath != "" {
		if token, tokenErr := resp.ExtractField(cfg.Auth.TokenPath); tokenErr == nil && token != "" {
			if saveErr := config.SaveToken(token); saveErr == nil && !opts.Silent && !opts.BodyOnly && !opts.HeadersOnly {
				output.PrintTokenCaptured()
			}
		}
	}

	if !opts.SkipSaveLast {
		_ = request.SaveLast(request.SavedRequest{
			Method:  method,
			Path:    path,
			Headers: opts.Headers,
			Query:   opts.Query,
			Body:    bodyStr,
		})
	}

	return resp, nil
}

func executeSavedRequest(name string, baseOpts ExecuteOptions) error {
	_, err := executeSavedRequestWithResponse(name, baseOpts)
	return err
}

func executeSavedRequestWithResponse(name string, baseOpts ExecuteOptions) (*apixhttp.Response, error) {
	saved, err := request.Load(name)
	if err != nil {
		return nil, fmt.Errorf("loading saved request %q: %w", name, err)
	}

	return executeSavedDefinitionWithResponse(name, saved, baseOpts)
}

func executeSavedDefinitionWithResponse(name string, saved *request.SavedRequest, baseOpts ExecuteOptions) (*apixhttp.Response, error) {
	if saved == nil {
		return nil, fmt.Errorf("saved request %q is nil", name)
	}

	opts := baseOpts
	opts.Headers = saved.Headers
	opts.Query = saved.Query
	opts.Body = saved.Body
	opts.BodyFile = ""
	opts.Form = nil
	opts.URLEncoded = nil
	opts.RequestName = name

	if strings.EqualFold(saved.Method, "HEAD") && !opts.BodyOnly && !opts.Silent {
		opts.HeadersOnly = true
	}

	resp, err := executeFromOptionsWithResponse(saved.Method, saved.Path, opts)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func executeFromOptionsWithResponse(method, path string, opts ExecuteOptions) (*apixhttp.Response, error) {
	return executeFromOptionsInternal(method, path, opts, false)
}

func executeRequest(cmd *cobra.Command, method string, args []string) error {
	path := args[0]

	headerFlags, _ := cmd.Flags().GetStringSlice("header")
	queryFlags, _ := cmd.Flags().GetStringSlice("query")
	varFlags, _ := cmd.Flags().GetStringSlice("var")
	formFlags, _ := cmd.Flags().GetStringSlice("form")
	urlencodedFlags, _ := cmd.Flags().GetStringSlice("urlencoded")

	formFields, err := parseFieldSlice(formFlags)
	if err != nil {
		return err
	}
	urlencodedFields, err := parseFieldSlice(urlencodedFlags)
	if err != nil {
		return err
	}

	timeoutSeconds, _ := cmd.Flags().GetInt("timeout")
	noFollow, _ := cmd.Flags().GetBool("no-follow")
	outputFile, _ := cmd.Flags().GetString("output")
	raw, _ := cmd.Flags().GetBool("raw")
	verbose, _ := cmd.Flags().GetBool("verbose")
	headersOnly, _ := cmd.Flags().GetBool("headers-only")
	bodyOnly, _ := cmd.Flags().GetBool("body-only")
	silent, _ := cmd.Flags().GetBool("silent")

	opts := ExecuteOptions{
		Headers:     parseKeyValueSlice(headerFlags, ":"),
		Query:       parseQueryFlags(queryFlags),
		Vars:        parseKeyValueSlice(varFlags, "="),
		Form:        formFields,
		URLEncoded:  urlencodedFields,
		Raw:         raw,
		Verbose:     verbose,
		HeadersOnly: headersOnly,
		BodyOnly:    bodyOnly,
		Silent:      silent,
		OutputFile:  outputFile,
		NoFollow:    noFollow,
		Timeout:     time.Duration(timeoutSeconds) * time.Second,
	}

	if flag := cmd.Flags().Lookup("data"); flag != nil && flag.Changed {
		opts.Body, _ = cmd.Flags().GetString("data")
	}
	if flag := cmd.Flags().Lookup("file"); flag != nil && flag.Changed {
		opts.BodyFile, _ = cmd.Flags().GetString("file")
	}

	if strings.EqualFold(method, "HEAD") && !opts.BodyOnly && !opts.Silent {
		opts.HeadersOnly = true
	}

	return executeFromOptions(method, path, opts)
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceP("header", "H", nil, "Additional headers (key:value)")
	cmd.Flags().StringSliceP("query", "q", nil, "Query parameters (key=value or key1=v1&key2=v2)")
	cmd.Flags().StringSliceP("var", "V", nil, "Variables (key=value)")
	addExecutionFlags(cmd)
}

func addExecutionFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("raw", false, "Print raw response body without formatting")
	cmd.Flags().BoolP("verbose", "v", false, "Show response headers")
	cmd.Flags().Bool("headers-only", false, "Print status and response headers only")
	cmd.Flags().Bool("body-only", false, "Print response body only")
	cmd.Flags().BoolP("silent", "s", false, "Print only the response body")
	cmd.Flags().StringP("output", "o", "", "Write response body to a file")
	cmd.Flags().IntP("timeout", "t", 0, "Request timeout in seconds (overrides config)")
	cmd.Flags().Bool("no-follow", false, "Do not follow redirects")
}

func addBodyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "Request body as JSON string")
	cmd.Flags().StringP("file", "f", "", "Read request body from file")
	cmd.Flags().StringSlice("form", nil, "Multipart form field (key=value or key=@file)")
	cmd.Flags().StringSlice("urlencoded", nil, "URL-encoded field (key=value)")
}

func buildRequestBody(opts ExecuteOptions, vars map[string]string) (io.Reader, string, string, error) {
	modeCount := 0
	if opts.Body != "" {
		modeCount++
	}
	if opts.BodyFile != "" {
		modeCount++
	}
	if len(opts.Form) > 0 {
		modeCount++
	}
	if len(opts.URLEncoded) > 0 {
		modeCount++
	}
	if modeCount > 1 {
		return nil, "", "", fmt.Errorf("--data, --file, --form, and --urlencoded are mutually exclusive")
	}

	if opts.BodyFile != "" {
		data, err := os.ReadFile(opts.BodyFile)
		if err != nil {
			return nil, "", "", fmt.Errorf("reading body file: %w", err)
		}
		bodyStr := request.ResolveVariables(string(data), vars)
		return strings.NewReader(bodyStr), bodyStr, "", nil
	}

	if opts.Body != "" {
		bodyStr := request.ResolveVariables(opts.Body, vars)
		return strings.NewReader(bodyStr), bodyStr, "", nil
	}

	if len(opts.Form) > 0 {
		resolved := resolveFormFields(opts.Form, vars)
		bodyBytes, contentType, err := apixhttp.BuildMultipartForm(resolved)
		if err != nil {
			return nil, "", "", err
		}
		return bytes.NewReader(bodyBytes), summarizeFormFields(resolved), contentType, nil
	}

	if len(opts.URLEncoded) > 0 {
		resolved := resolveFormFields(opts.URLEncoded, vars)
		encoded, contentType, err := apixhttp.BuildURLEncodedForm(resolved)
		if err != nil {
			return nil, "", "", err
		}
		return strings.NewReader(encoded), encoded, contentType, nil
	}

	return nil, "", "", nil
}

func resolveFormFields(fields []apixhttp.FormField, vars map[string]string) []apixhttp.FormField {
	resolved := make([]apixhttp.FormField, 0, len(fields))
	for _, f := range fields {
		resolved = append(resolved, apixhttp.FormField{
			Key:   request.ResolveVariables(f.Key, vars),
			Value: request.ResolveVariables(f.Value, vars),
		})
	}
	return resolved
}

func summarizeFormFields(fields []apixhttp.FormField) string {
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s=%s", f.Key, f.Value))
	}
	return strings.Join(parts, "&")
}

func validateDisplayModes(opts ExecuteOptions) error {
	if opts.HeadersOnly && opts.BodyOnly {
		return fmt.Errorf("--headers-only and --body-only cannot be used together")
	}
	return nil
}

func validateBodyModes(opts ExecuteOptions) error {
	modeCount := 0
	if opts.Body != "" {
		modeCount++
	}
	if opts.BodyFile != "" {
		modeCount++
	}
	if len(opts.Form) > 0 {
		modeCount++
	}
	if len(opts.URLEncoded) > 0 {
		modeCount++
	}
	if modeCount > 1 {
		return fmt.Errorf("--data, --file, --form, and --urlencoded are mutually exclusive")
	}
	return nil
}

func parseFieldSlice(items []string) ([]apixhttp.FormField, error) {
	fields := make([]apixhttp.FormField, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field %q (expected key=value)", item)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid field %q (empty key)", item)
		}
		fields = append(fields, apixhttp.FormField{Key: key, Value: value})
	}
	return fields, nil
}

func writeOutputFile(path string, body []byte) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating output directory %q: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("writing output file %q: %w", path, err)
	}
	return nil
}

func hasHeader(headers map[string]string, key string) bool {
	for k := range headers {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
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

func parseQueryFlags(items []string) map[string]string {
	result := make(map[string]string)
	for _, item := range items {
		if strings.Contains(item, "&") {
			values, err := url.ParseQuery(item)
			if err == nil {
				for k, vals := range values {
					if len(vals) == 0 {
						result[k] = ""
						continue
					}
					result[k] = vals[len(vals)-1]
				}
				continue
			}
		}

		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
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
