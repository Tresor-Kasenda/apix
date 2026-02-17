package apixhttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientNoFollowRedirect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{
		Timeout:         2 * time.Second,
		FollowRedirects: false,
	})

	resp, err := client.Send(RequestOptions{
		Method: http.MethodGet,
		URL:    srv.URL + "/redirect",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, resp.StatusCode)
	}
}

func TestClientTimeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow"))
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{
		Timeout:         20 * time.Millisecond,
		FollowRedirects: true,
	})

	_, err := client.Send(RequestOptions{
		Method: http.MethodGet,
		URL:    srv.URL,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded error, got %v", err)
	}
}

func TestClientRetryPolicyOn5xx(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"temporary"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{
		Timeout:         2 * time.Second,
		FollowRedirects: true,
		Network: NetworkOptions{
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
		},
	})

	resp, err := client.Send(RequestOptions{
		Method: http.MethodGet,
		URL:    srv.URL + "/unstable",
	})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected final status 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 calls with retry, got %d", calls)
	}
}

func TestClientProxyUsage(t *testing.T) {
	t.Parallel()

	var proxyHits int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&proxyHits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"via":"proxy"}`))
	}))
	defer proxy.Close()

	client := NewClientWithConfig(ClientConfig{
		Timeout:         2 * time.Second,
		FollowRedirects: true,
		Network: NetworkOptions{
			ProxyURL: proxy.URL,
		},
	})

	resp, err := client.Send(RequestOptions{
		Method: http.MethodGet,
		URL:    "http://example.com/users",
	})
	if err != nil {
		t.Fatalf("expected proxy request to succeed, got %v", err)
	}
	if atomic.LoadInt32(&proxyHits) == 0 {
		t.Fatal("expected proxy to be used at least once")
	}
	if !strings.Contains(string(resp.Body), `"via":"proxy"`) {
		t.Fatalf("expected proxy body, got %s", string(resp.Body))
	}
}

func TestClientInsecureTLS(t *testing.T) {
	t.Parallel()

	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "secure")
	}))
	defer tlsServer.Close()

	secureClient := NewClientWithConfig(ClientConfig{
		Timeout:         2 * time.Second,
		FollowRedirects: true,
	})
	if _, err := secureClient.Send(RequestOptions{
		Method: http.MethodGet,
		URL:    tlsServer.URL,
	}); err == nil {
		t.Fatal("expected TLS validation error without --insecure")
	}

	insecureClient := NewClientWithConfig(ClientConfig{
		Timeout:         2 * time.Second,
		FollowRedirects: true,
		Network: NetworkOptions{
			Insecure: true,
		},
	})
	resp, err := insecureClient.Send(RequestOptions{
		Method: http.MethodGet,
		URL:    tlsServer.URL,
	})
	if err != nil {
		t.Fatalf("expected success with insecure TLS, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with insecure TLS, got %d", resp.StatusCode)
	}
}
