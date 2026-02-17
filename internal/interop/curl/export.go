package curl

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/request"
)

func ToCommand(req request.SavedRequest) (string, error) {
	method := strings.TrimSpace(strings.ToUpper(req.Method))
	if method == "" {
		method = "GET"
	}

	targetURL, err := buildURL(req.Path, req.Query)
	if err != nil {
		return "", err
	}

	parts := []string{
		"curl",
		"-X",
		quoteArg(method),
		quoteArg(targetURL),
	}

	if len(req.Headers) > 0 {
		keys := make([]string, 0, len(req.Headers))
		for key := range req.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.TrimSpace(req.Headers[key])
			parts = append(parts, "-H", quoteArg(fmt.Sprintf("%s: %s", key, value)))
		}
	}

	if req.Body != "" {
		parts = append(parts, "--data-raw", quoteArg(req.Body))
	}

	return strings.Join(parts, " "), nil
}

func buildURL(pathValue string, query map[string]string) (string, error) {
	base := strings.TrimSpace(pathValue)
	if base == "" {
		base = "/"
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid path/url %q: %w", pathValue, err)
	}

	values := parsed.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values.Set(key, query[key])
	}
	parsed.RawQuery = values.Encode()

	return parsed.String(), nil
}

func quoteArg(value string) string {
	if value == "" {
		return "''"
	}
	escaped := strings.ReplaceAll(value, "'", "'\"'\"'")
	return "'" + escaped + "'"
}
