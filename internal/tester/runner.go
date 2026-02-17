package tester

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

const defaultRequestsDir = "requests"

type ExecuteFunc func(name string, saved *request.SavedRequest, vars map[string]string, envOverride string) (*apixhttp.Response, error)

type RunnerOptions struct {
	Name        string
	Dir         string
	Vars        map[string]string
	EnvOverride string
}

type testCase struct {
	Name    string
	Request *request.SavedRequest
}

func Run(options RunnerOptions, execute ExecuteFunc) (*SuiteResult, error) {
	if execute == nil {
		return nil, fmt.Errorf("test runner execute callback is required")
	}

	cases, err := loadTestCases(options)
	if err != nil {
		return nil, err
	}

	suite := &SuiteResult{
		Total: len(cases),
	}
	if len(cases) == 0 {
		return suite, nil
	}

	startSuite := time.Now()
	for _, tc := range cases {
		startTest := time.Now()
		result := RequestResult{
			Name: tc.Name,
		}

		resp, execErr := execute(tc.Name, tc.Request, cloneVars(options.Vars), options.EnvOverride)
		result.Duration = time.Since(startTest)
		if execErr != nil {
			result.Error = fmt.Sprintf("execution error: %v", execErr)
			suite.Failed++
			suite.Results = append(suite.Results, result)
			continue
		}
		if resp == nil {
			result.Error = "execution error: empty response"
			suite.Failed++
			suite.Results = append(suite.Results, result)
			continue
		}

		failures, assertErr := EvaluateExpect(tc.Request.Expect, resp)
		if assertErr != nil {
			result.Error = assertErr.Error()
			suite.Failed++
			suite.Results = append(suite.Results, result)
			continue
		}
		result.Failures = failures
		if len(failures) > 0 {
			suite.Failed++
			suite.Results = append(suite.Results, result)
			continue
		}

		result.Passed = true
		suite.Passed++
		suite.Results = append(suite.Results, result)
	}
	suite.Duration = time.Since(startSuite)

	return suite, nil
}

func loadTestCases(options RunnerOptions) ([]testCase, error) {
	if options.Name != "" {
		tc, err := loadSingleCase(options.Name, options.Dir)
		if err != nil {
			return nil, err
		}
		if !tc.Request.HasExpect() {
			return nil, fmt.Errorf("request %q has no expect block", tc.Name)
		}
		return []testCase{tc}, nil
	}

	if options.Dir != "" {
		return loadCasesFromDir(options.Dir)
	}
	return loadCasesFromDefaultDir()
}

func loadSingleCase(name, dir string) (testCase, error) {
	if dir == "" {
		saved, err := request.Load(name)
		if err != nil {
			return testCase{}, err
		}
		return testCase{Name: name, Request: saved}, nil
	}

	path, err := findRequestFileByName(dir, name)
	if err != nil {
		return testCase{}, err
	}
	saved, err := request.LoadFromPath(path)
	if err != nil {
		return testCase{}, err
	}

	requestName := saved.Name
	if requestName == "" {
		requestName = nameFromPath(path)
	}
	return testCase{Name: requestName, Request: saved}, nil
}

func loadCasesFromDefaultDir() ([]testCase, error) {
	names, err := request.ListSaved()
	if err != nil {
		return nil, err
	}

	cases := make([]testCase, 0)
	for _, name := range names {
		saved, err := request.Load(name)
		if err != nil {
			return nil, err
		}
		if !saved.HasExpect() {
			continue
		}
		requestName := saved.Name
		if requestName == "" {
			requestName = name
		}
		cases = append(cases, testCase{Name: requestName, Request: saved})
	}

	return cases, nil
}

func loadCasesFromDir(dir string) ([]testCase, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if isYAMLFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning directory %q: %w", dir, err)
	}
	sort.Strings(paths)

	cases := make([]testCase, 0, len(paths))
	for _, path := range paths {
		saved, err := request.LoadFromPath(path)
		if err != nil {
			return nil, err
		}
		if !saved.HasExpect() {
			continue
		}
		requestName := saved.Name
		if requestName == "" {
			requestName = nameFromPath(path)
		}
		cases = append(cases, testCase{
			Name:    requestName,
			Request: saved,
		})
	}

	return cases, nil
}

func findRequestFileByName(dir, name string) (string, error) {
	if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		path := filepath.Join(dir, name)
		if isYAMLFile(path) {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	candidates := []string{
		filepath.Join(dir, name+".yaml"),
		filepath.Join(dir, name+".yml"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("request %q not found in directory %q", name, dir)
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func nameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")
}

func cloneVars(vars map[string]string) map[string]string {
	out := make(map[string]string, len(vars))
	for key, value := range vars {
		out[key] = value
	}
	return out
}
