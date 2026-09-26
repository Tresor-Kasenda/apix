// Package apixhttp provides the HTTP client and response handling for apix.
package apixhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
}

func ParseResponse(resp *http.Response, duration time.Duration) (*Response, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       body,
		Duration:   duration,
	}, nil
}

func (r *Response) IsJSON() bool {
	ct := r.Headers.Get("Content-Type")
	return strings.Contains(ct, "application/json") || json.Valid(r.Body)
}

// ExtractFirst tries each "|"-separated path in order and returns the first
// non-empty value, e.g. "access_token|token|data.token". This lets a single
// config work across APIs that name their token field differently.
func (r *Response) ExtractFirst(paths string) (string, error) {
	var lastErr error
	for _, path := range strings.Split(paths, "|") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		value, err := r.ExtractField(path)
		if err == nil && value != "" {
			return value, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no value found for %q", paths)
	}
	return "", lastErr
}

func (r *Response) ExtractField(path string) (string, error) {
	if len(r.Body) == 0 {
		return "", fmt.Errorf("empty response body")
	}

	var obj interface{}
	if err := json.Unmarshal(r.Body, &obj); err != nil {
		return "", fmt.Errorf("response is not JSON: %w", err)
	}

	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		switch node := current.(type) {
		case map[string]interface{}:
			next, ok := node[part]
			if !ok {
				return "", fmt.Errorf("field %q not found in path %q", part, path)
			}
			current = next
		case []interface{}:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(node) {
				return "", fmt.Errorf("invalid array index %q in path %q", part, path)
			}
			current = node[idx]
		default:
			return "", fmt.Errorf("cannot navigate path %q: not an object at %q", path, part)
		}
	}

	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		return fmt.Sprintf("%v", v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("extracting field %q: %w", path, err)
		}
		return string(b), nil
	}
}
