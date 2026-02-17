package detect

import "regexp"

type scanConfig struct {
	fileExts []string         // file extensions to scan (e.g. ".py", ".go")
	files    []string         // specific files to scan (e.g. "routes/api.php")
	patterns []*regexp.Regexp // regexes with "method" and "path" named groups
}

// Django
var djangoPatterns = []*regexp.Regexp{
	regexp.MustCompile(`path\(\s*['"](?P<path>[^'"]+)['"]`),
	regexp.MustCompile(`url\(\s*r?['"](?P<path>[^'"]+)['"]`),
}

// FastAPI
var fastapiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`@(?:\w+)\.(?P<method>get|post|put|patch|delete)\(\s*["'](?P<path>[^"']+)["']`),
}

// Flask
var flaskPatterns = []*regexp.Regexp{
	regexp.MustCompile(`@(?:\w+)\.route\(\s*["'](?P<path>[^"']+)["']`),
	regexp.MustCompile(`@(?:\w+)\.(?P<method>get|post|put|patch|delete)\(\s*["'](?P<path>[^"']+)["']`),
}

// Express.js / Fastify
var expressPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:app|router|server)\s*\.(?P<method>get|post|put|patch|delete)\(\s*["'](?P<path>[^"']+)["']`),
}

// NestJS
var nestPatterns = []*regexp.Regexp{
	regexp.MustCompile(`@(?P<method>Get|Post|Put|Patch|Delete)\(\s*["'](?P<path>[^"']+)["']`),
}

// Gin
var ginPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.(?P<method>GET|POST|PUT|PATCH|DELETE)\(\s*"(?P<path>[^"]+)"`),
}

// Chi
var chiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.(?P<method>Get|Post|Put|Patch|Delete)\(\s*"(?P<path>[^"]+)"`),
}

// Echo
var echoPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.(?P<method>GET|POST|PUT|PATCH|DELETE)\(\s*"(?P<path>[^"]+)"`),
}

// Fiber
var fiberPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.(?P<method>Get|Post|Put|Patch|Delete)\(\s*"(?P<path>[^"]+)"`),
}

// Laravel
var laravelPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Route::(?P<method>get|post|put|patch|delete)\(\s*['"](?P<path>[^'"]+)['"]`),
}

// Rails
var railsPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?P<method>get|post|put|patch|delete)\s+['"](?P<path>[^'"]+)['"]`),
}

// Spring Boot
var springPatterns = []*regexp.Regexp{
	regexp.MustCompile(`@(?P<method>Get|Post|Put|Patch|Delete)Mapping\(\s*(?:value\s*=\s*)?["'](?P<path>[^"']+)["']`),
	regexp.MustCompile(`@RequestMapping\(\s*(?:value\s*=\s*)?["'](?P<path>[^"']+)["']`),
}

// Actix / Rocket (Rust attribute macros)
var rustAttrPatterns = []*regexp.Regexp{
	regexp.MustCompile(`#\[(?P<method>get|post|put|patch|delete)\(\s*"(?P<path>[^"]+)"`),
}

// Axum
var axumPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.route\(\s*"(?P<path>[^"]+)"\s*,\s*(?P<method>get|post|put|patch|delete)`),
}

func scanConfigFor(fw *Framework) scanConfig {
	switch fw.Name {
	case "Django":
		return scanConfig{fileExts: []string{".py"}, files: []string{"urls.py"}, patterns: djangoPatterns}
	case "FastAPI":
		return scanConfig{fileExts: []string{".py"}, patterns: fastapiPatterns}
	case "Flask":
		return scanConfig{fileExts: []string{".py"}, patterns: flaskPatterns}
	case "Express.js", "Fastify":
		return scanConfig{fileExts: []string{".js", ".ts", ".mjs"}, patterns: expressPatterns}
	case "NestJS":
		return scanConfig{fileExts: []string{".ts"}, patterns: nestPatterns}
	case "Gin":
		return scanConfig{fileExts: []string{".go"}, patterns: ginPatterns}
	case "Chi":
		return scanConfig{fileExts: []string{".go"}, patterns: chiPatterns}
	case "Echo":
		return scanConfig{fileExts: []string{".go"}, patterns: echoPatterns}
	case "Fiber":
		return scanConfig{fileExts: []string{".go"}, patterns: fiberPatterns}
	case "Laravel":
		return scanConfig{files: []string{"routes/api.php", "routes/web.php"}, patterns: laravelPatterns}
	case "Rails":
		return scanConfig{files: []string{"config/routes.rb"}, patterns: railsPatterns}
	case "Spring Boot":
		return scanConfig{fileExts: []string{".java", ".kt"}, patterns: springPatterns}
	case "Actix", "Rocket":
		return scanConfig{fileExts: []string{".rs"}, patterns: rustAttrPatterns}
	case "Axum":
		return scanConfig{fileExts: []string{".rs"}, patterns: axumPatterns}
	default:
		return scanConfig{}
	}
}
