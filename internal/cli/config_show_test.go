package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tresor-Kasend/apix/internal/config"
)

func TestConfigShowMergedWithActiveEnv(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	apixYAML := `project: test
base_url: http://localhost:8000
timeout: 30
current_env: staging
headers:
  Accept: application/json
auth:
  type: none
variables:
  API_KEY: base-key
`
	if err := os.WriteFile("apix.yaml", []byte(apixYAML), 0o644); err != nil {
		t.Fatalf("writing apix.yaml: %v", err)
	}

	if err := os.MkdirAll("env", 0o755); err != nil {
		t.Fatalf("creating env dir: %v", err)
	}
	stagingYAML := `base_url: https://staging.api.example.com
headers:
  X-Env: staging
variables:
  USER_ID: "42"
`
	if err := os.WriteFile(filepath.Join("env", "staging.yaml"), []byte(stagingYAML), 0o644); err != nil {
		t.Fatalf("writing env/staging.yaml: %v", err)
	}

	cmd := newConfigShowCmd()
	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute config show: %v", err)
		}
	})

	var got config.Config
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("unmarshal config show output: %v\noutput:\n%s", err, out)
	}

	if got.BaseURL != "https://staging.api.example.com" {
		t.Fatalf("expected merged base_url from env, got %q", got.BaseURL)
	}
	if mapLookupCaseInsensitive(got.Headers, "Accept") != "application/json" {
		t.Fatalf("expected base header preserved, got headers=%v", got.Headers)
	}
	if mapLookupCaseInsensitive(got.Headers, "X-Env") != "staging" {
		t.Fatalf("expected env header merged, got headers=%v", got.Headers)
	}
	if mapLookupCaseInsensitive(got.Variables, "API_KEY") != "base-key" || mapLookupCaseInsensitive(got.Variables, "USER_ID") != "42" {
		t.Fatalf("expected merged variables, got %v", got.Variables)
	}
}

func mapLookupCaseInsensitive(values map[string]string, key string) string {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}
