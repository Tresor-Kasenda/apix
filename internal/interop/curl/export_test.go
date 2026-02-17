package curl

import (
	"strings"
	"testing"

	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestToCommandAndParseRoundTrip(t *testing.T) {
	in := request.SavedRequest{
		Method: "PUT",
		Path:   "/users/42",
		Headers: map[string]string{
			"Accept":  "application/json",
			"X-Token": "abc123",
		},
		Query: map[string]string{
			"force": "true",
		},
		Body: `{"name":"John"}`,
	}

	command, err := ToCommand(in)
	if err != nil {
		t.Fatalf("to command: %v", err)
	}
	if !strings.Contains(command, "curl") || !strings.Contains(command, "-X 'PUT'") {
		t.Fatalf("unexpected command output: %s", command)
	}
	if !strings.Contains(command, "/users/42?force=true") {
		t.Fatalf("expected query URL in command, got: %s", command)
	}

	out, err := ParseCommand(command)
	if err != nil {
		t.Fatalf("parse exported command: %v", err)
	}
	if out.Method != "PUT" {
		t.Fatalf("expected method PUT after roundtrip, got %q", out.Method)
	}
	if out.Path != "/users/42" {
		t.Fatalf("expected path /users/42 after roundtrip, got %q", out.Path)
	}
	if out.Query["force"] != "true" {
		t.Fatalf("expected query force=true after roundtrip, got %v", out.Query)
	}
	if out.Headers["X-Token"] != "abc123" {
		t.Fatalf("expected header X-Token after roundtrip, got %v", out.Headers)
	}
	if out.Body == "" {
		t.Fatalf("expected non-empty body after roundtrip")
	}
}
