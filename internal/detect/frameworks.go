package detect

import (
	"os"
	"path/filepath"
	"strings"
)

type contentCheck struct {
	files      []string // files to check for content (relative to root)
	substrings []string // any of these must appear (case-insensitive)
}

type frameworkRule struct {
	framework    Framework
	markerFiles  []string      // at least one must exist
	contentCheck *contentCheck // optional content check
}

var rules = []frameworkRule{
	// Python
	{
		framework:   Framework{Name: "Django", Language: "python", DefaultPort: 8000},
		markerFiles: []string{"manage.py"},
		contentCheck: &contentCheck{
			files:      []string{"requirements.txt", "pyproject.toml", "Pipfile", "setup.py"},
			substrings: []string{"django"},
		},
	},
	{
		framework:   Framework{Name: "FastAPI", Language: "python", DefaultPort: 8000},
		markerFiles: []string{"requirements.txt", "pyproject.toml", "Pipfile"},
		contentCheck: &contentCheck{
			files:      []string{"requirements.txt", "pyproject.toml", "Pipfile"},
			substrings: []string{"fastapi"},
		},
	},
	{
		framework:   Framework{Name: "Flask", Language: "python", DefaultPort: 5000},
		markerFiles: []string{"requirements.txt", "pyproject.toml", "Pipfile"},
		contentCheck: &contentCheck{
			files:      []string{"requirements.txt", "pyproject.toml", "Pipfile"},
			substrings: []string{"flask"},
		},
	},
	// JavaScript / TypeScript
	{
		framework:   Framework{Name: "NestJS", Language: "javascript", DefaultPort: 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"@nestjs/core"`},
		},
	},
	{
		framework:   Framework{Name: "AdonisJS", Language: "javascript", DefaultPort: 3333},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"@adonisjs/core"`},
		},
	},
	{
		framework:   Framework{Name: "Hono", Language: "javascript", DefaultPort: 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"hono"`},
		},
	},
	{
		framework:   Framework{Name: "Koa", Language: "javascript", DefaultPort: 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"koa"`, `"@koa/router"`},
		},
	},
	{
		framework:   Framework{Name: "Express.js", Language: "javascript", DefaultPort: 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"express"`},
		},
	},
	{
		framework:   Framework{Name: "Fastify", Language: "javascript", DefaultPort: 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"fastify"`},
		},
	},
	// Go
	{
		framework:   Framework{Name: "Gin", Language: "go", DefaultPort: 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"gin-gonic/gin"},
		},
	},
	{
		framework:   Framework{Name: "Chi", Language: "go", DefaultPort: 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"go-chi/chi"},
		},
	},
	{
		framework:   Framework{Name: "Echo", Language: "go", DefaultPort: 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"labstack/echo"},
		},
	},
	{
		framework:   Framework{Name: "Fiber", Language: "go", DefaultPort: 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"gofiber/fiber"},
		},
	},
	{
		framework:   Framework{Name: "Gorilla Mux", Language: "go", DefaultPort: 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"gorilla/mux"},
		},
	},
	// Rust
	{
		framework:   Framework{Name: "Actix", Language: "rust", DefaultPort: 8080},
		markerFiles: []string{"Cargo.toml"},
		contentCheck: &contentCheck{
			files:      []string{"Cargo.toml"},
			substrings: []string{"actix-web"},
		},
	},
	{
		framework:   Framework{Name: "Axum", Language: "rust", DefaultPort: 8080},
		markerFiles: []string{"Cargo.toml"},
		contentCheck: &contentCheck{
			files:      []string{"Cargo.toml"},
			substrings: []string{"axum"},
		},
	},
	{
		framework:   Framework{Name: "Rocket", Language: "rust", DefaultPort: 8080},
		markerFiles: []string{"Cargo.toml"},
		contentCheck: &contentCheck{
			files:      []string{"Cargo.toml"},
			substrings: []string{"rocket"},
		},
	},
	// Java
	{
		framework:   Framework{Name: "Spring Boot", Language: "java", DefaultPort: 8080},
		markerFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		contentCheck: &contentCheck{
			files:      []string{"pom.xml", "build.gradle", "build.gradle.kts"},
			substrings: []string{"spring-boot", "org.springframework.boot"},
		},
	},
	{
		framework:   Framework{Name: "Quarkus", Language: "java", DefaultPort: 8080},
		markerFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		contentCheck: &contentCheck{
			files:      []string{"pom.xml", "build.gradle", "build.gradle.kts"},
			substrings: []string{"io.quarkus"},
		},
	},
	{
		framework:   Framework{Name: "Micronaut", Language: "java", DefaultPort: 8080},
		markerFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		contentCheck: &contentCheck{
			files:      []string{"pom.xml", "build.gradle", "build.gradle.kts"},
			substrings: []string{"io.micronaut"},
		},
	},
	// Kotlin
	{
		framework:   Framework{Name: "Ktor", Language: "kotlin", DefaultPort: 8080},
		markerFiles: []string{"build.gradle.kts", "build.gradle", "pom.xml"},
		contentCheck: &contentCheck{
			files:      []string{"build.gradle.kts", "build.gradle", "pom.xml"},
			substrings: []string{"io.ktor"},
		},
	},
	// C# / .NET
	{
		framework:   Framework{Name: "ASP.NET Core", Language: "csharp", DefaultPort: 5000},
		markerFiles: []string{"*.csproj"},
		contentCheck: &contentCheck{
			files:      []string{"*.csproj"},
			substrings: []string{"Microsoft.NET.Sdk.Web", "Microsoft.AspNetCore"},
		},
	},
	// Elixir
	{
		framework:   Framework{Name: "Phoenix", Language: "elixir", DefaultPort: 4000},
		markerFiles: []string{"mix.exs"},
		contentCheck: &contentCheck{
			files:      []string{"mix.exs"},
			substrings: []string{":phoenix"},
		},
	},
	// Swift
	{
		framework:   Framework{Name: "Vapor", Language: "swift", DefaultPort: 8080},
		markerFiles: []string{"Package.swift"},
		contentCheck: &contentCheck{
			files:      []string{"Package.swift"},
			substrings: []string{"vapor/vapor"},
		},
	},
	// Ruby
	{
		framework:   Framework{Name: "Rails", Language: "ruby", DefaultPort: 3000},
		markerFiles: []string{"Gemfile"},
		contentCheck: &contentCheck{
			files:      []string{"Gemfile"},
			substrings: []string{"rails"},
		},
	},
	{
		framework:   Framework{Name: "Sinatra", Language: "ruby", DefaultPort: 4567},
		markerFiles: []string{"Gemfile"},
		contentCheck: &contentCheck{
			files:      []string{"Gemfile"},
			substrings: []string{"sinatra"},
		},
	},
	// PHP
	{
		framework:   Framework{Name: "Laravel", Language: "php", DefaultPort: 8000, APIPrefix: "/api"},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"laravel/framework", "laravel/lumen"},
		},
	},
	{
		framework:   Framework{Name: "Symfony", Language: "php", DefaultPort: 8000},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"symfony/framework-bundle", "symfony/routing"},
		},
	},
	{
		framework:   Framework{Name: "Slim", Language: "php", DefaultPort: 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"slim/slim"},
		},
	},
	{
		framework:   Framework{Name: "CakePHP", Language: "php", DefaultPort: 8765},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"cakephp/cakephp"},
		},
	},
	{
		framework:   Framework{Name: "CodeIgniter", Language: "php", DefaultPort: 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"codeigniter4/framework", "codeigniter4/appstarter"},
		},
	},
	{
		framework:   Framework{Name: "Yii", Language: "php", DefaultPort: 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"yiisoft/yii2", "yiisoft/yii-runner"},
		},
	},
	{
		framework:   Framework{Name: "Laminas", Language: "php", DefaultPort: 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"laminas/laminas-mvc", "laminas/laminas-mezzio"},
		},
	},
	// Generic fallbacks, evaluated last.
	{
		framework:   Framework{Name: "Go net/http", Language: "go", DefaultPort: 8080},
		markerFiles: []string{"go.mod"},
	},
}

func detectFramework(root string) *Framework {
	for _, rule := range rules {
		if matchesRule(root, rule) {
			fw := rule.framework
			return &fw
		}
	}
	return nil
}

func matchesRule(root string, rule frameworkRule) bool {
	if len(expandFiles(root, rule.markerFiles)) == 0 {
		return false
	}
	if rule.contentCheck == nil {
		return true
	}
	return checkContent(root, rule.contentCheck)
}

func checkContent(root string, cc *contentCheck) bool {
	for _, path := range expandFiles(root, cc.files) {
		content, err := readFileHead(path, 64*1024)
		if err != nil {
			continue
		}
		lower := strings.ToLower(content)
		for _, sub := range cc.substrings {
			if strings.Contains(lower, strings.ToLower(sub)) {
				return true
			}
		}
	}
	return false
}

// expandFiles resolves file names (which may contain glob patterns such as
// "*.csproj") relative to root and returns the ones that exist.
func expandFiles(root string, patterns []string) []string {
	var found []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil && !info.IsDir() {
				found = append(found, m)
			}
		}
	}
	return found
}

func readFileHead(path string, maxBytes int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, maxBytes)
	n, _ := f.Read(buf)
	return string(buf[:n]), nil
}
