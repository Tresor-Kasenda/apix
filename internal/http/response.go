// Package apixhttp provides the HTTP client and response handling for apix.
package apixhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		m, ok := current.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("cannot navigate path %q: not an object at %q", path, part)
		}
		current, ok = m[part]
		if !ok {
			return "", fmt.Errorf("field %q not found in path %q", part, path)
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
