package curl

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/request"
)

func ParseCommand(command string) (*request.SavedRequest, error) {
	tokens, err := tokenize(command)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty curl command")
	}
	if strings.ToLower(tokens[0]) == "curl" {
		tokens = tokens[1:]
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("curl command has no arguments")
	}

	method := "GET"
	methodExplicit := false
	headers := make(map[string]string)
	var bodyParts []string
	var targetURL string

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok {
		case "-X", "--request":
			i++
			if i >= len(tokens) {
				return nil, fmt.Errorf("%s requires a value", tok)
			}
			method = strings.ToUpper(strings.TrimSpace(tokens[i]))
			methodExplicit = true
		case "-H", "--header":
			i++
			if i >= len(tokens) {
				return nil, fmt.Errorf("%s requires a value", tok)
			}
			key, value, ok := parseHeader(tokens[i])
			if !ok {
				return nil, fmt.Errorf("invalid header %q (expected key:value)", tokens[i])
			}
			headers[key] = value
		case "-d", "--data", "--data-raw", "--data-binary", "--data-urlencode":
			i++
			if i >= len(tokens) {
				return nil, fmt.Errorf("%s requires a value", tok)
			}
			bodyParts = append(bodyParts, tokens[i])
		case "-G", "--get":
			method = "GET"
			methodExplicit = true
		default:
			if strings.HasPrefix(tok, "-") {
				// Unsupported flag in v1 mapping: ignore.
				continue
			}
			targetURL = tok
		}
	}

	if targetURL == "" {
		return nil, fmt.Errorf("curl command does not include a URL")
	}
	if len(bodyParts) > 0 && !methodExplicit {
		method = "POST"
	}

	pathValue, query := splitPathAndQuery(targetURL)

	return &request.SavedRequest{
		Method:  method,
		Path:    pathValue,
		Headers: headers,
		Query:   query,
		Body:    strings.Join(bodyParts, "&"),
	}, nil
}

func parseHeader(value string) (string, string, bool) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(parts[1]), true
}

func splitPathAndQuery(value string) (string, map[string]string) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return value, nil
	}

	query := make(map[string]string)
	for key, values := range parsed.Query() {
		if len(values) == 0 {
			query[key] = ""
			continue
		}
		query[key] = values[len(values)-1]
	}
	if len(query) == 0 {
		query = nil
	}

	if parsed.IsAbs() {
		pathValue := parsed.Scheme + "://" + parsed.Host + parsed.EscapedPath()
		if parsed.RawPath != "" {
			pathValue = parsed.Scheme + "://" + parsed.Host + parsed.RawPath
		}
		if parsed.Path == "" && parsed.RawPath == "" {
			pathValue += "/"
		}
		return pathValue, query
	}
	if parsed.Path == "" {
		return value, query
	}
	if strings.HasPrefix(parsed.Path, "/") {
		return parsed.Path, query
	}
	return "/" + parsed.Path, query
}

func tokenize(command string) ([]string, error) {
	out := make([]string, 0)
	var current strings.Builder
	var quote rune
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		out = append(out, current.String())
		current.Reset()
	}

	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}

		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if r == '\'' || r == '"' {
			quote = r
			continue
		}

		if r == ' ' || r == '\t' || r == '\n' {
			flush()
			continue
		}

		current.WriteRune(r)
	}

	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in curl command")
	}

	flush()
	return out, nil
}
