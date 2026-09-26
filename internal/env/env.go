package env

import (
	"bytes"
	"fmt"
	"github.com/Tresor-Kasend/apix/internal/project"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type EnvConfig struct {
	BaseURL   string            `yaml:"base_url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	Auth      *AuthOverride     `yaml:"auth,omitempty"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

type AuthOverride struct {
	Type         string `yaml:"type,omitempty"`
	Token        string `yaml:"token,omitempty"`
	TokenPath    string `yaml:"token_path,omitempty"`
	HeaderName   string `yaml:"header_name,omitempty"`
	HeaderFormat string `yaml:"header_format,omitempty"`
	LoginRequest string `yaml:"login_request,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	APIKey       string `yaml:"api_key,omitempty"`
}

func Load(name string) (*EnvConfig, error) {
	path := envFilePath(name)
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

func List() ([]string, error) {
	entries, err := os.ReadDir(project.EnvDir())
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

func Create(name string) error {
	if err := os.MkdirAll(project.EnvDir(), 0o755); err != nil {
		return fmt.Errorf("creating env directory: %w", err)
	}

	path := envFilePath(name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("environment %q already exists", name)
	}

	// Every key is optional: an empty environment inherits everything from
	// apix.yaml, so creating one never silently overrides the project base_url.
	data := []byte(newEnvTemplate)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing environment file: %w", err)
	}
	return nil
}

const newEnvTemplate = `# Environment overrides. Every key is optional and falls back to apix.yaml.
# Values may reference variables: ${VAR} or ${VAR:-default}.
#
# base_url: https://staging.example.com
# headers:
#   X-Debug: "true"
# auth:
#   token: ${STAGING_TOKEN}
variables: {}
`

func Copy(source, dest string) error {
	sourcePath := envFilePath(source)
	destPath := envFilePath(dest)

	if source == dest {
		return fmt.Errorf("source and destination environments must be different")
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading source environment %q: %w", source, err)
	}

	if err := os.MkdirAll(project.EnvDir(), 0o755); err != nil {
		return fmt.Errorf("creating env directory: %w", err)
	}

	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("environment %q already exists", dest)
	}

	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return fmt.Errorf("writing destination environment %q: %w", dest, err)
	}

	return nil
}

func Delete(name, activeName string) error {
	if name == "" {
		return fmt.Errorf("environment name is required")
	}
	if activeName != "" && name == activeName {
		return fmt.Errorf("cannot delete active environment %q", name)
	}

	path := envFilePath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment %q does not exist", name)
		}
		return fmt.Errorf("deleting environment %q: %w", name, err)
	}
	return nil
}

func SetActive(name string) error {
	path := envFilePath(name)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("environment %q does not exist", name)
	}

	return updateApixYAMLField("current_env", name)
}

func Show(name string) (string, error) {
	path := envFilePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading environment %q: %w", name, err)
	}
	return string(data), nil
}

func envFilePath(name string) string {
	return filepath.Join(project.EnvDir(), name+".yaml")
}

// updateApixYAMLField sets a top-level key in apix.yaml while preserving the
// file's comments and key order.
func updateApixYAMLField(key string, value string) error {
	data, err := os.ReadFile(project.ConfigFile())
	if err != nil {
		return fmt.Errorf("reading apix.yaml: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing apix.yaml: %w", err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("parsing apix.yaml: top level must be a mapping")
	}

	updated := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
			updated = true
			break
		}
	}
	if !updated {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
		)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("marshaling apix.yaml: %w", err)
	}
	if err := os.WriteFile(project.ConfigFile(), buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing apix.yaml: %w", err)
	}
	return nil
}
