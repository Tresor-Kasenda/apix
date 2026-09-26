package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/dotenv"
	"github.com/Tresor-Kasend/apix/internal/env"
	"github.com/Tresor-Kasend/apix/internal/project"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Environment variables that override the project configuration at runtime.
// They make apix usable in CI and in projects that don't commit an apix.yaml.
const (
	EnvVarEnvironment = "APIX_ENV"
	EnvVarBaseURL     = "APIX_BASE_URL"
)

// DefaultTokenPath lists the response fields most APIs use for auth tokens
// (OAuth2, Laravel Sanctum/Passport, JWT libraries, ...). The first match wins.
const DefaultTokenPath = "access_token|token|data.token|data.access_token"

// defaultDotEnvFiles are loaded from the project root when `dotenv` is not set.
var defaultDotEnvFiles = []string{".env", ".env.local"}

type Config struct {
	Project     string            `mapstructure:"project"      yaml:"project"                json:"project"`
	BaseURL     string            `mapstructure:"base_url"     yaml:"base_url"               json:"base_url"`
	Timeout     int               `mapstructure:"timeout"      yaml:"timeout"                json:"timeout"`
	Headers     map[string]string `mapstructure:"headers"      yaml:"headers"                json:"headers"`
	Auth        AuthConfig        `mapstructure:"auth"         yaml:"auth"                   json:"auth"`
	CurrentEnv  string            `mapstructure:"current_env"  yaml:"current_env"            json:"current_env"`
	Variables   map[string]string `mapstructure:"variables"    yaml:"variables,omitempty"    json:"variables,omitempty"`
	RequestsDir string            `mapstructure:"requests_dir" yaml:"requests_dir,omitempty" json:"requests_dir,omitempty"`
	EnvDir      string            `mapstructure:"env_dir"      yaml:"env_dir,omitempty"      json:"env_dir,omitempty"`
	DotEnv      []string          `mapstructure:"dotenv"       yaml:"dotenv,omitempty"       json:"dotenv,omitempty"`

	// DotEnvVars holds values read from the project's dotenv files. They are
	// kept apart from Variables so secrets never show up in `apix config show`.
	DotEnvVars map[string]string `mapstructure:"-" yaml:"-" json:"-"`
}

type AuthConfig struct {
	Type         string `mapstructure:"type"          yaml:"type"                   json:"type"`
	Token        string `mapstructure:"token"         yaml:"token,omitempty"        json:"token,omitempty"`
	TokenPath    string `mapstructure:"token_path"    yaml:"token_path,omitempty"   json:"token_path,omitempty"`
	HeaderName   string `mapstructure:"header_name"   yaml:"header_name,omitempty"  json:"header_name,omitempty"`
	HeaderFormat string `mapstructure:"header_format" yaml:"header_format,omitempty" json:"header_format,omitempty"`
	LoginRequest string `mapstructure:"login_request" yaml:"login_request,omitempty" json:"login_request,omitempty"`
	Username     string `mapstructure:"username"      yaml:"username,omitempty"     json:"username,omitempty"`
	Password     string `mapstructure:"password"      yaml:"password,omitempty"     json:"password,omitempty"`
	APIKey       string `mapstructure:"api_key"       yaml:"api_key,omitempty"      json:"api_key,omitempty"`
}

// VariableMap returns the variables available to requests, from lowest to
// highest precedence: dotenv files, then apix.yaml, then the active env file.
// OS environment variables are resolved at substitution time and take
// precedence over dotenv values (see request.ResolveVariables).
func (c *Config) VariableMap() map[string]string {
	vars := make(map[string]string, len(c.DotEnvVars)+len(c.Variables))
	for k, v := range c.DotEnvVars {
		if _, inOS := os.LookupEnv(k); inOS {
			continue
		}
		vars[k] = v
	}
	for k, v := range c.Variables {
		vars[k] = v
	}
	return vars
}

func Load() (*Config, error) {
	cfg, err := loadBase()
	if err != nil {
		return nil, err
	}

	envName := cfg.CurrentEnv
	if fromOS := strings.TrimSpace(os.Getenv(EnvVarEnvironment)); fromOS != "" {
		envName = fromOS
		cfg.CurrentEnv = fromOS
		if err := overlayEnv(cfg, envName); err != nil {
			return nil, err
		}
	} else if envName != "" {
		_ = overlayEnv(cfg, envName)
	}

	applyRuntimeOverrides(cfg)
	return cfg, nil
}

func LoadWithEnvOverride(envName string) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(envName) == "" {
		return cfg, nil
	}
	if err := overlayEnv(cfg, envName); err != nil {
		return nil, err
	}
	cfg.CurrentEnv = envName
	applyRuntimeOverrides(cfg)
	return cfg, nil
}

func loadBase() (*Config, error) {
	var cfg *Config
	if Exists() {
		v := viper.New()
		v.SetConfigFile(project.ConfigFile())
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading apix.yaml: %w", err)
		}
		cfg = &Config{}
		if err := v.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("parsing apix.yaml: %w", err)
		}
	} else {
		cfg = defaultConfig()
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30
	}
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	if cfg.Variables == nil {
		cfg.Variables = make(map[string]string)
	}

	files := cfg.DotEnv
	if files == nil {
		files = defaultDotEnvFiles
	}
	cfg.DotEnvVars = make(map[string]string)
	for _, f := range files {
		path := f
		if !filepath.IsAbs(path) {
			path = project.Path(f)
		}
		values, err := dotenv.LoadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading dotenv file %q: %w", f, err)
		}
		for k, v := range values {
			cfg.DotEnvVars[k] = v
		}
	}

	if token, err := loadToken(); err == nil && token != "" {
		cfg.Auth.Token = token
	}
	return cfg, nil
}

