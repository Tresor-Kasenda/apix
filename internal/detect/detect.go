package detect

type Framework struct {
	Name        string
	Language    string
	DefaultPort int
}

type DetectedRoute struct {
	Method string
	Path   string
}

type Result struct {
	Framework *Framework
	Routes    []DetectedRoute
}

func Detect(root string) Result {
	fw := detectFramework(root)
	if fw == nil {
		return Result{}
	}

	cfg := scanConfigFor(fw)
	routes := scanRoutes(root, cfg)

	return Result{
		Framework: fw,
		Routes:    routes,
	}
}
