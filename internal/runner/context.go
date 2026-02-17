package runner

type RuntimeContext struct {
	vars map[string]string
}

func NewRuntimeContext(initialVars map[string]string) *RuntimeContext {
	ctx := &RuntimeContext{vars: make(map[string]string)}
	ctx.Merge(initialVars)
	return ctx
}

func (c *RuntimeContext) Snapshot() map[string]string {
	out := make(map[string]string, len(c.vars))
	for k, v := range c.vars {
		out[k] = v
	}
	return out
}

func (c *RuntimeContext) Merge(values map[string]string) {
	for k, v := range values {
		c.vars[k] = v
	}
}
