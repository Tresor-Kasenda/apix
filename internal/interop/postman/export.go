package postman

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/request"
)

type exportCollection struct {
	Info exportInfo   `json:"info"`
	Item []exportItem `json:"item"`
}

type exportInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type exportItem struct {
	Name    string        `json:"name"`
	Request exportRequest `json:"request"`
}

type exportRequest struct {
	Method string          `json:"method"`
	Header []header        `json:"header,omitempty"`
	URL    exportURL       `json:"url"`
	Body   *exportBodyWrap `json:"body,omitempty"`
}

type exportURL struct {
	Raw   string       `json:"raw"`
	Query []queryParam `json:"query,omitempty"`
}

type exportBodyWrap struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

const postmanSchemaURL = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

func ExportCollection(requests []request.SavedRequest, name string) ([]byte, error) {
	if strings.TrimSpace(name) == "" {
		name = "apix export"
	}

	items := make([]exportItem, 0, len(requests))
	for _, req := range requests {
		items = append(items, exportItem{
			Name:    exportRequestName(req),
			Request: toPostmanRequest(req),
		})
	}

	payload := exportCollection{
		Info: exportInfo{
			Name:   name,
			Schema: postmanSchemaURL,
		},
		Item: items,
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding postman export: %w", err)
	}
	return out, nil
}

func exportRequestName(req request.SavedRequest) string {
	if strings.TrimSpace(req.Name) != "" {
		return strings.TrimSpace(req.Name)
	}
	method := strings.TrimSpace(strings.ToUpper(req.Method))
	pathValue := strings.TrimSpace(req.Path)
	if method == "" {
		method = "REQUEST"
	}
	if pathValue == "" {
		pathValue = "/"
	}
	return method + " " + pathValue
}

func toPostmanRequest(req request.SavedRequest) exportRequest {
	rawURL := buildRawURL(req.Path, req.Query)
	headers := make([]header, 0, len(req.Headers))
	if len(req.Headers) > 0 {
		keys := make([]string, 0, len(req.Headers))
		for key := range req.Headers {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			headers = append(headers, header{
				Key:   key,
				Value: req.Headers[key],
			})
		}
	}

	out := exportRequest{
		Method: strings.ToUpper(strings.TrimSpace(req.Method)),
		Header: headers,
		URL: exportURL{
			Raw: rawURL,
		},
	}
	if len(req.Query) > 0 {
		keys := make([]string, 0, len(req.Query))
		for key := range req.Query {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			out.URL.Query = append(out.URL.Query, queryParam{
				Key:   key,
				Value: req.Query[key],
			})
		}
	}
	if req.Body != "" {
		out.Body = &exportBodyWrap{
			Mode: "raw",
			Raw:  req.Body,
		}
	}
	return out
}

func buildRawURL(pathValue string, query map[string]string) string {
	base := strings.TrimSpace(pathValue)
	if base == "" {
		base = "/"
	}
	if len(query) == 0 {
		return base
	}

	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}

	values := parsed.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values.Set(key, query[key])
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}
