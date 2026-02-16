// collection.go provides loading and saving of request YAML files.
package request

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SavedRequest represents a saved HTTP request stored in requests/<name>.yaml.
type SavedRequest struct {
	Name    string            `yaml:"name"`
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Query   map[string]string `yaml:"query,omitempty"`
	Body    string            `yaml:"body,omitempty"`
}

// Save writes a request to requests/<name>.yaml.
func Save(name string, req SavedRequest) error {
	if err := os.MkdirAll("requests", 0o755); err != nil {
		return fmt.Errorf("creating requests directory: %w", err)
	}

	req.Name = name
	data, err := yaml.Marshal(&req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	path := filepath.Join("requests", name+".yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing request file: %w", err)
	}
	return nil
}

// Load reads a saved request from requests/<name>.yaml.
func Load(name string) (*SavedRequest, error) {
	path := filepath.Join("requests", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading request %q: %w", name, err)
	}

	var req SavedRequest
	if err := yaml.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parsing request %q: %w", name, err)
	}
	return &req, nil
}

// SaveLast persists the most recent request to .apix/last_request.yaml.
func SaveLast(req SavedRequest) error {
	if err := os.MkdirAll(".apix", 0o755); err != nil {
		return fmt.Errorf("creating .apix directory: %w", err)
	}

	data, err := yaml.Marshal(&req)
	if err != nil {
		return fmt.Errorf("marshaling last request: %w", err)
	}

	path := filepath.Join(".apix", "last_request.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("saving last request: %w", err)
	}
	return nil
}

// LoadLast reads the most recently executed request from .apix/last_request.yaml.
func LoadLast() (*SavedRequest, error) {
	path := filepath.Join(".apix", "last_request.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no last request found: %w", err)
	}

	var req SavedRequest
	if err := yaml.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parsing last request: %w", err)
	}
	return &req, nil
}

// ListSaved returns the names of all saved requests.
func ListSaved() ([]string, error) {
	entries, err := os.ReadDir("requests")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing requests: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			names = append(names, strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml"))
		}
	}
	return names, nil
}
