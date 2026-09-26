package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name      string
		files     map[string]string
		framework string
		dir       string
		baseURL   string
		minRoutes int
	}{
		{
			name: "laravel with default APP_URL keeps artisan port and /api prefix",
			files: map[string]string{
				"composer.json":  `{"require": {"laravel/framework": "^11.0"}}`,
				".env":           "APP_URL=http://localhost\n",
				"routes/api.php": "<?php\nRoute::get('/users', [UserController::class, 'index']);\nRoute::post('/login', LoginController::class);\n",
			},
			framework: "Laravel",
			baseURL:   "http://localhost:8000/api",
			minRoutes: 2,
		},
		{
			name: "laravel with a custom domain in APP_URL",
			files: map[string]string{
				"composer.json": `{"require": {"laravel/framework": "^11.0"}}`,
				".env":          "APP_URL=https://shop.test\n",
			},
			framework: "Laravel",
			baseURL:   "https://shop.test/api",
		},
		{
			name: "express port read from .env",
			files: map[string]string{
				"package.json": `{"dependencies": {"express": "^4"}}`,
				".env":         "PORT=4000\n",
				"src/app.js":   "router.get('/health', h)\napp.post('/items', h)\n",
			},
			framework: "Express.js",
			baseURL:   "http://localhost:4000",
			minRoutes: 2,
		},
		{
			name: "nestjs is not mistaken for express",
			files: map[string]string{
				"package.json": `{"dependencies": {"@nestjs/core": "^10", "express": "^4"}}`,
			},
			framework: "NestJS",
			baseURL:   "http://localhost:3000",
		},
		{
			name: "monorepo with a python backend folder",
			files: map[string]string{
				"frontend/package.json":    `{"dependencies": {"vue": "^3"}}`,
				"backend/requirements.txt": "fastapi==0.110\nuvicorn\n",
				"backend/main.py":          "@app.get(\"/items/{id}\")\ndef read(id): ...\n",
			},
			framework: "FastAPI",
			dir:       "backend",
			baseURL:   "http://localhost:8000",
			minRoutes: 1,
		},
		{
			name: "asp.net core detected through a csproj glob",
			files: map[string]string{
				"Api/Api.csproj": `<Project Sdk="Microsoft.NET.Sdk.Web"></Project>`,
				"Api/Program.cs": `app.MapGet("/todos", () => todos);`,
			},
			framework: "ASP.NET Core",
			dir:       "Api",
			baseURL:   "http://localhost:5000",
			minRoutes: 1,
		},
		{
			name: "explicit API_URL wins",
			files: map[string]string{
				"go.mod": "module x\n\nrequire github.com/gin-gonic/gin v1.9.0\n",
				".env":   "API_URL=https://api.example.com/v1/\nPORT=9000\n",
			},
			framework: "Gin",
			baseURL:   "https://api.example.com/v1",
		},
		{
			name: "spring server.port",
			files: map[string]string{
				"pom.xml": "<artifactId>spring-boot-starter-web</artifactId>",
				"src/main/resources/application.properties": "server.port=9090\n",
			},
			framework: "Spring Boot",
			baseURL:   "http://localhost:9090",
		},
		{
			name:    "unknown stack falls back to a generic URL",
			files:   map[string]string{"README.md": "hello"},
			baseURL: fallbackBaseURL,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tc.files)

			res := Detect(root)

			gotName := ""
			if res.Framework != nil {
				gotName = res.Framework.Name
			}
			if gotName != tc.framework {
				t.Fatalf("framework: want %q, got %q", tc.framework, gotName)
			}
			if res.Dir != tc.dir {
				t.Errorf("dir: want %q, got %q", tc.dir, res.Dir)
			}
			if res.BaseURL != tc.baseURL {
				t.Errorf("base URL: want %q, got %q", tc.baseURL, res.BaseURL)
			}
			if len(res.Routes) < tc.minRoutes {
				t.Errorf("routes: want at least %d, got %v", tc.minRoutes, res.Routes)
			}
		})
	}
}

func TestDetectFindsOpenAPISpec(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"requirements.txt":  "django\n",
		"manage.py":         "",
		"docs/openapi.yaml": "openapi: 3.0.0\n",
	})

	res := Detect(root)
	if want := filepath.Join(root, "docs/openapi.yaml"); res.SpecPath != want {
		t.Fatalf("spec path: want %q, got %q", want, res.SpecPath)
	}
}

func TestNormalizePathParams(t *testing.T) {
	cases := map[string]string{
		"/users/{id}":            "/users/${id}",
		"/users/{id?}":           "/users/${id}",
		"/users/{id:\\d+}/posts": "/users/${id}/posts",
		"/users/:id/posts/:pid":  "/users/${id}/posts/${pid}",
		"users/<int:pk>/":        "users/${pk}/",
		"/plain/path":            "/plain/path",
		"/time/12:30":            "/time/12:30",
	}
	for in, want := range cases {
		if got := NormalizePathParams(in); got != want {
			t.Errorf("NormalizePathParams(%q) = %q, want %q", in, got, want)
		}
	}
}
