package tester

import (
	"net/http"
	"strings"
	"testing"
	"time"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

func TestEvaluateExpectOperators(t *testing.T) {
	resp := &apixhttp.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Duration:   150 * time.Millisecond,
		Headers: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
			"X-Trace-ID":   []string{"trace-123"},
		},
		Body: []byte(`{"data":{"token":"abc123","ids":[1,2,3],"active":true,"meta":null,"count":3,"name":"Alice"}}`),
	}

	tests := []struct {
		name   string
		expect *request.Expect
	}{
		{
			name: "exists",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.token": {"exists": true},
				},
			},
		},
		{
			name: "eq",
			expect: &request.Expect{
				Status: request.AssertionRule{"eq": 200},
			},
		},
		{
			name: "contains",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.token": {"contains": "abc"},
				},
			},
		},
		{
			name: "is_number",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.count": {"is_number": true},
				},
			},
		},
		{
			name: "is_string",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.name": {"is_string": true},
				},
			},
		},
		{
			name: "is_array",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.ids": {"is_array": true},
				},
			},
		},
		{
			name: "is_bool",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.active": {"is_bool": true},
				},
			},
		},
		{
			name: "is_null",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.meta": {"is_null": true},
				},
			},
		},
		{
			name: "gt",
			expect: &request.Expect{
				Status: request.AssertionRule{"gt": 199},
			},
		},
		{
			name: "gte",
			expect: &request.Expect{
				Status: request.AssertionRule{"gte": 200},
			},
		},
		{
			name: "lt",
			expect: &request.Expect{
				ResponseTime: request.AssertionRule{"lt": 300},
			},
		},
		{
			name: "lte",
			expect: &request.Expect{
				ResponseTime: request.AssertionRule{"lte": 150},
			},
		},
		{
			name: "length",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.ids": {"length": 3},
				},
			},
		},
		{
			name: "header_exists_contains_eq_case_insensitive",
			expect: &request.Expect{
				Headers: map[string]request.AssertionRule{
					"content-type": {"exists": true, "contains": "application/json"},
					"x-trace-id":   {"eq": "trace-123"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			failures, err := EvaluateExpect(tc.expect, resp)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if len(failures) != 0 {
				t.Fatalf("expected no failures, got %+v", failures)
			}
		})
	}
}

func TestEvaluateExpectInvalidTypesAndMessages(t *testing.T) {
	resp := &apixhttp.Response{
		StatusCode: 200,
		Duration:   10 * time.Millisecond,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"data":{"count":3}}`),
	}

	tests := []struct {
		name        string
		expect      *request.Expect
		wantErrText string
	}{
		{
			name: "exists_non_bool",
			expect: &request.Expect{
				Body: map[string]request.AssertionRule{
					"data.count": {"exists": "yes"},
				},
			},
			wantErrText: "expects boolean value",
		},
		{
			name: "gt_non_numeric",
			expect: &request.Expect{
				Status: request.AssertionRule{"gt": "high"},
			},
			wantErrText: "expects numeric value",
		},
		{
			name: "unknown_operator",
			expect: &request.Expect{
				Status: request.AssertionRule{"nope": 1},
			},
			wantErrText: "unsupported operator",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EvaluateExpect(tc.expect, resp)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErrText) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErrText, err)
			}
		})
	}
}

func TestEvaluateExpectBodyRequiresJSON(t *testing.T) {
	resp := &apixhttp.Response{
		StatusCode: 200,
		Body:       []byte("not-json"),
	}
	expect := &request.Expect{
		Body: map[string]request.AssertionRule{
			"data.token": {"exists": true},
		},
	}

	_, err := EvaluateExpect(expect, resp)
	if err == nil {
		t.Fatalf("expected error for non-json body")
	}
	if !strings.Contains(err.Error(), "require a JSON response") {
		t.Fatalf("expected json error, got %v", err)
	}
}
