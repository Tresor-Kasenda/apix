package cli

import (
	"testing"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
)

func TestRootRegistersWatchCommand(t *testing.T) {
	t.Parallel()

	cmd := rootCmd()
	found, _, err := cmd.Find([]string{"watch"})
	if err != nil {
		t.Fatalf("expected watch command to be registered: %v", err)
	}
	if found == nil || found.Name() != "watch" {
		t.Fatalf("expected to find watch command, got %+v", found)
	}
}

func TestParseQueryFlags(t *testing.T) {
	t.Parallel()

	result := parseQueryFlags([]string{"page=2&limit=10", "q=search"})

	if result["page"] != "2" {
		t.Fatalf("expected page=2, got %q", result["page"])
	}
	if result["limit"] != "10" {
		t.Fatalf("expected limit=10, got %q", result["limit"])
	}
	if result["q"] != "search" {
		t.Fatalf("expected q=search, got %q", result["q"])
	}
}

func TestParseFieldSlice(t *testing.T) {
	t.Parallel()

	fields, err := parseFieldSlice([]string{"name=John", "file=@avatar.png"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []apixhttp.FormField{
		{Key: "name", Value: "John"},
		{Key: "file", Value: "@avatar.png"},
	}

	if len(fields) != len(expected) {
		t.Fatalf("expected %d fields, got %d", len(expected), len(fields))
	}
	for i := range expected {
		if fields[i] != expected[i] {
			t.Fatalf("expected field %d = %+v, got %+v", i, expected[i], fields[i])
		}
	}
}

func TestParseFieldSliceInvalid(t *testing.T) {
	t.Parallel()

	_, err := parseFieldSlice([]string{"invalid-field"})
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestValidateDisplayModes(t *testing.T) {
	t.Parallel()

	err := validateDisplayModes(ExecuteOptions{HeadersOnly: true, BodyOnly: true})
	if err == nil {
		t.Fatal("expected conflicting display mode error")
	}
}
