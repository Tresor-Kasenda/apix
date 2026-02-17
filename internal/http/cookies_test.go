package apixhttp

import (
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
)

func TestPersistentCookieJarPersistsCookies(t *testing.T) {
	t.Parallel()

	jarPath := filepath.Join(t.TempDir(), "cookies.jar")
	targetURL, err := url.Parse("https://api.example.com/login")
	if err != nil {
		t.Fatalf("parse target URL: %v", err)
	}

	jarOne, err := NewPersistentCookieJar(jarPath)
	if err != nil {
		t.Fatalf("create first cookie jar: %v", err)
	}
	jarOne.SetCookies(targetURL, []*http.Cookie{
		{
			Name:  "session_id",
			Value: "abc123",
			Path:  "/",
		},
	})

	jarTwo, err := NewPersistentCookieJar(jarPath)
	if err != nil {
		t.Fatalf("create second cookie jar: %v", err)
	}
	cookies := jarTwo.Cookies(targetURL)
	if len(cookies) == 0 {
		t.Fatalf("expected persisted cookies to be loaded")
	}
	if cookies[0].Name != "session_id" || cookies[0].Value != "abc123" {
		t.Fatalf("unexpected cookie loaded: %+v", cookies[0])
	}
}
