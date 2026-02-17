package auth

import (
	"fmt"
	"net/http"

	"github.com/Tresor-Kasend/apix/internal/config"
)

func RefreshIfNeeded(
	cfg *config.Config,
	requestName string,
	statusCode int,
	alreadyRetried bool,
	skipAutoRefresh bool,
	runLogin func(name string) error,
) (bool, error) {
	if skipAutoRefresh || alreadyRetried {
		return false, nil
	}
	if statusCode != http.StatusUnauthorized {
		return false, nil
	}
	if cfg == nil {
		return false, nil
	}

	loginRequest := cfg.Auth.LoginRequest
	if loginRequest == "" {
		return false, nil
	}
	if requestName == loginRequest {
		return false, nil
	}
	if runLogin == nil {
		return false, fmt.Errorf("auto refresh callback is not configured")
	}

	if err := runLogin(loginRequest); err != nil {
		return false, fmt.Errorf("re-authentication with %q failed: %w", loginRequest, err)
	}

	return true, nil
}
