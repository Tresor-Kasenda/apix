package auth

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/Tresor-Kasend/apix/internal/config"
)

var templatePattern = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

func Apply(headers map[string]string, cfg *config.Config, vars map[string]string) error {
	if cfg == nil {
		return fmt.Errorf("auth config cannot be nil")
	}

	authType := strings.ToLower(strings.TrimSpace(cfg.Auth.Type))
	if authType == "" || authType == "none" {
		return nil
	}

	tmplVars := makeTemplateVars(cfg, vars)

	switch authType {
	case "bearer":
		token := cfg.Auth.Token
		if token == "" {
			token = tmplVars["TOKEN"]
		}
		if token == "" {
			return nil
		}

		headerName := defaultHeaderName(cfg.Auth.HeaderName, "Authorization")
		format := cfg.Auth.HeaderFormat
		if strings.TrimSpace(format) == "" {
			format = "Bearer ${TOKEN}"
		}
		tmplVars["TOKEN"] = token
		headers[headerName] = renderTemplate(format, tmplVars)
		return nil

	case "basic":
		username := cfg.Auth.Username
		password := cfg.Auth.Password
		if username == "" || password == "" {
			return fmt.Errorf("basic auth requires username and password")
		}
		headerName := defaultHeaderName(cfg.Auth.HeaderName, "Authorization")
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		headers[headerName] = "Basic " + encoded
		return nil

	case "api_key":
		apiKey := cfg.Auth.APIKey
		if apiKey == "" {
			apiKey = cfg.Auth.Token
		}
		if apiKey == "" {
			return fmt.Errorf("api_key auth requires api_key or token")
		}

		headerName := defaultHeaderName(cfg.Auth.HeaderName, "X-API-Key")
		format := cfg.Auth.HeaderFormat
		if strings.TrimSpace(format) == "" {
			format = "${API_KEY}"
		}
		tmplVars["API_KEY"] = apiKey
		headers[headerName] = renderTemplate(format, tmplVars)
		return nil

	case "custom":
		headerName := defaultHeaderName(cfg.Auth.HeaderName, "Authorization")
		format := cfg.Auth.HeaderFormat
		if strings.TrimSpace(format) == "" {
			format = "${TOKEN}"
		}
		headers[headerName] = renderTemplate(format, tmplVars)
		return nil

	default:
		return fmt.Errorf("unsupported auth type %q", cfg.Auth.Type)
	}
}

func makeTemplateVars(cfg *config.Config, vars map[string]string) map[string]string {
	result := make(map[string]string, len(vars)+5)
	for k, v := range vars {
		result[k] = v
	}

	if cfg.Auth.Token != "" {
		result["TOKEN"] = cfg.Auth.Token
	}
	if cfg.Auth.Username != "" {
		result["USERNAME"] = cfg.Auth.Username
	}
	if cfg.Auth.Password != "" {
		result["PASSWORD"] = cfg.Auth.Password
	}
	if cfg.Auth.APIKey != "" {
		result["API_KEY"] = cfg.Auth.APIKey
	}
	return result
}

func defaultHeaderName(configured, fallback string) string {
	if strings.TrimSpace(configured) == "" {
		return fallback
	}
	return configured
}

func renderTemplate(template string, vars map[string]string) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		groups := templatePattern.FindStringSubmatch(match)
		if len(groups) != 2 {
			return match
		}
		if value, ok := vars[groups[1]]; ok {
			return value
		}
		return match
	})
}
