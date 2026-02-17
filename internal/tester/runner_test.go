package tester

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestRunSummaryAndCounts(t *testing.T) {
	dir := t.TempDir()
	writeRequestYAML(t, filepath.Join(dir, "pass.yaml"), ""+
		"name: pass\n"+
		"method: GET\n"+
		"path: /ok\n"+
		"expect:\n"+
		"  status:\n"+
		"    eq: 200\n")
	writeRequestYAML(t, filepath.Join(dir, "fail.yaml"), ""+
		"name: fail\n"+
		"method: GET\n"+
		"path: /fail\n"+
		"expect:\n"+
		"  status:\n"+
		"    eq: 201\n")
	writeRequestYAML(t, filepath.Join(dir, "ignored.yaml"), ""+
		"name: ignored\n"+
		"method: GET\n"+
		"path: /ignored\n")

	suite, err := Run(RunnerOptions{Dir: dir}, func(name string, saved *request.SavedRequest, vars map[string]string, envOverride string) (*apixhttp.Response, error) {
		return &apixhttp.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"ok":true}`),
		}, nil
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if suite.Total != 2 {
		t.Fatalf("expected total=2 (ignored without expect), got %d", suite.Total)
	}
	if suite.Passed != 1 {
		t.Fatalf("expected passed=1, got %d", suite.Passed)
	}
	if suite.Failed != 1 {
		t.Fatalf("expected failed=1, got %d", suite.Failed)
	}
	if suite.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", suite.ExitCode())
	}
}

func TestRunExitCodeAndSingleRequestSelection(t *testing.T) {
	dir := t.TempDir()
	writeRequestYAML(t, filepath.Join(dir, "login.yaml"), ""+
		"name: login\n"+
		"method: POST\n"+
		"path: /login\n"+
		"expect:\n"+
		"  status:\n"+
		"    eq: 200\n")

	suite, err := Run(RunnerOptions{Name: "login", Dir: dir}, func(name string, saved *request.SavedRequest, vars map[string]string, envOverride string) (*apixhttp.Response, error) {
		if name != "login" {
			return nil, fmt.Errorf("unexpected test name %q", name)
		}
		return &apixhttp.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"ok":true}`),
		}, nil
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if suite.Total != 1 || suite.Passed != 1 || suite.Failed != 0 {
		t.Fatalf("unexpected suite summary: %+v", suite)
	}
	if suite.ExitCode() != 0 {
		t.Fatalf("expected exit code 0, got %d", suite.ExitCode())
	}
}

func writeRequestYAML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
