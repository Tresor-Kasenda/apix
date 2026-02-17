package watch

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestExecutorRunsPreMainPostInOrder(t *testing.T) {
	withTempDirAsWorkingDirWatch(t)

	mustSaveWatchRequest(t, "login", request.SavedRequest{Method: "POST", Path: "/login"})
	mustSaveWatchRequest(t, "metrics", request.SavedRequest{Method: "POST", Path: "/metrics"})
	mustSaveWatchRequest(t, "main", request.SavedRequest{
		Method:  "GET",
		Path:    "/users/${USER_ID}",
		Capture: map[string]string{"USER_ID": "data.user.id"},
		PreRequest: []request.Hook{
			{
				Run:     "login",
				Capture: map[string]string{"SESSION_ID": "data.session.id"},
			},
		},
		PostRequest: []request.Hook{{Run: "metrics"}},
	})

	order := make([]string, 0, 3)
	executor, err := NewExecutor(func(name string, vars map[string]string, envOverride string) (*apixhttp.Response, error) {
		order = append(order, name)
		switch name {
		case "login":
			return makeWatchJSONResponse(http.StatusOK, `{"data":{"session":{"id":"sess-1"}}}`), nil
		case "main":
			if got := vars["SESSION_ID"]; got != "sess-1" {
				t.Fatalf("expected SESSION_ID to be available for main request, got %q", got)
			}
			return makeWatchJSONResponse(http.StatusOK, `{"data":{"user":{"id":42}}}`), nil
		case "metrics":
			if got := vars["USER_ID"]; got != "42" {
				t.Fatalf("expected USER_ID=42 for post hook, got %q", got)
			}
			return makeWatchJSONResponse(http.StatusOK, `{"ok":true}`), nil
		default:
			return nil, fmt.Errorf("unexpected request %q", name)
		}
	}, ExecutorOptions{})
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	result, err := executor.Run("main", map[string]string{"INIT": "1"}, "")
	if err != nil {
		t.Fatalf("executor run failed: %v", err)
	}

	expectedOrder := []string{"login", "main", "metrics"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("expected order %v, got %v", expectedOrder, order)
	}
	if result.Executed != 3 {
		t.Fatalf("expected executed=3, got %d", result.Executed)
	}
	if result.Captured != 2 {
		t.Fatalf("expected captured=2, got %d", result.Captured)
	}
	if result.FinalVars["SESSION_ID"] != "sess-1" {
		t.Fatalf("expected final SESSION_ID=sess-1, got %q", result.FinalVars["SESSION_ID"])
	}
	if result.FinalVars["USER_ID"] != "42" {
		t.Fatalf("expected final USER_ID=42, got %q", result.FinalVars["USER_ID"])
	}
}

func TestExecutorDetectsRecursiveHooks(t *testing.T) {
	withTempDirAsWorkingDirWatch(t)

	mustSaveWatchRequest(t, "a", request.SavedRequest{Method: "GET", Path: "/a", PreRequest: []request.Hook{{Run: "b"}}})
	mustSaveWatchRequest(t, "b", request.SavedRequest{Method: "GET", Path: "/b", PreRequest: []request.Hook{{Run: "a"}}})

	executor, err := NewExecutor(func(name string, vars map[string]string, envOverride string) (*apixhttp.Response, error) {
		return makeWatchJSONResponse(http.StatusOK, `{"ok":true}`), nil
	}, ExecutorOptions{})
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Run("a", nil, "")
	if err == nil {
		t.Fatal("expected recursion guardrail error")
	}
	if !strings.Contains(err.Error(), "recursive reference") {
		t.Fatalf("expected recursion error, got: %v", err)
	}
}

func TestExecutorStopsOnHookError(t *testing.T) {
	withTempDirAsWorkingDirWatch(t)

	mustSaveWatchRequest(t, "login", request.SavedRequest{Method: "POST", Path: "/login"})
	mustSaveWatchRequest(t, "main", request.SavedRequest{
		Method:     "GET",
		Path:       "/protected",
		PreRequest: []request.Hook{{Run: "login"}},
	})

	calls := make([]string, 0, 2)
	executor, err := NewExecutor(func(name string, vars map[string]string, envOverride string) (*apixhttp.Response, error) {
		calls = append(calls, name)
		switch name {
		case "login":
			return makeWatchJSONResponse(http.StatusUnauthorized, `{"error":"invalid credentials"}`), nil
		case "main":
			return makeWatchJSONResponse(http.StatusOK, `{"ok":true}`), nil
		default:
			return nil, fmt.Errorf("unexpected request %q", name)
		}
	}, ExecutorOptions{})
	if err != nil {
		t.Fatalf("new executor: %v", err)
	}

	_, err = executor.Run("main", nil, "")
	if err == nil {
		t.Fatal("expected pre hook failure")
	}
	if !strings.Contains(err.Error(), "pre_request hook #1") {
		t.Fatalf("expected explicit pre hook error, got: %v", err)
	}

	expectedCalls := []string{"login"}
	if !reflect.DeepEqual(calls, expectedCalls) {
		t.Fatalf("expected only login to run, got %v", calls)
	}
}

func mustSaveWatchRequest(t *testing.T, name string, req request.SavedRequest) {
	t.Helper()
	if err := request.Save(name, req); err != nil {
		t.Fatalf("saving request %q: %v", name, err)
	}
}

func makeWatchJSONResponse(status int, body string) *apixhttp.Response {
	return &apixhttp.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(body),
	}
}

func withTempDirAsWorkingDirWatch(t *testing.T) {
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
