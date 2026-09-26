package request

import "testing"

func TestResolveVariables(t *testing.T) {
	t.Setenv("APIX_TEST_FROM_OS", "os-value")
	t.Setenv("APIX_TEST_SHADOWED", "os-shadowed")

	vars := map[string]string{
		"ID":                 "42",
		"APIX_TEST_SHADOWED": "from-vars",
	}

	cases := map[string]string{
		"/users/${ID}":                   "/users/42",
		"${APIX_TEST_FROM_OS}":           "os-value",
		"${APIX_TEST_SHADOWED}":          "from-vars",
		"${APIX_TEST_MISSING:-fallback}": "fallback",
		"${APIX_TEST_MISSING:-}":         "",
		"${ID:-ignored}":                 "42",
		"${APIX_TEST_MISSING}":           "${APIX_TEST_MISSING}",
		"http://${APIX_TEST_H:-localhost}:${APIX_TEST_P:-80}": "http://localhost:80",
	}
	for input, want := range cases {
		if got := ResolveVariables(input, vars); got != want {
			t.Errorf("ResolveVariables(%q) = %q, want %q", input, got, want)
		}
	}
}
