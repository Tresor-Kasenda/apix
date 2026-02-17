package runner

import (
	"fmt"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

type ExecuteSavedRequestFunc func(name string, vars map[string]string, envOverride string) (*apixhttp.Response, error)

type ChainResult struct {
	Total      int
	Executed   int
	Captured   int
	FinalVars  map[string]string
	LastStatus int
}

func RunChain(names []string, flagVars map[string]string, envOverride string, execute ExecuteSavedRequestFunc) (*ChainResult, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one request name is required")
	}
	if execute == nil {
		return nil, fmt.Errorf("chain executor callback is required")
	}

	ctx := NewRuntimeContext(flagVars)
	result := &ChainResult{Total: len(names)}

	for _, name := range names {
		saved, err := request.Load(name)
		if err != nil {
			return result, fmt.Errorf("chain request %q: %w", name, err)
		}

		resp, err := execute(name, ctx.Snapshot(), envOverride)
		if err != nil {
			return result, fmt.Errorf("chain request %q failed: %w", name, err)
		}
		if resp == nil {
			return result, fmt.Errorf("chain request %q failed: empty response", name)
		}

		result.Executed++
		result.LastStatus = resp.StatusCode

		if resp.StatusCode >= 400 {
			return result, fmt.Errorf("chain request %q returned HTTP %d %s", name, resp.StatusCode, resp.Status)
		}

		captured, err := CaptureVariables(saved.Capture, resp)
		if err != nil {
			return result, fmt.Errorf("chain request %q capture failed: %w", name, err)
		}
		if len(captured) > 0 {
			ctx.Merge(captured)
			result.Captured += len(captured)
		}
	}

	result.FinalVars = ctx.Snapshot()
	return result, nil
}

func CaptureVariables(capture map[string]string, resp *apixhttp.Response) (map[string]string, error) {
	if len(capture) == 0 {
		return nil, nil
	}

	captured := make(map[string]string, len(capture))
	for varName, path := range capture {
		if varName == "" {
			return nil, fmt.Errorf("capture variable name cannot be empty")
		}
		if path == "" {
			return nil, fmt.Errorf("capture path for %q cannot be empty", varName)
		}

		value, err := resp.ExtractField(path)
		if err != nil {
			return nil, fmt.Errorf("%s <- %s: %w", varName, path, err)
		}
		captured[varName] = value
	}

	return captured, nil
}
