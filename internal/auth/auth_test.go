package auth

import (
	"testing"

	"github.com/Tresor-Kasend/apix/internal/config"
)

func TestApplyBearer(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{Type: "bearer", Token: "abc123"}}
	headers := map[string]string{}

	if err := Apply(headers, cfg, map[string]string{}); err != nil {
		t.Fatalf("apply bearer failed: %v", err)
	}

	if got := headers["Authorization"]; got != "Bearer abc123" {
		t.Fatalf("expected Authorization Bearer abc123, got %q", got)
	}
}

func TestApplyBasic(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{Type: "basic", Username: "john", Password: "secret"}}
	headers := map[string]string{}

	if err := Apply(headers, cfg, map[string]string{}); err != nil {
		t.Fatalf("apply basic failed: %v", err)
	}

	if got := headers["Authorization"]; got != "Basic am9objpzZWNyZXQ=" {
		t.Fatalf("unexpected basic header: %q", got)
	}
}

func TestApplyAPIKey(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{
		Type:       "api_key",
		APIKey:     "k-42",
		HeaderName: "X-API-Key",
	}}
	headers := map[string]string{}

	if err := Apply(headers, cfg, map[string]string{}); err != nil {
		t.Fatalf("apply api_key failed: %v", err)
	}

	if got := headers["X-API-Key"]; got != "k-42" {
		t.Fatalf("unexpected api key header: %q", got)
	}
}

func TestApplyCustom(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{
		Type:         "custom",
		HeaderName:   "X-Custom-Auth",
		HeaderFormat: "Token ${TOKEN}|${USERNAME}",
		Token:        "t-99",
		Username:     "alice",
	}}
	headers := map[string]string{}

	if err := Apply(headers, cfg, map[string]string{}); err != nil {
		t.Fatalf("apply custom failed: %v", err)
	}

	if got := headers["X-Custom-Auth"]; got != "Token t-99|alice" {
		t.Fatalf("unexpected custom header: %q", got)
	}
}

func TestApplyBasicMissingCredentials(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{Type: "basic", Username: "john"}}
	headers := map[string]string{}

	if err := Apply(headers, cfg, map[string]string{}); err == nil {
		t.Fatal("expected error for missing basic password")
	}
}

func TestApplyAPIKeyMissingKey(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{Type: "api_key"}}
	headers := map[string]string{}

	if err := Apply(headers, cfg, map[string]string{}); err == nil {
		t.Fatal("expected error for missing api key")
	}
}
