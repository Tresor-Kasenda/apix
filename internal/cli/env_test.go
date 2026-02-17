package cli

import (
	"os"
	"testing"
)

func TestResolveEnvNameForShowWithArg(t *testing.T) {
	name, err := resolveEnvNameForShow([]string{"staging"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if name != "staging" {
		t.Fatalf("expected staging, got %q", name)
	}
}

func TestResolveEnvNameForShowWithoutArgUsesCurrentEnv(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	content := "current_env: dev\n"
	if err := os.WriteFile("apix.yaml", []byte(content), 0o644); err != nil {
		t.Fatalf("writing apix.yaml: %v", err)
	}

	name, err := resolveEnvNameForShow(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if name != "dev" {
		t.Fatalf("expected dev, got %q", name)
	}
}

func TestResolveEnvNameForShowWithoutArgFailsWhenNoActive(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	content := "project: test\n"
	if err := os.WriteFile("apix.yaml", []byte(content), 0o644); err != nil {
		t.Fatalf("writing apix.yaml: %v", err)
	}

	_, err := resolveEnvNameForShow(nil)
	if err == nil {
		t.Fatal("expected error when no active environment is set")
	}
}

func withTempDirAsWorkingDirCLI(t *testing.T) {
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
