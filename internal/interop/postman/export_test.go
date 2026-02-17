package postman

import (
	"testing"

	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestExportCollectionAndReparse(t *testing.T) {
	in := []request.SavedRequest{
		{
			Name:   "login",
			Method: "POST",
			Path:   "/login",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Query: map[string]string{
				"tenant": "acme",
			},
			Body: `{"email":"dev@acme.com","password":"secret"}`,
		},
	}

	data, err := ExportCollection(in, "apix tests")
	if err != nil {
		t.Fatalf("export collection: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty export")
	}

	out, err := ParseCollection(data)
	if err != nil {
		t.Fatalf("reparse exported collection: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 request after reparse, got %d", len(out))
	}

	got := out[0]
	if got.Method != "POST" {
		t.Fatalf("expected method POST, got %q", got.Method)
	}
	if got.Path != "/login" {
		t.Fatalf("expected path /login, got %q", got.Path)
	}
	if got.Query["tenant"] != "acme" {
		t.Fatalf("expected tenant query param, got %v", got.Query)
	}
	if got.Body == "" {
		t.Fatalf("expected body to survive roundtrip")
	}
}
