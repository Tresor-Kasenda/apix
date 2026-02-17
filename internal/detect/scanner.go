package detect

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var skipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
	".apix":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"target":       true,
	"build":        true,
	"dist":         true,
	".next":        true,
	".tox":         true,
	"env":          true,
	"bin":          true,
}

func scanRoutes(root string, cfg scanConfig) []DetectedRoute {
	var routes []DetectedRoute

	// Scan specific files first (e.g. routes/api.php, config/routes.rb).
	for _, f := range cfg.files {
		path := filepath.Join(root, f)
		routes = append(routes, scanFile(path, cfg.patterns)...)
	}

	// Walk directory tree for file extension matches.
	if len(cfg.fileExts) > 0 {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			// Check specific filenames (e.g. "urls.py" for Django).
			if len(cfg.files) > 0 && !matchesFileName(d.Name(), cfg.files) && !matchesExt(d.Name(), cfg.fileExts) {
				return nil
			}
			if len(cfg.files) == 0 && !matchesExt(d.Name(), cfg.fileExts) {
				return nil
			}

			routes = append(routes, scanFile(path, cfg.patterns)...)
			return nil
		})
	}

	return deduplicate(routes)
}

func scanFile(path string, patterns []*regexp.Regexp) []DetectedRoute {
	data, err := readCapped(path, 512*1024)
	if err != nil {
		return nil
	}

	var routes []DetectedRoute
	content := string(data)

	for _, pat := range patterns {
		matches := pat.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			route := extractRoute(pat, m)
			if route.Path != "" {
				routes = append(routes, route)
			}
		}
	}
	return routes
}

func extractRoute(pat *regexp.Regexp, match []string) DetectedRoute {
	route := DetectedRoute{Method: "GET"}

	for i, name := range pat.SubexpNames() {
		if i >= len(match) {
			break
		}
		switch name {
		case "method":
			route.Method = strings.ToUpper(match[i])
		case "path":
			route.Path = match[i]
		}
	}
	return route
}

func readCapped(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := info.Size()
	if size > maxBytes {
		size = maxBytes
	}

	buf := make([]byte, size)
	n, _ := f.Read(buf)
	return buf[:n], nil
}

func matchesExt(name string, exts []string) bool {
	ext := filepath.Ext(name)
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

func matchesFileName(name string, files []string) bool {
	for _, f := range files {
		// Compare just the filename portion (e.g. "urls.py" matches "app/urls.py").
		if filepath.Base(f) == name {
			return true
		}
	}
	return false
}

func deduplicate(routes []DetectedRoute) []DetectedRoute {
	seen := make(map[string]bool)
	var result []DetectedRoute
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if !seen[key] {
			seen[key] = true
			result = append(result, r)
		}
	}
	return result
}
