package watch

import (
	"fmt"
	"strings"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
	"github.com/Tresor-Kasend/apix/internal/runner"
)

const (
	defaultMaxDepth      = 8
	defaultMaxIterations = 128
	defaultMaxVisits     = 16
)

type ExecutorOptions struct {
	MaxDepth      int
	MaxIterations int
	MaxVisits     int
}

type Executor struct {
	execute       runner.ExecuteSavedRequestFunc
	maxDepth      int
	maxIterations int
	maxVisits     int
}

type RunResult struct {
	Response  *apixhttp.Response
	Executed  int
	Captured  int
	FinalVars map[string]string
}

type executionState struct {
	stack      []string
	visits     map[string]int
	iterations int
}

func NewExecutor(execute runner.ExecuteSavedRequestFunc, opts ExecutorOptions) (*Executor, error) {
	if execute == nil {
		return nil, fmt.Errorf("watch executor callback is required")
	}

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	maxIterations := opts.MaxIterations
	if maxIterations <= 0 {
		maxIterations = defaultMaxIterations
	}

	maxVisits := opts.MaxVisits
	if maxVisits <= 0 {
		maxVisits = defaultMaxVisits
	}

	return &Executor{
		execute:       execute,
		maxDepth:      maxDepth,
		maxIterations: maxIterations,
		maxVisits:     maxVisits,
	}, nil
}

func (e *Executor) Run(name string, vars map[string]string, envOverride string) (*RunResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("request name is required")
	}

	ctx := runner.NewRuntimeContext(vars)
	state := &executionState{
		stack:  make([]string, 0, 8),
		visits: make(map[string]int),
	}

	resp, executed, captured, err := e.runRequest(name, ctx, envOverride, state)
	if err != nil {
		return nil, err
	}

	return &RunResult{
		Response:  resp,
		Executed:  executed,
		Captured:  captured,
		FinalVars: ctx.Snapshot(),
	}, nil
}

func (e *Executor) runRequest(name string, ctx *runner.RuntimeContext, envOverride string, state *executionState) (*apixhttp.Response, int, int, error) {
	if state.iterations >= e.maxIterations {
		return nil, 0, 0, fmt.Errorf("hook guardrail: max iterations reached (%d)", e.maxIterations)
	}
	if len(state.stack) >= e.maxDepth {
		return nil, 0, 0, fmt.Errorf("hook guardrail: max depth reached (%d)", e.maxDepth)
	}
	if count := state.visits[name]; count >= e.maxVisits {
		return nil, 0, 0, fmt.Errorf("hook guardrail: request %q visited too many times (%d)", name, e.maxVisits)
	}
	if stackContains(state.stack, name) {
		return nil, 0, 0, fmt.Errorf("hook guardrail: recursive reference detected (%s)", stackWithNext(state.stack, name))
	}

	saved, err := request.Load(name)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("loading request %q: %w", name, err)
	}

	state.iterations++
	state.visits[name]++
	state.stack = append(state.stack, name)
	defer func() {
		state.stack = state.stack[:len(state.stack)-1]
	}()

	executed := 1
	capturedCount := 0

	for i, hook := range saved.PreRequest {
		hookExecuted, hookCaptured, hookErr := e.runHook(name, "pre_request", i, hook, ctx, envOverride, state)
		executed += hookExecuted
		capturedCount += hookCaptured
		if hookErr != nil {
			return nil, executed, capturedCount, hookErr
		}
	}

	resp, err := e.execute(name, ctx.Snapshot(), envOverride)
	if err != nil {
		return nil, executed, capturedCount, fmt.Errorf("request %q failed: %w", name, err)
	}
	if resp == nil {
		return nil, executed, capturedCount, fmt.Errorf("request %q failed: empty response", name)
	}
	if resp.StatusCode >= 400 {
		return resp, executed, capturedCount, fmt.Errorf("request %q returned HTTP %d %s", name, resp.StatusCode, resp.Status)
	}

	captured, err := runner.CaptureVariables(saved.Capture, resp)
	if err != nil {
		return resp, executed, capturedCount, fmt.Errorf("request %q capture failed: %w", name, err)
	}
	if len(captured) > 0 {
		ctx.Merge(captured)
		capturedCount += len(captured)
	}

	for i, hook := range saved.PostRequest {
		hookExecuted, hookCaptured, hookErr := e.runHook(name, "post_request", i, hook, ctx, envOverride, state)
		executed += hookExecuted
		capturedCount += hookCaptured
		if hookErr != nil {
			return resp, executed, capturedCount, hookErr
		}
	}

	return resp, executed, capturedCount, nil
}

func (e *Executor) runHook(parentName, phase string, index int, hook request.Hook, ctx *runner.RuntimeContext, envOverride string, state *executionState) (int, int, error) {
	hookName := strings.TrimSpace(hook.Run)
	if hookName == "" {
		return 0, 0, fmt.Errorf("%s hook #%d in %q is missing run target", phase, index+1, parentName)
	}

	resp, executed, capturedCount, err := e.runRequest(hookName, ctx, envOverride, state)
	if err != nil {
		return executed, capturedCount, fmt.Errorf("%s hook #%d (%q) in %q failed: %w", phase, index+1, hookName, parentName, err)
	}

	captured, err := runner.CaptureVariables(hook.Capture, resp)
	if err != nil {
		return executed, capturedCount, fmt.Errorf("%s hook #%d (%q) in %q capture failed: %w", phase, index+1, hookName, parentName, err)
	}
	if len(captured) > 0 {
		ctx.Merge(captured)
		capturedCount += len(captured)
	}

	return executed, capturedCount, nil
}

func stackContains(stack []string, name string) bool {
	for _, v := range stack {
		if v == name {
			return true
		}
	}
	return false
}

func stackWithNext(stack []string, next string) string {
	path := make([]string, 0, len(stack)+1)
	path = append(path, stack...)
	path = append(path, next)
	return strings.Join(path, " -> ")
}
