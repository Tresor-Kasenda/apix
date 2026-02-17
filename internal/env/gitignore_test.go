package env

import (
	"os"
	"strings"
	"testing"
)

func TestEnsureGitignoreEntryAppendsOnlyOnce(t *testing.T) {
	withTempDirAsWorkingDir(t)

	initial := "bin/\n.apix/\n"
	if err := os.WriteFile(".gitignore", []byte(initial), 0o644); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}

	if err := EnsureGitignoreEntry(".apix/"); err != nil {
		t.Fatalf("first ensure failed: %v", err)
	}
	if err := EnsureGitignoreEntry(".apix/"); err != nil {
		t.Fatalf("second ensure failed: %v", err)
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)
	count := strings.Count(content, ".apix/")
	if count != 1 {
		t.Fatalf("expected .apix/ to appear once, got %d\ncontent:\n%s", count, content)
	}
}

func TestEnsureGitignoreEntryCreatesFile(t *testing.T) {
	withTempDirAsWorkingDir(t)

	if err := EnsureGitignoreEntry(".apix/"); err != nil {
		t.Fatalf("ensure failed: %v", err)
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	if strings.TrimSpace(string(data)) != ".apix/" {
		t.Fatalf("unexpected .gitignore content: %q", string(data))
	}
}
