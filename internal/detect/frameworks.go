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
		framework:   Framework{"Django", "python", 8000},
		markerFiles: []string{"manage.py"},
		contentCheck: &contentCheck{
			files:      []string{"requirements.txt", "pyproject.toml", "Pipfile", "setup.py"},
			substrings: []string{"django"},
		},
	},
	{
		framework:   Framework{"FastAPI", "python", 8000},
		markerFiles: []string{"requirements.txt", "pyproject.toml", "Pipfile"},
		contentCheck: &contentCheck{
			files:      []string{"requirements.txt", "pyproject.toml", "Pipfile"},
			substrings: []string{"fastapi"},
		},
	},
	{
		framework:   Framework{"Flask", "python", 5000},
		markerFiles: []string{"requirements.txt", "pyproject.toml", "Pipfile"},
		contentCheck: &contentCheck{
			files:      []string{"requirements.txt", "pyproject.toml", "Pipfile"},
			substrings: []string{"flask"},
		},
	},
	// JavaScript / TypeScript
	{
		framework:   Framework{"Express.js", "javascript", 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"express"`},
		},
	},
	{
		framework:   Framework{"Fastify", "javascript", 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"fastify"`},
		},
	},
	{
		framework:   Framework{"NestJS", "javascript", 3000},
		markerFiles: []string{"package.json"},
		contentCheck: &contentCheck{
			files:      []string{"package.json"},
			substrings: []string{`"@nestjs/core"`},
		},
	},
	// Go
	{
		framework:   Framework{"Gin", "go", 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"gin-gonic/gin"},
		},
	},
	{
		framework:   Framework{"Chi", "go", 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"go-chi/chi"},
		},
	},
	{
		framework:   Framework{"Echo", "go", 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"labstack/echo"},
		},
	},
	{
		framework:   Framework{"Fiber", "go", 8080},
		markerFiles: []string{"go.mod"},
		contentCheck: &contentCheck{
			files:      []string{"go.mod"},
			substrings: []string{"gofiber/fiber"},
		},
	},
	// Rust
	{
		framework:   Framework{"Actix", "rust", 8080},
		markerFiles: []string{"Cargo.toml"},
		contentCheck: &contentCheck{
			files:      []string{"Cargo.toml"},
			substrings: []string{"actix-web"},
		},
	},
	{
		framework:   Framework{"Axum", "rust", 8080},
		markerFiles: []string{"Cargo.toml"},
		contentCheck: &contentCheck{
			files:      []string{"Cargo.toml"},
			substrings: []string{"axum"},
		},
	},
	{
		framework:   Framework{"Rocket", "rust", 8080},
		markerFiles: []string{"Cargo.toml"},
		contentCheck: &contentCheck{
			files:      []string{"Cargo.toml"},
			substrings: []string{"rocket"},
		},
	},
	// Java
	{
		framework:   Framework{"Spring Boot", "java", 8080},
		markerFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		contentCheck: &contentCheck{
			files:      []string{"pom.xml", "build.gradle", "build.gradle.kts"},
			substrings: []string{"spring-boot", "org.springframework.boot"},
		},
	},
	// Ruby
	{
		framework:   Framework{"Rails", "ruby", 3000},
		markerFiles: []string{"Gemfile"},
		contentCheck: &contentCheck{
			files:      []string{"Gemfile"},
			substrings: []string{"rails"},
		},
	},
	// PHP
	{
		framework:   Framework{"Laravel", "php", 8000},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"laravel/framework", "laravel/lumen"},
		},
	},
	{
		framework:   Framework{"Symfony", "php", 8000},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"symfony/framework-bundle", "symfony/routing"},
		},
	},
	{
		framework:   Framework{"Slim", "php", 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"slim/slim"},
		},
	},
	{
		framework:   Framework{"CakePHP", "php", 8765},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"cakephp/cakephp"},
		},
	},
	{
		framework:   Framework{"CodeIgniter", "php", 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"codeigniter4/framework", "codeigniter4/appstarter"},
		},
	},
	{
		framework:   Framework{"Yii", "php", 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"yiisoft/yii2", "yiisoft/yii-runner"},
		},
	},
	{
		framework:   Framework{"Laminas", "php", 8080},
		markerFiles: []string{"composer.json"},
		contentCheck: &contentCheck{
			files:      []string{"composer.json"},
			substrings: []string{"laminas/laminas-mvc", "laminas/laminas-mezzio"},
		},
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
	markerFound := false
	for _, mf := range rule.markerFiles {
		path := filepath.Join(root, mf)
		if _, err := os.Stat(path); err == nil {
			markerFound = true
			break
		}
	}
	if !markerFound {
		return false
	}
	if rule.contentCheck == nil {
		return true
	}
	return checkContent(root, rule.contentCheck)
}

func checkContent(root string, cc *contentCheck) bool {
	for _, f := range cc.files {
		path := filepath.Join(root, f)
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
