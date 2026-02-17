package request

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListSavedSorted(t *testing.T) {
	withTempDirAsWorkingDirRequest(t)

	if err := os.MkdirAll("requests", 0o755); err != nil {
		t.Fatalf("creating requests dir: %v", err)
	}

	for _, name := range []string{"zeta.yaml", "alpha.yaml", "middle.yaml"} {
		if err := os.WriteFile(filepath.Join("requests", name), []byte("name: test\n"), 0o644); err != nil {
			t.Fatalf("writing request file %s: %v", name, err)
		}
	}

	names, err := ListSaved()
	if err != nil {
		t.Fatalf("list saved failed: %v", err)
	}

	expected := []string{"alpha", "middle", "zeta"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}
}

func TestDeleteRequest(t *testing.T) {
	withTempDirAsWorkingDirRequest(t)

	if err := Save("login", SavedRequest{Method: "POST", Path: "/login"}); err != nil {
		t.Fatalf("saving request: %v", err)
	}
	if err := Delete("login"); err != nil {
		t.Fatalf("delete request failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join("requests", "login.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected request file deleted, stat err=%v", err)
	}
}

func TestRenameRequest(t *testing.T) {
	withTempDirAsWorkingDirRequest(t)

	if err := Save("login", SavedRequest{Method: "POST", Path: "/login"}); err != nil {
		t.Fatalf("saving request: %v", err)
	}

	if err := Rename("login", "auth-login"); err != nil {
		t.Fatalf("rename request failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join("requests", "login.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected old file removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join("requests", "auth-login.yaml")); err != nil {
		t.Fatalf("expected new file exists, stat err=%v", err)
	}
}

func TestLoadRequestWithCapture(t *testing.T) {
	withTempDirAsWorkingDirRequest(t)

	if err := os.MkdirAll("requests", 0o755); err != nil {
		t.Fatalf("creating requests dir: %v", err)
	}

	yamlContent := "" +
		"name: login\n" +
		"method: POST\n" +
		"path: /login\n" +
		"capture:\n" +
		"  TOKEN: data.token\n" +
		"  USER_ID: data.user.id\n"
	if err := os.WriteFile(filepath.Join("requests", "login.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("writing request file: %v", err)
	}

	got, err := Load("login")
	if err != nil {
		t.Fatalf("loading request failed: %v", err)
	}

	expected := map[string]string{
		"TOKEN":   "data.token",
		"USER_ID": "data.user.id",
	}
	if !reflect.DeepEqual(got.Capture, expected) {
		t.Fatalf("expected capture=%v, got %v", expected, got.Capture)
	}
}

func withTempDirAsWorkingDirRequest(t *testing.T) {
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
