package apixhttp

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient Doer
}

type ClientConfig struct {
	Timeout         time.Duration
	FollowRedirects bool
}

type RequestOptions struct {
	Method  string
	URL     string
	Headers map[string]string
	Query   map[string]string
	Body    io.Reader
}

func NewClient(timeout time.Duration) *Client {
	return NewClientWithConfig(ClientConfig{
		Timeout:         timeout,
		FollowRedirects: true,
	})
}

func NewClientWithConfig(cfg ClientConfig) *Client {
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
	}

	if !cfg.FollowRedirects {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Client{httpClient: httpClient}
}

func NewClientWithDoer(d Doer) *Client {
	return &Client{httpClient: d}
}

// Send builds and executes an HTTP request, returning the parsed response.
func (c *Client) Send(opts RequestOptions) (*Response, error) {
	req, err := BuildRequest(opts)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	return ParseResponse(resp, duration)
}

func BuildRequest(opts RequestOptions) (*http.Request, error) {
	req, err := http.NewRequest(opts.Method, opts.URL, opts.Body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	if len(opts.Query) > 0 {
		q := req.URL.Query()
		for k, v := range opts.Query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req, nil
}
