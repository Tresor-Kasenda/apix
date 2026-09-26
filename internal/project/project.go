// Package project locates the apix project root and resolves every
// project-relative path (config, saved requests, environments, local state).
//
// The root is discovered like git does: starting from the working directory,
// apix walks up the tree until it finds an apix.yaml. This lets apix be used
// from any sub-directory of a project (backend/, src/, services/api/, ...).
package project

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the project configuration file.
	ConfigFileName = "apix.yaml"

	// RootEnvVar forces the project root, bypassing upward discovery.
	RootEnvVar = "APIX_PROJECT_DIR"

	defaultRequestsDir = "requests"
	defaultEnvDir      = "env"
	stateDirName       = ".apix"
)

// layout holds the directory overrides a project can declare in apix.yaml.
type layout struct {
	RequestsDir string `yaml:"requests_dir"`
	EnvDir      string `yaml:"env_dir"`
}

// Root returns the project root directory. It returns "." when the root is the
// working directory (or when no apix.yaml exists anywhere up the tree), so
// paths stay short and relative in the common case.
func Root() string {
	if forced := os.Getenv(RootEnvVar); forced != "" {
		return forced
	}

	wd, err := os.Getwd()
	if err != nil {
		return "."
	}

	dir := wd
	for {
		if fileExists(filepath.Join(dir, ConfigFileName)) {
			if dir == wd {
				return "."
			}
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// Found reports whether an apix.yaml exists at the project root.
func Found() bool {
	return fileExists(ConfigFile())
}

// Path joins elements onto the project root.
func Path(elem ...string) string {
	return filepath.Join(append([]string{Root()}, elem...)...)
}

// ConfigFile returns the path to apix.yaml.
func ConfigFile() string {
	return Path(ConfigFileName)
}

// RequestsDir returns the directory holding saved requests.
func RequestsDir() string {
	return resolveDir(readLayout().RequestsDir, defaultRequestsDir)
}

// EnvDir returns the directory holding environment files.
func EnvDir() string {
	return resolveDir(readLayout().EnvDir, defaultEnvDir)
}

// StateDir returns the directory holding local, git-ignored state
// (token, cookies, history, last request).
func StateDir() string {
	return Path(stateDirName)
}

// StatePath joins elements onto the state directory.
func StatePath(elem ...string) string {
	return filepath.Join(append([]string{StateDir()}, elem...)...)
}

func resolveDir(configured, fallback string) string {
	if configured == "" {
		return Path(fallback)
	}
	if filepath.IsAbs(configured) {
		return configured
	}
	return Path(configured)
}

func readLayout() layout {
	var l layout
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		return l
	}
	_ = yaml.Unmarshal(data, &l)
	return l
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
