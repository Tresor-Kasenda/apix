package detect

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/dotenv"
)

type Framework struct {
	Name        string
	Language    string
	DefaultPort int
	// APIPrefix is the path the framework mounts API routes under by default
	// (e.g. Laravel serves routes/api.php under /api). Empty for most stacks.
	APIPrefix string
}

type DetectedRoute struct {
	Method string
	Path   string
}

type Result struct {
	Framework *Framework
	// Dir is the directory (relative to the scanned root) where the framework
	// was found; "" means the root itself. Non-empty for monorepos such as
	// backend/, api/ or services/users/.
	Dir    string
	Routes []DetectedRoute
	// SpecPath is an OpenAPI/Swagger document found in the project, if any.
	SpecPath string
	// BaseURL is the suggested base URL, derived from .env files, framework
	// config or defaults.
	BaseURL string
}

const fallbackBaseURL = "http://localhost:8000"

// specCandidates lists where frameworks and tools usually write OpenAPI docs.
var specCandidates = []string{
	"openapi.yaml", "openapi.yml", "openapi.json",
	"swagger.yaml", "swagger.yml", "swagger.json",
	"docs/openapi.yaml", "docs/openapi.yml", "docs/openapi.json",
	"docs/swagger.yaml", "docs/swagger.json",
	"api/openapi.yaml", "api/openapi.yml", "api/openapi.json",
	"spec/openapi.yaml", "spec/openapi.json",
	"storage/api-docs/api-docs.json",          // Laravel L5-Swagger
	"public/docs/openapi.yaml",                // Laravel Scribe
	"storage/app/private/scribe/openapi.yaml", // Laravel Scribe (v4+)
	"src/main/resources/openapi.yaml",         // Spring / Quarkus
	"src/main/resources/static/openapi.yaml",
}

// Detect inspects root (and, for monorepos, its immediate sub-directories) to
// find the backend framework, its routes, an OpenAPI document and a sensible
// base URL. It never fails: an unknown stack simply yields an empty result
// with a generic base URL.
func Detect(root string) Result {
	res := Result{}

	fw, dir := locateFramework(root)
	res.Framework = fw
	res.Dir = dir

	appRoot := filepath.Join(root, dir)
	res.SpecPath = findSpec(appRoot)
	if res.SpecPath == "" && dir != "" {
		res.SpecPath = findSpec(root)
	}

	if fw != nil {
		res.Routes = scanRoutes(appRoot, scanConfigFor(fw))
	}
	res.BaseURL = suggestBaseURL(root, appRoot, fw)
	return res
}

func locateFramework(root string) (*Framework, string) {
	if fw := detectFramework(root); fw != nil {
		return fw, ""
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, ""
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() && !skipDirs[e.Name()] && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	// Prefer conventional backend folder names, then alphabetical order.
	sort.SliceStable(names, func(i, j int) bool {
		pi, pj := backendDirPriority(names[i]), backendDirPriority(names[j])
		if pi != pj {
			return pi < pj
		}
		return names[i] < names[j]
	})

	for _, name := range names {
		if fw := detectFramework(filepath.Join(root, name)); fw != nil {
			return fw, name
		}
	}
	return nil, ""
}

func backendDirPriority(name string) int {
	switch strings.ToLower(name) {
	case "backend", "api", "server":
		return 0
	case "app", "apps", "services", "src":
		return 1
	default:
		return 2
	}
}

func findSpec(dir string) string {
	for _, candidate := range specCandidates {
		path := filepath.Join(dir, candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// suggestBaseURL derives the base URL, from most to least specific:
// explicit URL keys in .env, APP_URL, a port key in .env, framework config
// files, then the framework's default port.
func suggestBaseURL(root, appRoot string, fw *Framework) string {
	prefix := ""
	port := 0
	if fw != nil {
		prefix = fw.APIPrefix
		port = fw.DefaultPort
	}

	vars := readDotEnv(appRoot)
	if appRoot != root {
		for k, v := range readDotEnv(root) {
			if _, ok := vars[k]; !ok {
				vars[k] = v
			}
		}
	}

	for _, key := range []string{"APIX_BASE_URL", "API_BASE_URL", "API_URL", "BASE_URL"} {
		if v := strings.TrimSpace(vars[key]); isHTTPURL(v) {
			return strings.TrimRight(v, "/")
		}
	}

	if appURL := strings.TrimSpace(vars["APP_URL"]); isHTTPURL(appURL) {
		if u, err := url.Parse(appURL); err == nil {
			// "http://localhost" (Laravel's default) is not where `artisan serve`
			// listens; keep the framework's dev port in that case.
			if u.Port() == "" && isLoopback(u.Hostname()) && port != 0 {
				u.Host = fmt.Sprintf("%s:%d", u.Hostname(), port)
			}
			return strings.TrimRight(u.String(), "/") + prefix
		}
	}

	for _, key := range []string{"PORT", "APP_PORT", "SERVER_PORT", "HTTP_PORT", "API_PORT"} {
		if p := strings.TrimSpace(vars[key]); isPort(p) {
			return fmt.Sprintf("http://localhost:%s%s", p, prefix)
		}
	}

	if p := springServerPort(appRoot); p != "" {
		return fmt.Sprintf("http://localhost:%s%s", p, prefix)
	}

	if port != 0 {
		return fmt.Sprintf("http://localhost:%d%s", port, prefix)
	}
	return fallbackBaseURL
}

func readDotEnv(dir string) map[string]string {
	vars := map[string]string{}
	for _, name := range []string{".env", ".env.local"} {
		values, err := dotenv.LoadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		for k, v := range values {
			vars[k] = v
		}
	}
	return vars
}

func springServerPort(dir string) string {
	content, err := readFileHead(filepath.Join(dir, "src/main/resources/application.properties"), 64*1024)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && strings.TrimSpace(key) == "server.port" && isPort(strings.TrimSpace(value)) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isHTTPURL(v string) bool {
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1"
}

func isPort(v string) bool {
	if v == "" || len(v) > 5 {
		return false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
