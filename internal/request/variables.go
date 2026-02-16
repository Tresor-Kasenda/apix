// Package request provides variable resolution and request collection management.
package request

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var varPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// ResolveVariables replaces all ${KEY} patterns in the input string with values
// from the provided variable map. Unresolved variables are left as-is.
func ResolveVariables(input string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(input, func(match string) string {
		key := varPattern.FindStringSubmatch(match)[1]
		if val, ok := vars[key]; ok {
			return val
		}
		return match
	})
}

// BuildVariableMap constructs a merged variable map from environment variables,
// a stored token, built-in variables, and flag overrides.
// Priority (highest wins): flagVars > builtins > envVars
func BuildVariableMap(envVars map[string]string, token string, flagVars map[string]string) map[string]string {
	vars := make(map[string]string)

	// Environment variables (lowest priority).
	for k, v := range envVars {
		vars[k] = v
	}

	// Token from config/auth.
	if token != "" {
		vars["TOKEN"] = token
	}

	// Built-in variables.
	vars["TIMESTAMP"] = strconv.FormatInt(time.Now().Unix(), 10)
	vars["UUID"] = generateUUID()
	vars["RANDOM"] = generateRandom()

	// Flag overrides (highest priority).
	for k, v := range flagVars {
		vars[k] = v
	}

	return vars
}

// generateUUID produces a version 4 UUID using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// generateRandom produces a random 8-character hex string using crypto/rand.
func generateRandom() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x", b)
}
