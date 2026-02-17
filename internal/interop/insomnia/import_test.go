package insomnia

import (
	"path/filepath"
	"testing"
)

func TestParseExportFile(t *testing.T) {
	path := filepath.Join("testdata", "export.json")
	requests, err := ParseExportFile(path)
	if err != nil {
		t.Fatalf("parse insomnia export: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	got := requests[0]
	if got.Method != "POST" {
		t.Fatalf("expected method POST, got %q", got.Method)
	}
	if got.Path != "https://api.example.com/login" {
		t.Fatalf("expected absolute login path, got %q", got.Path)
	}
	if got.Query["source"] != "mobile" || got.Query["version"] != "v1" {
		t.Fatalf("expected merged query values, got %v", got.Query)
	}
	if got.Headers["Content-Type"] != "application/json" {
		t.Fatalf("expected content-type header, got %v", got.Headers)
	}
	if got.Body == "" {
		t.Fatalf("expected body mapped from insomnia payload")
	}
}
