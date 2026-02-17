package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyEnvironment(t *testing.T) {
	withTempDirAsWorkingDir(t)

	if err := os.MkdirAll("env", 0o755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	sourceContent := "base_url: https://api.dev.local\nvariables:\n  FOO: bar\n"
	if err := os.WriteFile(filepath.Join("env", "dev.yaml"), []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("writing source env: %v", err)
	}

	if err := Copy("dev", "staging"); err != nil {
		t.Fatalf("copy environment failed: %v", err)
	}

	copied, err := os.ReadFile(filepath.Join("env", "staging.yaml"))
	if err != nil {
		t.Fatalf("reading copied env: %v", err)
	}

	if string(copied) != sourceContent {
		t.Fatalf("copied content mismatch\nwant:\n%s\ngot:\n%s", sourceContent, string(copied))
	}
}

func TestDeleteEnvironment(t *testing.T) {
	withTempDirAsWorkingDir(t)

	if err := os.MkdirAll("env", 0o755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	target := filepath.Join("env", "staging.yaml")
	if err := os.WriteFile(target, []byte("base_url: https://api.staging.local\n"), 0o644); err != nil {
		t.Fatalf("writing env file: %v", err)
	}

	if err := Delete("staging", "dev"); err != nil {
		t.Fatalf("delete environment failed: %v", err)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected environment file to be deleted, stat err=%v", err)
	}
}

func TestDeleteActiveEnvironment(t *testing.T) {
	withTempDirAsWorkingDir(t)

	if err := os.MkdirAll("env", 0o755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join("env", "dev.yaml"), []byte("base_url: http://localhost\n"), 0o644); err != nil {
		t.Fatalf("writing env file: %v", err)
	}

	err := Delete("dev", "dev")
	if err == nil {
		t.Fatal("expected error when deleting active environment")
	}
	if !strings.Contains(err.Error(), "cannot delete active environment") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func withTempDirAsWorkingDir(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}
}
