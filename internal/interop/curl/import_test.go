package curl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCommandFromFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "command.txt"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	req, err := ParseCommand(string(data))
	if err != nil {
		t.Fatalf("parse command: %v", err)
	}
	if req.Method != "POST" {
		t.Fatalf("expected method POST, got %q", req.Method)
	}
	if req.Path != "https://api.example.com/users" {
		t.Fatalf("expected path without query, got %q", req.Path)
	}
	if req.Query["page"] != "2" {
		t.Fatalf("expected query page=2, got %v", req.Query)
	}
	if req.Headers["Content-Type"] != "application/json" || req.Headers["X-Trace"] != "abc" {
		t.Fatalf("expected parsed headers, got %v", req.Headers)
	}
	if req.Body == "" {
		t.Fatalf("expected body from -d flag")
	}
}

func TestParseCommandInvalidQuote(t *testing.T) {
	if _, err := ParseCommand(`curl -X GET "https://api.example.com`); err == nil {
		t.Fatalf("expected unterminated quote error")
	}
}
