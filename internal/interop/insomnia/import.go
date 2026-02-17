package insomnia

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/request"
)

type exportData struct {
	Resources []resource `json:"resources"`
}

type resource struct {
	Type       string            `json:"_type"`
	Name       string            `json:"name"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    []headerField     `json:"headers"`
	Parameters []parameterField  `json:"parameters"`
	Body       *insomniaBody     `json:"body"`
	BodyText   string            `json:"body_text"`
	RawBody    string            `json:"rawBody"`
	Metadata   map[string]string `json:"metadata"`
}

type headerField struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type parameterField struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type insomniaBody struct {
	Text string `json:"text"`
}

func ParseExportFile(filePath string) ([]request.SavedRequest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading insomnia file %q: %w", filePath, err)
	}
	return ParseExport(data)
}

func ParseExport(data []byte) ([]request.SavedRequest, error) {
	var parsed exportData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parsing insomnia export: %w", err)
	}

	out := make([]request.SavedRequest, 0)
	for _, res := range parsed.Resources {
		if strings.TrimSpace(res.Type) != "request" {
			continue
		}

		pathValue, query := splitPathAndQuery(res.URL)
		if query == nil {
			query = make(map[string]string)
		}
		for _, p := range res.Parameters {
			if p.Disabled || strings.TrimSpace(p.Name) == "" {
				continue
			}
			query[strings.TrimSpace(p.Name)] = p.Value
		}
		if len(query) == 0 {
			query = nil
		}

		headers := make(map[string]string)
		for _, h := range res.Headers {
			if h.Disabled || strings.TrimSpace(h.Name) == "" {
				continue
			}
			headers[strings.TrimSpace(h.Name)] = h.Value
		}

		body := strings.TrimSpace(bodyFromResource(res))
		method := strings.ToUpper(strings.TrimSpace(res.Method))
		if method == "" {
			method = "GET"
		}

		out = append(out, request.SavedRequest{
			Name:    strings.TrimSpace(res.Name),
			Method:  method,
			Path:    pathValue,
			Headers: headers,
			Query:   query,
			Body:    body,
		})
	}

	return out, nil
}

func bodyFromResource(res resource) string {
	if res.Body != nil && strings.TrimSpace(res.Body.Text) != "" {
		return res.Body.Text
	}
	if strings.TrimSpace(res.BodyText) != "" {
		return res.BodyText
	}
	if strings.TrimSpace(res.RawBody) != "" {
		return res.RawBody
	}
	return ""
}

func splitPathAndQuery(value string) (string, map[string]string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/", nil
	}

	parsed, err := url.Parse(value)
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
