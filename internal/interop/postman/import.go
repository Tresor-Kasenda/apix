package postman

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/request"
)

type collection struct {
	Info postmanInfo `json:"info"`
	Item []item      `json:"item"`
}

type postmanInfo struct {
	Name string `json:"name"`
}

type item struct {
	Name    string        `json:"name"`
	Item    []item        `json:"item"`
	Request requestObject `json:"request"`
}

type requestObject struct {
	Method string          `json:"method"`
	Header []header        `json:"header"`
	URL    json.RawMessage `json:"url"`
	Body   bodyObject      `json:"body"`
}

type header struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type bodyObject struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

type urlObject struct {
	Raw   string       `json:"raw"`
	Path  []string     `json:"path"`
	Query []queryParam `json:"query"`
}

type queryParam struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

func ParseCollectionFile(filePath string) ([]request.SavedRequest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading postman file %q: %w", filePath, err)
	}
	return ParseCollection(data)
}

func ParseCollection(data []byte) ([]request.SavedRequest, error) {
	var c collection
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing postman collection: %w", err)
	}

	out := make([]request.SavedRequest, 0)
	for _, it := range c.Item {
		flattenItems(it, "", &out)
	}
	return out, nil
}

func flattenItems(it item, prefix string, out *[]request.SavedRequest) {
	fullName := strings.TrimSpace(it.Name)
	if prefix != "" && fullName != "" {
		fullName = prefix + " " + fullName
	}
	if fullName == "" {
		fullName = strings.TrimSpace(prefix)
	}

	if len(it.Item) > 0 {
		for _, child := range it.Item {
			flattenItems(child, fullName, out)
		}
		return
	}

	if strings.TrimSpace(it.Request.Method) == "" {
		return
	}

	pathValue, query := parseRequestURL(it.Request.URL)
	headers := make(map[string]string)
	for _, h := range it.Request.Header {
		if h.Disabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		headers[strings.TrimSpace(h.Key)] = h.Value
	}

	method := strings.ToUpper(strings.TrimSpace(it.Request.Method))
	req := request.SavedRequest{
		Name:    fullName,
		Method:  method,
		Path:    pathValue,
		Headers: headers,
		Query:   query,
		Body:    parseRequestBody(it.Request.Body),
	}
	*out = append(*out, req)
}

func parseRequestURL(rawURL json.RawMessage) (string, map[string]string) {
	if len(rawURL) == 0 {
		return "/", nil
	}

	var direct string
	if err := json.Unmarshal(rawURL, &direct); err == nil {
		return splitPathAndQuery(direct)
	}

	var u urlObject
	if err := json.Unmarshal(rawURL, &u); err == nil {
		if strings.TrimSpace(u.Raw) != "" {
			pathValue, query := splitPathAndQuery(u.Raw)
			if query == nil && len(u.Query) > 0 {
				query = make(map[string]string)
			}
			mergeQuery(query, u.Query)
			return pathValue, query
		}

		pathValue := "/"
		if len(u.Path) > 0 {
			pathValue = "/" + path.Join(u.Path...)
		}
		query := make(map[string]string)
		mergeQuery(query, u.Query)
		if len(query) == 0 {
			query = nil
		}
		return pathValue, query
	}

	return "/", nil
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

func mergeQuery(query map[string]string, extra []queryParam) {
	if query == nil {
		return
	}
	for _, q := range extra {
		if q.Disabled || strings.TrimSpace(q.Key) == "" {
			continue
		}
		query[strings.TrimSpace(q.Key)] = q.Value
	}
}

func parseRequestBody(body bodyObject) string {
	mode := strings.ToLower(strings.TrimSpace(body.Mode))
	switch mode {
	case "", "raw":
		return body.Raw
	default:
		return body.Raw
	}
}
