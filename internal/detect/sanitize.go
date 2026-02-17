package detect

import (
	"regexp"
	"strings"
)

var (
	paramPattern    = regexp.MustCompile(`[:{<](\w+)[}>]?`)
	nonAlphaPattern = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

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
