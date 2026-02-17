package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestRunWithEnvOverrideDoesNotPersistCurrentEnv(t *testing.T) {
	withTempDirAsWorkingDirRun(t)

	var devCalls int32
	devServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&devCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"env":"dev"}`))
	}))
	defer devServer.Close()

	var stagingCalls int32
	stagingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&stagingCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"env":"staging"}`))
	}))
	defer stagingServer.Close()

	apixYAML := fmt.Sprintf(`project: test
base_url: %s
timeout: 10
current_env: dev
headers:
  Accept: application/json
auth:
  type: none
`, devServer.URL)

	if err := os.WriteFile("apix.yaml", []byte(apixYAML), 0o644); err != nil {
		t.Fatalf("writing apix.yaml: %v", err)
	}

	if err := os.MkdirAll("env", 0o755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("env", "dev.yaml"), []byte(fmt.Sprintf("base_url: %s\n", devServer.URL)), 0o644); err != nil {
		t.Fatalf("writing env/dev.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join("env", "staging.yaml"), []byte(fmt.Sprintf("base_url: %s\n", stagingServer.URL)), 0o644); err != nil {
		t.Fatalf("writing env/staging.yaml: %v", err)
	}

	if err := os.MkdirAll("requests", 0o755); err != nil {
		t.Fatalf("creating requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("requests", "ping.yaml"), []byte("name: ping\nmethod: GET\npath: /\n"), 0o644); err != nil {
		t.Fatalf("writing request file: %v", err)
	}

	err := executeSavedRequest("ping", ExecuteOptions{
		EnvOverride:    "staging",
		Silent:         true,
		SuppressOutput: true,
	})
	if err != nil {
		t.Fatalf("executeSavedRequest failed: %v", err)
	}

	if atomic.LoadInt32(&devCalls) != 0 {
		t.Fatalf("expected dev server to not be called, got %d", devCalls)
	}
	if atomic.LoadInt32(&stagingCalls) != 1 {
		t.Fatalf("expected staging server to be called once, got %d", stagingCalls)
	}

	data, err := os.ReadFile("apix.yaml")
	if err != nil {
		t.Fatalf("reading apix.yaml: %v", err)
	}
	if !containsLine(string(data), "current_env: dev") {
		t.Fatalf("expected current_env to remain dev, got:\n%s", string(data))
	}
}

func containsLine(content, expected string) bool {
	for _, line := range splitLines(content) {
		if line == expected {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start <= len(s)-1 {
		lines = append(lines, s[start:])
	}
	return lines
}

func withTempDirAsWorkingDirRun(t *testing.T) {
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
