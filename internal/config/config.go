package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Tresor-Kasend/apix/internal/env"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Project    string            `mapstructure:"project"    yaml:"project"`
	BaseURL    string            `mapstructure:"base_url"   yaml:"base_url"`
	Timeout    int               `mapstructure:"timeout"    yaml:"timeout"`
	Headers    map[string]string `mapstructure:"headers"    yaml:"headers"`
	Auth       AuthConfig        `mapstructure:"auth"       yaml:"auth"`
	CurrentEnv string            `mapstructure:"current_env" yaml:"current_env"`
	Variables  map[string]string `mapstructure:"variables"  yaml:"variables,omitempty"`
}

type AuthConfig struct {
	Type         string `mapstructure:"type"          yaml:"type"`
	Token        string `mapstructure:"token"         yaml:"token,omitempty"`
	TokenPath    string `mapstructure:"token_path"    yaml:"token_path,omitempty"`
	HeaderName   string `mapstructure:"header_name"   yaml:"header_name,omitempty"`
	HeaderFormat string `mapstructure:"header_format" yaml:"header_format,omitempty"`
	LoginRequest string `mapstructure:"login_request" yaml:"login_request,omitempty"`
	Username     string `mapstructure:"username"      yaml:"username,omitempty"`
	Password     string `mapstructure:"password"      yaml:"password,omitempty"`
	APIKey       string `mapstructure:"api_key"       yaml:"api_key,omitempty"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("apix")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("reading apix.yaml: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing apix.yaml: %w", err)
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

	if token, err := loadToken(); err == nil && token != "" {
		cfg.Auth.Token = token
	}

	if cfg.CurrentEnv != "" {
		_ = overlayEnv(&cfg, cfg.CurrentEnv)
	}

	return &cfg, nil
}

func SaveToken(token string) error {
	if err := os.MkdirAll(".apix", 0o755); err != nil {
		return fmt.Errorf("creating .apix directory: %w", err)
	}
	path := filepath.Join(".apix", "token")
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	return nil
}

func loadToken() (string, error) {
	path := filepath.Join(".apix", "token")
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
	_, err := os.Stat("apix.yaml")
	return err == nil
}

func WriteDefault(projectName, baseURL, authType string) error {
	if projectName == "" {
		projectName = "my-api"
	}
	if authType == "" {
		authType = "bearer"
	}

	cfg := map[string]interface{}{
		"project":     projectName,
		"base_url":    baseURL,
		"timeout":     30,
		"current_env": "dev",
		"headers": map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		"auth": defaultAuthConfig(authType),
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	if err := os.WriteFile("apix.yaml", data, 0o644); err != nil {
		return fmt.Errorf("writing apix.yaml: %w", err)
	}
	return nil
}

func defaultAuthConfig(authType string) map[string]interface{} {
	switch authType {
	case "none":
		return map[string]interface{}{
			"type": "none",
		}
	case "basic":
		return map[string]interface{}{
			"type":        "basic",
			"header_name": "Authorization",
		}
	case "api_key":
		return map[string]interface{}{
			"type":          "api_key",
			"header_name":   "X-API-Key",
			"header_format": "${API_KEY}",
		}
	case "custom":
		return map[string]interface{}{
			"type":          "custom",
			"header_name":   "Authorization",
			"header_format": "${TOKEN}",
		}
	case "bearer":
		fallthrough
	default:
		return map[string]interface{}{
			"type":          "bearer",
			"token_path":    "data.token",
			"header_name":   "Authorization",
			"header_format": "Bearer ${TOKEN}",
		}
	}
}
