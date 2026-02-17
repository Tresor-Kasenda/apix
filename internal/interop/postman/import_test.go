package postman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCollectionFile(t *testing.T) {
	path := filepath.Join("testdata", "collection.json")
	requests, err := ParseCollectionFile(path)
	if err != nil {
		t.Fatalf("parse collection file: %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requests))
	}

	first := requests[0]
	if first.Method != "GET" {
		t.Fatalf("expected first method GET, got %q", first.Method)
	}
	if first.Path != "/users" {
		t.Fatalf("expected first path /users, got %q", first.Path)
	}
	if first.Query["page"] != "2" || first.Query["limit"] != "10" {
		t.Fatalf("expected merged query params, got %v", first.Query)
	}
	if first.Headers["Accept"] != "application/json" {
		t.Fatalf("expected first header Accept, got %v", first.Headers)
	}

	second := requests[1]
	if second.Method != "POST" {
		t.Fatalf("expected second method POST, got %q", second.Method)
	}
	if second.Path != "https://api.example.com/login" {
		t.Fatalf("expected second path absolute URL, got %q", second.Path)
	}
	if second.Body == "" {
		t.Fatalf("expected second body to be mapped")
	}
}

func TestParseCollectionInvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "invalid.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	if _, err := ParseCollectionFile(path); err == nil {
		t.Fatalf("expected parse error")
	}
}
