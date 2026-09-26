// Package dotenv reads .env files as written by most frameworks
// (Laravel, Django, Rails, Node, Symfony, ...).
//
// Supported syntax:
//
//	# comment
//	KEY=value
//	export KEY=value
//	KEY="double quoted, supports \n escapes and ${OTHER} interpolation"
//	KEY='single quoted, taken literally'
//	KEY=value # inline comment
//	KEY=${OTHER}/suffix
package dotenv

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	keyPattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)
	interpPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
)

// LoadFile parses the dotenv file at path.
func LoadFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads dotenv content. Invalid lines are ignored rather than rejected so
// that a slightly unusual .env never prevents apix from running.
func Parse(r io.Reader) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			continue
		}

		value, interpolate := parseValue(strings.TrimSpace(rawValue))
		if interpolate {
			value = expand(value, values)
		}
		values[key] = value
	}
	return values, scanner.Err()
}

// parseValue strips quotes and inline comments. It reports whether ${VAR}
// interpolation applies (everything except single-quoted values).
func parseValue(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}

	switch raw[0] {
	case '\'':
		if end := strings.Index(raw[1:], "'"); end >= 0 {
			return raw[1 : end+1], false
		}
		return strings.TrimPrefix(raw, "'"), false
	case '"':
		if end := closingQuote(raw); end > 0 {
			return unescape(raw[1:end]), true
		}
		return strings.TrimPrefix(raw, `"`), true
	}

	if idx := strings.Index(raw, " #"); idx >= 0 {
		raw = raw[:idx]
	}
	return strings.TrimSpace(raw), true
}

func closingQuote(raw string) int {
	for i := 1; i < len(raw); i++ {
		switch raw[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}

func unescape(s string) string {
	return strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t", `\"`, `"`, `\\`, `\`).Replace(s)
}

func expand(value string, known map[string]string) string {
	return interpPattern.ReplaceAllStringFunc(value, func(match string) string {
		name := interpPattern.FindStringSubmatch(match)[1]
		if v, ok := known[name]; ok {
			return v
		}
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return ""
	})
}
