package auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Tresor-Kasend/apix/internal/config"
)

func TestRefreshIfNeededSuccess(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{LoginRequest: "login"}}
	called := 0

	retry, err := RefreshIfNeeded(cfg, "get-users", http.StatusUnauthorized, false, false, func(name string) error {
		called++
		if name != "login" {
			t.Fatalf("expected login request name, got %q", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !retry {
		t.Fatal("expected retry=true")
	}
	if called != 1 {
		t.Fatalf("expected callback called once, got %d", called)
	}
}

func TestRefreshIfNeededFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{LoginRequest: "login"}}

	retry, err := RefreshIfNeeded(cfg, "get-users", http.StatusUnauthorized, false, false, func(name string) error {
		return errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if retry {
		t.Fatal("expected retry=false on refresh failure")
	}
}

func TestRefreshIfNeededNoLoop(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{Auth: config.AuthConfig{LoginRequest: "login"}}
	called := false

	retry, err := RefreshIfNeeded(cfg, "login", http.StatusUnauthorized, false, false, func(name string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retry {
		t.Fatal("expected retry=false when request is login request")
	}
	if called {
		t.Fatal("expected callback not to be called")
	}

	retry, err = RefreshIfNeeded(cfg, "get-users", http.StatusUnauthorized, true, false, func(name string) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if retry {
		t.Fatal("expected retry=false when already retried")
	}
}
