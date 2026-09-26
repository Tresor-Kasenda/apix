package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tresor-Kasend/apix/internal/config"
	"github.com/Tresor-Kasend/apix/internal/env"
	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestInitNonInteractiveUsesDetectedDefaults(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	mustWrite(t, "composer.json", `{"require": {"laravel/framework": "^11.0"}}`)
	mustWrite(t, ".env", "APP_URL=http://localhost\n")
	mustWrite(t, "routes/api.php", "<?php\nRoute::get('/users', fn () => []);\n")

	if err := runInit(initOptions{yes: true, name: "shop"}); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	// env/dev.yaml must inherit the detected base URL instead of overriding it.
	if cfg.BaseURL != "http://localhost:8000/api" {
		t.Fatalf("unexpected base URL %q", cfg.BaseURL)
	}
	if cfg.Project != "shop" || cfg.Auth.TokenPath != config.DefaultTokenPath {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if _, err := request.Load("get-users"); err != nil {
		t.Fatalf("detected route was not saved: %v", err)
	}
}

func TestInitImportsDetectedOpenAPISpec(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	mustWrite(t, "openapi.yaml", `openapi: 3.0.0
info: {title: Demo, version: "1"}
servers: [{url: /v1}]
paths:
  /pets/{petId}:
    get:
      operationId: showPet
`)

	if err := runInit(initOptions{yes: true, baseURL: "http://localhost:9000", auth: "none"}); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	saved, err := request.Load("showpet")
	if err != nil {
		t.Fatalf("spec operation was not imported: %v", err)
	}
	if saved.Path != "/v1/pets/${petId}" {
		t.Fatalf("expected base path to be prefixed, got %q", saved.Path)
	}
}

func TestInitRejectsInvalidAuthFlag(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)
	if err := runInit(initOptions{yes: true, auth: "oauth-magic"}); err == nil {
		t.Fatal("expected invalid auth type error")
	}
}

func TestCommandsWorkFromSubdirectoryWithDotEnvAndOSVariables(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	var gotPath, gotKey, gotTenant string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-API-Key")
		gotTenant = r.Header.Get("X-Tenant")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	mustWrite(t, "apix.yaml", `project: universal
base_url: ${API_HOST}/${API_VERSION:-v1}
current_env: dev
auth:
  type: api_key
  api_key: ${SECRET_KEY}
`)
	mustWrite(t, "env/dev.yaml", "variables: {}\n")
	mustWrite(t, ".env", "API_HOST="+server.URL+"\nSECRET_KEY=from-dotenv\n")
	mustWrite(t, "requests/ping.yaml", "name: ping\nmethod: GET\npath: /ping\nheaders:\n  X-Tenant: ${TENANT}\n")

	// The OS environment wins over .env, like in every dotenv implementation.
	t.Setenv("SECRET_KEY", "from-os")
	t.Setenv("TENANT", "acme")

	if err := os.MkdirAll(filepath.Join("src", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join("src", "deep")); err != nil {
		t.Fatal(err)
	}

	if err := executeSavedRequest("ping", ExecuteOptions{Silent: true, SuppressOutput: true}); err != nil {
		t.Fatalf("running saved request from a subdirectory: %v", err)
	}
	if gotPath != "/v1/ping" || gotKey != "from-os" || gotTenant != "acme" {
		t.Fatalf("unexpected request: path=%q key=%q tenant=%q", gotPath, gotKey, gotTenant)
	}

	// State files land in the project root, not in the current sub-directory.
	if _, err := os.Stat(filepath.Join("..", "..", ".apix", "last_request.yaml")); err != nil {
		t.Fatalf("expected state in project root: %v", err)
	}
	if _, err := os.Stat(".apix"); !os.IsNotExist(err) {
		t.Fatalf("no .apix directory should be created in the sub-directory")
	}
}

func TestRuntimeEnvironmentOverrides(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	mustWrite(t, "apix.yaml", "project: x\nbase_url: http://from-config\ncurrent_env: dev\n")
	mustWrite(t, "env/dev.yaml", "base_url: http://dev\n")
	mustWrite(t, "env/ci.yaml", "base_url: http://ci\n")

	t.Setenv(config.EnvVarEnvironment, "ci")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://ci" || cfg.CurrentEnv != "ci" {
		t.Fatalf("APIX_ENV not applied: %+v", cfg)
	}

	t.Setenv(config.EnvVarBaseURL, "http://override")
	cfg, err = config.LoadWithEnvOverride("dev")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "http://override" {
		t.Fatalf("APIX_BASE_URL not applied: %q", cfg.BaseURL)
	}

	t.Setenv(config.EnvVarBaseURL, "")
	t.Setenv(config.EnvVarEnvironment, "missing")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected an error for an unknown APIX_ENV")
	}
}

func TestCustomDirectoriesAndCommentPreservation(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	mustWrite(t, "apix.yaml", "# keep me\nproject: x\nrequests_dir: api/http\nenv_dir: api/envs\ncurrent_env: dev\n")
	if err := env.Create("staging"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("api", "envs", "staging.yaml")); err != nil {
		t.Fatalf("env not created in env_dir: %v", err)
	}
	if err := request.Save("hello", request.SavedRequest{Method: "GET", Path: "/"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("api", "http", "hello.yaml")); err != nil {
		t.Fatalf("request not saved in requests_dir: %v", err)
	}

	if err := env.SetActive("staging"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile("apix.yaml")
	if !strings.Contains(string(data), "# keep me") || !strings.Contains(string(data), "current_env: staging") {
		t.Fatalf("apix.yaml comments lost or env not switched:\n%s", data)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
