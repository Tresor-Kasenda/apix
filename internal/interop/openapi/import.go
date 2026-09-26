// Package openapi converts OpenAPI 3.x and Swagger 2.0 documents (JSON or
// YAML, local file or URL) into apix saved requests.
//
// Because virtually every backend stack can emit an OpenAPI document
// (FastAPI, NestJS, Spring, Laravel, Django REST, ASP.NET, Go swag, ...),
// this is the framework-agnostic way to bootstrap an apix collection.
package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Tresor-Kasend/apix/internal/request"
	"gopkg.in/yaml.v3"
)

const maxSampleDepth = 8

var (
	methodOrder = []string{"get", "post", "put", "patch", "delete", "head", "options"}
	pathParam   = regexp.MustCompile(`\{([^}/]+)\}`)
	varNameSafe = regexp.MustCompile(`[^A-Za-z0-9_]`)
)

// Result is the outcome of converting a document.
type Result struct {
	Title    string
	Version  string
	Server   string // first server URL (may be relative, e.g. "/api/v1")
	BasePath string // path component of Server ("" when none)
	Requests []request.SavedRequest
}

// Options tweak the conversion.
type Options struct {
	// IncludeBasePath prefixes every request path with the server base path,
	// for when apix's base_url points at the host root.
	IncludeBasePath bool
}

// ParseSource loads a document from a file path or an http(s) URL.
func ParseSource(source string, opts Options) (*Result, error) {
	data, err := readSource(source)
	if err != nil {
		return nil, err
	}
	return Parse(data, opts)
}

// Parse converts a raw OpenAPI/Swagger document.
func Parse(data []byte, opts Options) (*Result, error) {
	doc, err := decode(data)
	if err != nil {
		return nil, err
	}

	_, isV3 := doc["openapi"]
	_, isV2 := doc["swagger"]
	if !isV3 && !isV2 {
		return nil, fmt.Errorf("document is not an OpenAPI 3 or Swagger 2 specification")
	}

	c := converter{doc: doc}
	res := &Result{}
	if info, ok := doc["info"].(map[string]interface{}); ok {
		res.Title = asString(info["title"])
		res.Version = asString(info["version"])
	}
	res.Server, res.BasePath = serverOf(doc, isV2)

	paths, _ := doc["paths"].(map[string]interface{})
	for _, p := range sortedKeys(paths) {
		item, ok := c.deref(paths[p]).(map[string]interface{})
		if !ok {
			continue
		}
		shared := asSlice(item["parameters"])

		for _, method := range methodOrder {
			op, ok := item[method].(map[string]interface{})
			if !ok {
				continue
			}
			reqPath := p
			if opts.IncludeBasePath && res.BasePath != "" {
				reqPath = strings.TrimRight(res.BasePath, "/") + "/" + strings.TrimLeft(p, "/")
			}
			res.Requests = append(res.Requests, c.buildRequest(method, reqPath, op, shared, isV2))
		}
	}

	if len(res.Requests) == 0 {
		return nil, fmt.Errorf("no operations found in specification")
	}
	return res, nil
}

type converter struct {
	doc map[string]interface{}
}

func (c converter) buildRequest(method, path string, op map[string]interface{}, shared []interface{}, isV2 bool) request.SavedRequest {
	req := request.SavedRequest{
		Name:   asString(op["operationId"]),
		Method: strings.ToUpper(method),
		Path:   pathParam.ReplaceAllStringFunc(path, func(m string) string { return "${" + varName(m[1:len(m)-1]) + "}" }),
	}

	params := append(append([]interface{}{}, shared...), asSlice(op["parameters"])...)
	for _, raw := range params {
		param, ok := c.deref(raw).(map[string]interface{})
		if !ok {
			continue
		}
		name := asString(param["name"])
		required, _ := param["required"].(bool)

		switch asString(param["in"]) {
		case "query":
			if !required {
				continue
			}
			if req.Query == nil {
				req.Query = map[string]string{}
			}
			req.Query[name] = c.paramValue(param, name)
		case "header":
			if !required || strings.EqualFold(name, "Authorization") {
				continue
			}
			if req.Headers == nil {
				req.Headers = map[string]string{}
			}
			req.Headers[name] = c.paramValue(param, name)
		case "body": // Swagger 2
			if body := c.sampleJSON(param["schema"]); body != "" {
				req.Body = body
			}
		}
	}

	if !isV2 {
		c.applyRequestBody(&req, op["requestBody"])
	}
	return req
}

// applyRequestBody fills the body from an OpenAPI 3 requestBody, preferring an
// explicit example over a schema-generated sample.
func (c converter) applyRequestBody(req *request.SavedRequest, raw interface{}) {
	body, ok := c.deref(raw).(map[string]interface{})
	if !ok {
		return
	}
	content, _ := body["content"].(map[string]interface{})
	for _, mediaType := range sortedKeys(content) {
		if !strings.Contains(mediaType, "json") {
			continue
		}
		media, _ := content[mediaType].(map[string]interface{})
		if example, ok := media["example"]; ok {
			req.Body = toJSON(example)
			return
		}
		if examples, ok := media["examples"].(map[string]interface{}); ok {
			for _, key := range sortedKeys(examples) {
				if ex, ok := c.deref(examples[key]).(map[string]interface{}); ok {
					if value, ok := ex["value"]; ok {
						req.Body = toJSON(value)
						return
					}
				}
			}
		}
		req.Body = c.sampleJSON(media["schema"])
		return
	}
}

