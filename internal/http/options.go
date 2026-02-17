package apixhttp

import "time"

const (
	DefaultRetryDelay    = 200 * time.Millisecond
	DefaultCookieJarPath = ".apix/cookies.jar"
)

type NetworkOptions struct {
	Retry         int
	RetryDelay    time.Duration
	ProxyURL      string
	Insecure      bool
	CertFile      string
	KeyFile       string
	NoCookies     bool
	CookieJarPath string
}
