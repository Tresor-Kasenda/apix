package detect

import (
	"regexp"
	"strings"
)

var (
	paramPattern    = regexp.MustCompile(`[:{<](\w+)[}>]?`)
	nonAlphaPattern = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	bracePathParam = regexp.MustCompile(`\{(\w+)[^}]*\}`)   // {id}, {id?}, {id:\d+}
	anglePathParam = regexp.MustCompile(`<(?:\w+:)?(\w+)>`) // <id>, <int:id>
	colonPathParam = regexp.MustCompile(`(^|/):(\w+)`)      // :id
)

// NormalizePathParams rewrites framework-specific path parameters into apix
// variables so detected routes are directly runnable with --var:
// "/users/{id}", "/users/:id" and "/users/<int:id>" all become "/users/${id}".
func NormalizePathParams(path string) string {
	path = bracePathParam.ReplaceAllString(path, "$${$1}")
	path = anglePathParam.ReplaceAllString(path, "$${$1}")
	return colonPathParam.ReplaceAllString(path, "$1$${$2}")
}

// SanitizeName converts a method and path into a safe filename.
// Example: "GET", "/users/:id" -> "get-users-id"
func SanitizeName(method, path string) string {
	method = strings.ToLower(method)

	path = strings.Trim(path, "/")

	// Replace path param markers: :id, {id}, <id>, <int:id> -> id
	path = paramPattern.ReplaceAllString(path, "$1")

	// Replace non-alphanumeric chars with dashes.
	path = nonAlphaPattern.ReplaceAllString(path, "-")

	path = strings.Trim(path, "-")
	path = strings.ToLower(path)

	if path == "" {
		return method + "-root"
	}
	return method + "-" + path
}