func (c converter) paramValue(param map[string]interface{}, name string) string {
	if example, ok := param["example"]; ok {
		return fmt.Sprint(example)
	}
	schema, _ := c.deref(param["schema"]).(map[string]interface{})
	for _, source := range []map[string]interface{}{param, schema} {
		if source == nil {
			continue
		}
		if example, ok := source["example"]; ok {
			return fmt.Sprint(example)
		}
		if def, ok := source["default"]; ok {
			return fmt.Sprint(def)
		}
	}
	return "${" + varName(name) + "}"
}

func (c converter) sampleJSON(schema interface{}) string {
	if schema == nil {
		return ""
	}
	sample := c.sample(schema, 0, map[string]bool{})
	if sample == nil {
		return ""
	}
	return toJSON(sample)
}

// sample builds an example value from a JSON schema.
func (c converter) sample(raw interface{}, depth int, seen map[string]bool) interface{} {
	if depth > maxSampleDepth {
		return nil
	}

	node, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	if ref := asString(node["$ref"]); ref != "" {
		if seen[ref] {
			return nil
		}
		seen[ref] = true
		defer delete(seen, ref)
		return c.sample(c.resolveRef(ref), depth+1, seen)
	}

	if example, ok := node["example"]; ok {
		return example
	}
	if def, ok := node["default"]; ok {
		return def
	}
	if enum := asSlice(node["enum"]); len(enum) > 0 {
		return enum[0]
	}

	if all := asSlice(node["allOf"]); len(all) > 0 {
		merged := map[string]interface{}{}
		for _, part := range all {
			if obj, ok := c.sample(part, depth+1, seen).(map[string]interface{}); ok {
				for k, v := range obj {
					merged[k] = v
				}
			}
		}
		return merged
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if options := asSlice(node[key]); len(options) > 0 {
			return c.sample(options[0], depth+1, seen)
		}
	}

	schemaType := asString(node["type"])
	if types, ok := node["type"].([]interface{}); ok && len(types) > 0 { // OpenAPI 3.1
		schemaType = asString(types[0])
	}
	if schemaType == "" {
		if _, ok := node["properties"]; ok {
			schemaType = "object"
		} else if _, ok := node["items"]; ok {
			schemaType = "array"
		}
	}

	switch schemaType {
	case "object":
		obj := map[string]interface{}{}
		props, _ := node["properties"].(map[string]interface{})
		for _, name := range sortedKeys(props) {
			prop, _ := c.deref(props[name]).(map[string]interface{})
			if readOnly, _ := prop["readOnly"].(bool); readOnly {
				continue
			}
			obj[name] = c.sample(props[name], depth+1, seen)
		}
		return obj
	case "array":
		if item := c.sample(node["items"], depth+1, seen); item != nil {
			return []interface{}{item}
		}
		return []interface{}{}
	case "integer":
		return 0
	case "number":
		return 0.0
	case "boolean":
		return false
	case "string":
		return sampleString(asString(node["format"]))
	}
	return nil
}

func sampleString(format string) string {
	switch format {
	case "email":
		return "user@example.com"
	case "date":
		return "2024-01-01"
	case "date-time":
		return "2024-01-01T00:00:00Z"
	case "uuid":
		return "${UUID}"
	case "uri", "url":
		return "https://example.com"
	case "password":
		return "secret"
	default:
		return "string"
	}
}

func (c converter) deref(raw interface{}) interface{} {
	for i := 0; i < 10; i++ {
		node, ok := raw.(map[string]interface{})
		if !ok {
			return raw
		}
		ref := asString(node["$ref"])
		if ref == "" {
			return raw
		}
		raw = c.resolveRef(ref)
	}
	return raw
}

// resolveRef follows a local JSON pointer such as "#/components/schemas/User".
func (c converter) resolveRef(ref string) interface{} {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var current interface{} = c.doc
	for _, token := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[token]
	}
	return current
}

func serverOf(doc map[string]interface{}, isV2 bool) (server, basePath string) {
	if isV2 {
		basePath = asString(doc["basePath"])
		host := asString(doc["host"])
		if host == "" {
			return basePath, basePath
		}
		scheme := "https"
		if schemes := asSlice(doc["schemes"]); len(schemes) > 0 {
			scheme = asString(schemes[0])
		}
		return scheme + "://" + host + basePath, basePath
	}

	servers := asSlice(doc["servers"])
	if len(servers) == 0 {
		return "", ""
	}
	first, _ := servers[0].(map[string]interface{})
	server = asString(first["url"])
	if vars, ok := first["variables"].(map[string]interface{}); ok {
		for name, raw := range vars {
			if v, ok := raw.(map[string]interface{}); ok {
				server = strings.ReplaceAll(server, "{"+name+"}", asString(v["default"]))
			}
		}
	}
	if u, err := url.Parse(server); err == nil {
		basePath = strings.TrimRight(u.Path, "/")
	}
	return server, basePath
}

func readSource(source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(source)
		if err != nil {
			return nil, fmt.Errorf("downloading specification: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("downloading specification: HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("reading specification: %w", err)
	}
	return data, nil
}

func decode(data []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("{")) {
		if err := json.Unmarshal(trimmed, &doc); err != nil {
			return nil, fmt.Errorf("parsing JSON specification: %w", err)
		}
		return doc, nil
	}
	if err := yaml.Unmarshal(trimmed, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML specification: %w", err)
	}
	return doc, nil
}

func toJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func varName(name string) string {
	return varNameSafe.ReplaceAllString(name, "_")
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func asSlice(v interface{}) []interface{} {
	s, _ := v.([]interface{})
	return s
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
