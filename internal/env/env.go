// Package env manages apix environment files (env/<name>.yaml).
package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// EnvConfig represents the contents of an environment file.
type EnvConfig struct {
	BaseURL   string            `yaml:"base_url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	Auth      *AuthOverride     `yaml:"auth,omitempty"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

// AuthOverride holds auth fields that can be overridden per environment.
type AuthOverride struct {
	Type  string `yaml:"type,omitempty"`
	Token string `yaml:"token,omitempty"`
}

// Load reads and parses an environment file by name.
func Load(name string) (*EnvConfig, error) {
	path := filepath.Join("env", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading environment %q: %w", name, err)
	}

	var cfg EnvConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing environment %q: %w", name, err)
	}
	return &cfg, nil
}

// List returns the names of all available environments.
func List() ([]string, error) {
	entries, err := os.ReadDir("env")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing environments: %w", err)
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

// Create creates a new environment file with a scaffold template.
func Create(name string) error {
	if err := os.MkdirAll("env", 0o755); err != nil {
		return fmt.Errorf("creating env directory: %w", err)
	}

	path := filepath.Join("env", name+".yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("environment %q already exists", name)
	}

	cfg := EnvConfig{
		BaseURL:   "http://localhost:8000/api",
		Headers:   map[string]string{},
		Variables: map[string]string{},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling environment: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing environment file: %w", err)
	}
	return nil
}

// SetActive updates the current_env field in apix.yaml.
func SetActive(name string) error {
	// Verify the environment exists.
	path := filepath.Join("env", name+".yaml")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("environment %q does not exist", name)
	}

	return updateApixYAMLField("current_env", name)
}

// Show returns the raw contents of an environment file.
func Show(name string) (string, error) {
	path := filepath.Join("env", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading environment %q: %w", name, err)
	}
	return string(data), nil
}

// updateApixYAMLField reads apix.yaml, sets a top-level field, and writes it back.
func updateApixYAMLField(key string, value interface{}) error {
	data, err := os.ReadFile("apix.yaml")
	if err != nil {
		return fmt.Errorf("reading apix.yaml: %w", err)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing apix.yaml: %w", err)
	}

	if doc == nil {
		doc = make(map[string]interface{})
	}
	doc[key] = value

	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling apix.yaml: %w", err)
	}

	if err := os.WriteFile("apix.yaml", out, 0o644); err != nil {
		return fmt.Errorf("writing apix.yaml: %w", err)
	}
	return nil
}
