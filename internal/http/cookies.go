package apixhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type persistedCookieSet struct {
	URL     string         `json:"url"`
	Cookies []*http.Cookie `json:"cookies"`
}

type PersistentCookieJar struct {
	mu   sync.Mutex
	jar  *cookiejar.Jar
	path string
	urls map[string]struct{}
}

func NewPersistentCookieJar(path string) (*PersistentCookieJar, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultCookieJarPath
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	out := &PersistentCookieJar{
		jar:  jar,
		path: path,
		urls: make(map[string]struct{}),
	}
	if err := out.load(); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *PersistentCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.jar.SetCookies(u, cookies)
	p.urls[cookieScope(u)] = struct{}{}
	_ = p.saveLocked()
}

func (p *PersistentCookieJar) Cookies(u *url.URL) []*http.Cookie {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.jar.Cookies(u)
}

func (p *PersistentCookieJar) load() error {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading cookie jar %q: %w", p.path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var entries []persistedCookieSet
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parsing cookie jar %q: %w", p.path, err)
	}

	for _, entry := range entries {
		u, err := url.Parse(entry.URL)
		if err != nil {
			continue
		}
		p.jar.SetCookies(u, entry.Cookies)
		p.urls[cookieScope(u)] = struct{}{}
	}
	return nil
}

func (p *PersistentCookieJar) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return fmt.Errorf("creating cookie jar directory: %w", err)
	}

	keys := make([]string, 0, len(p.urls))
	for raw := range p.urls {
		keys = append(keys, raw)
	}
	sort.Strings(keys)

	entries := make([]persistedCookieSet, 0, len(keys))
	for _, raw := range keys {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		cookies := p.jar.Cookies(u)
		if len(cookies) == 0 {
			continue
		}
		entries = append(entries, persistedCookieSet{
			URL:     raw,
			Cookies: cookies,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding cookie jar: %w", err)
	}
	if err := os.WriteFile(p.path, data, 0o644); err != nil {
		return fmt.Errorf("writing cookie jar %q: %w", p.path, err)
	}
	return nil
}

func cookieScope(u *url.URL) string {
	scope := u.Scheme + "://" + u.Host
	p := strings.TrimSpace(u.Path)
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return scope + p
}
