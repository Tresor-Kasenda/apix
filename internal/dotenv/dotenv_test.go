package dotenv

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := `
# Laravel-style file
APP_NAME="My App"
APP_URL=http://localhost:8000
export PORT=3000
API_URL=${APP_URL}/api
SECRET='raw ${APP_URL}'
DEBUG=true # inline comment
MULTI="line1\nline2"
EMPTY=
not a valid line
1INVALID=x
`
	values, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	want := map[string]string{
		"APP_NAME": "My App",
		"APP_URL":  "http://localhost:8000",
		"PORT":     "3000",
		"API_URL":  "http://localhost:8000/api",
		"SECRET":   "raw ${APP_URL}",
		"DEBUG":    "true",
		"MULTI":    "line1\nline2",
		"EMPTY":    "",
	}
	for k, v := range want {
		if got, ok := values[k]; !ok || got != v {
			t.Errorf("%s: want %q, got %q (present=%v)", k, v, got, ok)
		}
	}
	if _, ok := values["1INVALID"]; ok {
		t.Errorf("invalid key should be ignored")
	}
	if len(values) != len(want) {
		t.Errorf("unexpected keys: %v", values)
	}
}
