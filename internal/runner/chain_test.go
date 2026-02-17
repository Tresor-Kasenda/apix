package runner

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestRunChainPropagatesCapturedVariables(t *testing.T) {
	withTempDirAsWorkingDirRunner(t)

	mustSaveRequest(t, "login", request.SavedRequest{
		Method:  "POST",
		Path:    "/login",
		Capture: map[string]string{"USER_ID": "data.user.id"},
	})
	mustSaveRequest(t, "get-profile", request.SavedRequest{Method: "GET", Path: "/users/${USER_ID}"})

	result, err := RunChain([]string{"login", "get-profile"}, map[string]string{}, "", func(name string, vars map[string]string, env string) (*apixhttp.Response, error) {
		switch name {
		case "login":
			return makeJSONResponse(http.StatusOK, `{"data":{"user":{"id":42}}}`), nil
		case "get-profile":
			if got := vars["USER_ID"]; got != "42" {
				t.Fatalf("expected USER_ID to be propagated as 42, got %q", got)
			}
			return makeJSONResponse(http.StatusOK, `{"ok":true}`), nil
		default:
			return nil, fmt.Errorf("unexpected request name %q", name)
		}
	})
	if err != nil {
		t.Fatalf("run chain failed: %v", err)
	}
	if result.Executed != 2 {
		t.Fatalf("expected 2 executed requests, got %d", result.Executed)
	}
	if result.Captured != 1 {
		t.Fatalf("expected 1 captured variable, got %d", result.Captured)
	}
	if result.FinalVars["USER_ID"] != "42" {
		t.Fatalf("expected final USER_ID=42, got %q", result.FinalVars["USER_ID"])
	}
}

func TestRunChainCapturedVariablesOverridePrevious(t *testing.T) {
	withTempDirAsWorkingDirRunner(t)

	mustSaveRequest(t, "login", request.SavedRequest{
		Method:  "POST",
		Path:    "/login",
		Capture: map[string]string{"USER_ID": "data.user.id"},
	})
	mustSaveRequest(t, "followup", request.SavedRequest{Method: "GET", Path: "/users/${USER_ID}"})

	_, err := RunChain([]string{"login", "followup"}, map[string]string{"USER_ID": "1"}, "", func(name string, vars map[string]string, env string) (*apixhttp.Response, error) {
		switch name {
		case "login":
			if got := vars["USER_ID"]; got != "1" {
				t.Fatalf("expected initial USER_ID=1, got %q", got)
			}
			return makeJSONResponse(http.StatusOK, `{"data":{"user":{"id":2}}}`), nil
		case "followup":
			if got := vars["USER_ID"]; got != "2" {
				t.Fatalf("expected overwritten USER_ID=2, got %q", got)
			}
			return makeJSONResponse(http.StatusOK, `{"ok":true}`), nil
		default:
			return nil, fmt.Errorf("unexpected request name %q", name)
		}
	})
	if err != nil {
		t.Fatalf("run chain failed: %v", err)
	}
}

func TestRunChainStopsOnHTTPError(t *testing.T) {
	withTempDirAsWorkingDirRunner(t)

	mustSaveRequest(t, "step1", request.SavedRequest{Method: "GET", Path: "/s1"})
	mustSaveRequest(t, "step2", request.SavedRequest{Method: "GET", Path: "/s2"})
	mustSaveRequest(t, "step3", request.SavedRequest{Method: "GET", Path: "/s3"})

	calls := 0
	result, err := RunChain([]string{"step1", "step2", "step3"}, nil, "", func(name string, vars map[string]string, env string) (*apixhttp.Response, error) {
		calls++
		switch name {
		case "step1":
			return makeJSONResponse(http.StatusOK, `{"ok":true}`), nil
		case "step2":
			return makeJSONResponse(http.StatusInternalServerError, `{"error":"boom"}`), nil
		default:
			return makeJSONResponse(http.StatusOK, `{"ok":true}`), nil
		}
	})
	if err == nil {
		t.Fatal("expected chain error on HTTP 500")
	}
	if !strings.Contains(err.Error(), "step2") {
		t.Fatalf("expected error to mention step2, got: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected execution to stop after second step, calls=%d", calls)
	}
	if result.Executed != 2 {
		t.Fatalf("expected executed=2, got %d", result.Executed)
	}
}

func mustSaveRequest(t *testing.T, name string, req request.SavedRequest) {
	t.Helper()
	if err := request.Save(name, req); err != nil {
		t.Fatalf("saving request %q: %v", name, err)
	}
}

func makeJSONResponse(status int, body string) *apixhttp.Response {
	return &apixhttp.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(body),
	}
}

func withTempDirAsWorkingDirRunner(t *testing.T) {
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