func applyRuntimeOverrides(cfg *Config) {
	if baseURL := strings.TrimSpace(os.Getenv(EnvVarBaseURL)); baseURL != "" {
		cfg.BaseURL = baseURL
	}
}

func SaveToken(token string) error {
	if err := os.MkdirAll(project.StateDir(), 0o755); err != nil {
		return fmt.Errorf("creating .apix directory: %w", err)
	}
	path := project.StatePath("token")
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	return nil
}

func loadToken() (string, error) {
	path := project.StatePath("token")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func overlayEnv(cfg *Config, envName string) error {
	envCfg, err := env.Load(envName)
	if err != nil {
		return fmt.Errorf("loading environment %q: %w", envName, err)
	}

	if envCfg.BaseURL != "" {
		cfg.BaseURL = envCfg.BaseURL
	}

	for k, v := range envCfg.Headers {
		cfg.Headers[k] = v
	}

	if envCfg.Auth != nil {
		if envCfg.Auth.Type != "" {
			cfg.Auth.Type = envCfg.Auth.Type
		}
		if envCfg.Auth.Token != "" {
			cfg.Auth.Token = envCfg.Auth.Token
		}
		if envCfg.Auth.TokenPath != "" {
			cfg.Auth.TokenPath = envCfg.Auth.TokenPath
		}
		if envCfg.Auth.HeaderName != "" {
			cfg.Auth.HeaderName = envCfg.Auth.HeaderName
		}
		if envCfg.Auth.HeaderFormat != "" {
			cfg.Auth.HeaderFormat = envCfg.Auth.HeaderFormat
		}
		if envCfg.Auth.LoginRequest != "" {
			cfg.Auth.LoginRequest = envCfg.Auth.LoginRequest
		}
		if envCfg.Auth.Username != "" {
			cfg.Auth.Username = envCfg.Auth.Username
		}
		if envCfg.Auth.Password != "" {
			cfg.Auth.Password = envCfg.Auth.Password
		}
		if envCfg.Auth.APIKey != "" {
			cfg.Auth.APIKey = envCfg.Auth.APIKey
		}
	}

	for k, v := range envCfg.Variables {
		cfg.Variables[k] = v
	}

	return nil
}

func defaultConfig() *Config {
	return &Config{
		BaseURL: "http://localhost:8000/api",
		Timeout: 30,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Variables: make(map[string]string),
	}
}

func Exists() bool {
	_, err := os.Stat(project.ConfigFile())
	return err == nil
}

func WriteDefault(projectName, baseURL, authType string) error {
	if projectName == "" {
		projectName = "my-api"
	}
	if authType == "" {
		authType = "bearer"
	}

	cfg := initialConfig{
		Project:    projectName,
		BaseURL:    baseURL,
		Timeout:    30,
		CurrentEnv: "dev",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		Auth: defaultAuthConfig(authType),
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}
	data := buf.Bytes()
	data = append([]byte(configHeader), data...)
	data = append(data, []byte(configFooter)...)

	// init always creates the config in the working directory, even when a
	// parent directory already holds another apix project.
	if err := os.WriteFile(project.ConfigFileName, data, 0o644); err != nil {
		return fmt.Errorf("writing apix.yaml: %w", err)
	}
	return nil
}

// initialConfig fixes the key order of a freshly generated apix.yaml.
type initialConfig struct {
	Project    string            `yaml:"project"`
	BaseURL    string            `yaml:"base_url"`
	Timeout    int               `yaml:"timeout"`
	CurrentEnv string            `yaml:"current_env"`
	Headers    map[string]string `yaml:"headers"`
	Auth       AuthConfig        `yaml:"auth"`
}

const configHeader = `# apix project configuration.
# Values support ${VAR} and ${VAR:-default}; variables are resolved from --var,
# env/<name>.yaml, the variables block below, OS environment, then .env files.
# Runtime overrides: APIX_ENV=<name>, APIX_BASE_URL=<url>.
`

const configFooter = `
# Optional settings:
# requests_dir: requests       # where saved requests live
# env_dir: env                 # where environment files live
# dotenv: [.env, .env.local]   # dotenv files loaded for ${VAR} resolution
# variables:
#   API_VERSION: v1
`

func defaultAuthConfig(authType string) AuthConfig {
	switch authType {
	case "none":
		return AuthConfig{Type: "none"}
	case "basic":
		return AuthConfig{Type: "basic", HeaderName: "Authorization", Username: "${API_USERNAME}", Password: "${API_PASSWORD}"}
	case "api_key":
		return AuthConfig{Type: "api_key", HeaderName: "X-API-Key", HeaderFormat: "${API_KEY}", APIKey: "${API_KEY}"}
	case "custom":
		return AuthConfig{Type: "custom", HeaderName: "Authorization", HeaderFormat: "${TOKEN}"}
	default:
		return AuthConfig{Type: "bearer", TokenPath: DefaultTokenPath, HeaderName: "Authorization", HeaderFormat: "Bearer ${TOKEN}"}
	}
}
