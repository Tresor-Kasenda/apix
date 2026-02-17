package apixhttp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient Doer
	retry      int
	retryDelay time.Duration
	initErr    error
}

type ClientConfig struct {
	Timeout         time.Duration
	FollowRedirects bool
	Network         NetworkOptions
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
	client := &Client{}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if strings.TrimSpace(cfg.Network.ProxyURL) != "" {
		parsedProxy, err := url.Parse(strings.TrimSpace(cfg.Network.ProxyURL))
		if err != nil {
			client.initErr = fmt.Errorf("invalid proxy URL %q: %w", cfg.Network.ProxyURL, err)
			return client
		}
		transport.Proxy = http.ProxyURL(parsedProxy)
	}

	tlsConfig := &tls.Config{}
	if cfg.Network.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}
	if cfg.Network.CertFile != "" || cfg.Network.KeyFile != "" {
		if cfg.Network.CertFile == "" || cfg.Network.KeyFile == "" {
			client.initErr = fmt.Errorf("--cert and --key must be provided together")
			return client
		}
		cert, err := tls.LoadX509KeyPair(cfg.Network.CertFile, cfg.Network.KeyFile)
		if err != nil {
			client.initErr = fmt.Errorf("loading client TLS certificate/key: %w", err)
			return client
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if cfg.Network.Insecure || len(tlsConfig.Certificates) > 0 {
		transport.TLSClientConfig = tlsConfig
	}

	var jar http.CookieJar
	if !cfg.Network.NoCookies {
		jarPath := cfg.Network.CookieJarPath
		if strings.TrimSpace(jarPath) == "" {
			jarPath = DefaultCookieJarPath
		}
		persistentJar, err := NewPersistentCookieJar(jarPath)
		if err != nil {
			client.initErr = err
			return client
		}
		jar = persistentJar
	}

	httpClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		Jar:       jar,
	}

	if !cfg.FollowRedirects {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	retry := cfg.Network.Retry
	if retry < 0 {
		retry = 0
	}
	retryDelay := cfg.Network.RetryDelay
	if retry > 0 && retryDelay <= 0 {
		retryDelay = DefaultRetryDelay
	}

	client.httpClient = httpClient
	client.retry = retry
	client.retryDelay = retryDelay
	return client
}

func NewClientWithDoer(d Doer) *Client {
	return &Client{
		httpClient: d,
	}
}

// Send builds and executes an HTTP request, returning the parsed response.
func (c *Client) Send(opts RequestOptions) (*Response, error) {
	if c.initErr != nil {
		return nil, c.initErr
	}

	bodyBytes, err := readBodyBytes(opts.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	attempts := c.retry + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := BuildRequest(RequestOptions{
			Method:  opts.Method,
			URL:     opts.URL,
			Headers: opts.Headers,
			Query:   opts.Query,
			Body:    bytes.NewReader(bodyBytes),
		})
		if err != nil {
			return nil, err
		}

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		duration := time.Since(start)

		if err != nil {
			lastErr = err
			if attempt < attempts && shouldRetryNetworkError(err) {
				sleepRetryDelay(c.retryDelay, attempt)
				continue
			}
			return nil, fmt.Errorf("sending request: %w", err)
		}

		if attempt < attempts && shouldRetryStatus(resp.StatusCode) {
			_ = resp.Body.Close()
			sleepRetryDelay(c.retryDelay, attempt)
			continue
		}

		return ParseResponse(resp, duration)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("sending request: %w", lastErr)
	}
	return nil, fmt.Errorf("request failed after retries")
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

func readBodyBytes(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return io.ReadAll(body)
}

func sleepRetryDelay(base time.Duration, attempt int) {
	delay := retryDelayForAttempt(base, attempt)
	if delay > 0 {
		time.Sleep(delay)
	}
}
