package tester

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"sort"
	"strings"

	apixhttp "github.com/Tresor-Kasend/apix/internal/http"
	"github.com/Tresor-Kasend/apix/internal/request"
)

var operatorOrder = []string{
	"exists",
	"eq",
	"contains",
	"is_number",
	"is_string",
	"is_array",
	"is_bool",
	"is_null",
	"gt",
	"gte",
	"lt",
	"lte",
	"length",
}

var supportedOperators = map[string]struct{}{
	"exists":    {},
	"eq":        {},
	"contains":  {},
	"is_number": {},
	"is_string": {},
	"is_array":  {},
	"is_bool":   {},
	"is_null":   {},
	"gt":        {},
	"gte":       {},
	"lt":        {},
	"lte":       {},
	"length":    {},
}

func EvaluateExpect(expect *request.Expect, resp *apixhttp.Response) ([]AssertionFailure, error) {
	if expect == nil {
		return nil, nil
	}
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	failures := make([]AssertionFailure, 0)

	if len(expect.Status) > 0 {
		ruleFailures, err := evaluateRules("status", expect.Status, resp.StatusCode, true)
		if err != nil {
			return nil, err
		}
		failures = append(failures, ruleFailures...)
	}

	if len(expect.ResponseTime) > 0 {
		ruleFailures, err := evaluateRules("response_time", expect.ResponseTime, resp.Duration.Milliseconds(), true)
		if err != nil {
			return nil, err
		}
		failures = append(failures, ruleFailures...)
	}

	if len(expect.Headers) > 0 {
		keys := sortedAssertionRuleKeys(expect.Headers)
		for _, headerName := range keys {
			value, exists := getHeaderValue(resp.Headers, headerName)
			ruleFailures, err := evaluateRules("headers."+headerName, expect.Headers[headerName], value, exists)
			if err != nil {
				return nil, err
			}
			failures = append(failures, ruleFailures...)
		}
	}

	if len(expect.Body) > 0 {
		var root interface{}
		if err := json.Unmarshal(resp.Body, &root); err != nil {
			return nil, fmt.Errorf("body assertions require a JSON response: %w", err)
		}

		keys := sortedAssertionRuleKeys(expect.Body)
		for _, path := range keys {
			value, exists, err := extractJSONPath(root, path)
			if err != nil {
				return nil, fmt.Errorf("body.%s: %w", path, err)
			}
			ruleFailures, err := evaluateRules("body."+path, expect.Body[path], value, exists)
			if err != nil {
				return nil, err
			}
			failures = append(failures, ruleFailures...)
		}
	}

	return failures, nil
}

func evaluateRules(target string, rules request.AssertionRule, actual interface{}, exists bool) ([]AssertionFailure, error) {
	for op := range rules {
		if _, ok := supportedOperators[op]; !ok {
			return nil, fmt.Errorf("%s: unsupported operator %q", target, op)
		}
	}

	failures := make([]AssertionFailure, 0)

	for _, op := range operatorOrder {
		expected, ok := rules[op]
		if !ok {
			continue
		}

		switch op {
		case "exists":
			expectedBool, err := expectedBoolValue(target, op, expected)
			if err != nil {
				return nil, err
			}
			if exists != expectedBool {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedBool,
					Actual:   exists,
					Message:  "existence check failed",
				})
			}

		case "eq":
			if !exists {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expected,
					Actual:   nil,
					Message:  "value does not exist",
				})
				continue
			}
			if !valuesEqual(actual, expected) {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expected,
					Actual:   actual,
					Message:  "equality check failed",
				})
			}

		case "contains":
			if !exists {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expected,
					Actual:   nil,
					Message:  "value does not exist",
				})
				continue
			}
			if !containsValue(actual, expected) {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expected,
					Actual:   actual,
					Message:  "contains check failed",
				})
			}

		case "is_number":
			expectedBool, err := expectedBoolValue(target, op, expected)
			if err != nil {
				return nil, err
			}
			_, isNumber := toFloat64(actual)
			if isNumber != expectedBool {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedBool,
					Actual:   actual,
					Message:  "type check failed",
				})
			}

		case "is_string":
			expectedBool, err := expectedBoolValue(target, op, expected)
			if err != nil {
				return nil, err
			}
			_, isString := actual.(string)
			if isString != expectedBool {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedBool,
					Actual:   actual,
					Message:  "type check failed",
				})
			}

		case "is_array":
			expectedBool, err := expectedBoolValue(target, op, expected)
			if err != nil {
				return nil, err
			}
			value := reflect.ValueOf(actual)
			isArray := value.IsValid() && (value.Kind() == reflect.Array || value.Kind() == reflect.Slice)
			if isArray != expectedBool {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedBool,
					Actual:   actual,
					Message:  "type check failed",
				})
			}

		case "is_bool":
			expectedBool, err := expectedBoolValue(target, op, expected)
			if err != nil {
				return nil, err
			}
			_, isBool := actual.(bool)
			if isBool != expectedBool {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedBool,
					Actual:   actual,
					Message:  "type check failed",
				})
			}

		case "is_null":
			expectedBool, err := expectedBoolValue(target, op, expected)
			if err != nil {
				return nil, err
			}
			isNull := actual == nil
			if isNull != expectedBool {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedBool,
					Actual:   actual,
					Message:  "type check failed",
				})
			}

		case "gt", "gte", "lt", "lte":
			if !exists {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expected,
					Actual:   nil,
					Message:  "value does not exist",
				})
				continue
			}

			expectedNumber, ok := toFloat64(expected)
			if !ok {
				return nil, fmt.Errorf("%s.%s expects numeric value, got %T", target, op, expected)
			}
			actualNumber, ok := toFloat64(actual)
			if !ok {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedNumber,
					Actual:   actual,
					Message:  "actual value is not numeric",
				})
				continue
			}

			var pass bool
			switch op {
			case "gt":
				pass = actualNumber > expectedNumber
			case "gte":
				pass = actualNumber >= expectedNumber
			case "lt":
				pass = actualNumber < expectedNumber
			case "lte":
				pass = actualNumber <= expectedNumber
			}
			if !pass {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expectedNumber,
					Actual:   actualNumber,
					Message:  "numeric comparison failed",
				})
			}

		case "length":
			if !exists {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: expected,
					Actual:   nil,
					Message:  "value does not exist",
				})
				continue
			}

			expectedLengthFloat, ok := toFloat64(expected)
			if !ok || expectedLengthFloat < 0 || math.Trunc(expectedLengthFloat) != expectedLengthFloat {
				return nil, fmt.Errorf("%s.length expects a non-negative integer, got %v", target, expected)
			}

			actualLength, ok := valueLength(actual)
			if !ok {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: int(expectedLengthFloat),
					Actual:   actual,
					Message:  "actual value has no length",
				})
				continue
			}

			if actualLength != int(expectedLengthFloat) {
				failures = append(failures, AssertionFailure{
					Target:   target,
					Operator: op,
					Expected: int(expectedLengthFloat),
					Actual:   actualLength,
					Message:  "length check failed",
				})
			}
		}
	}

	return failures, nil
}

func expectedBoolValue(target, operator string, value interface{}) (bool, error) {
	boolVal, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s.%s expects boolean value, got %T", target, operator, value)
	}
	return boolVal, nil
}

func sortedAssertionRuleKeys(m map[string]request.AssertionRule) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func getHeaderValue(headers http.Header, name string) (string, bool) {
	for key, values := range headers {
		if strings.EqualFold(key, name) {
			return strings.Join(values, ", "), true
		}
	}
	return "", false
}

func extractJSONPath(root interface{}, path string) (interface{}, bool, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, false, fmt.Errorf("path cannot be empty")
	}

	current := root
	for _, segment := range strings.Split(trimmed, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, false, fmt.Errorf("path segment cannot be empty")
		}

		object, ok := current.(map[string]interface{})
		if !ok {
			return nil, false, nil
		}
		next, ok := object[segment]
		if !ok {
			return nil, false, nil
		}
		current = next
	}
	return current, true, nil
}

func containsValue(actual interface{}, expected interface{}) bool {
	switch value := actual.(type) {
	case string:
		return strings.Contains(value, fmt.Sprint(expected))
	case []string:
		for _, item := range value {
			if valuesEqual(item, expected) {
				return true
			}
		}
		return false
	case []interface{}:
		for _, item := range value {
			if valuesEqual(item, expected) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func valuesEqual(actual, expected interface{}) bool {
	actualNumber, actualIsNumber := toFloat64(actual)
	expectedNumber, expectedIsNumber := toFloat64(expected)
	if actualIsNumber && expectedIsNumber {
		return actualNumber == expectedNumber
	}
	return reflect.DeepEqual(actual, expected)
}

func toFloat64(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int8:
		return float64(number), true
	case int16:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint:
		return float64(number), true
	case uint8:
		return float64(number), true
	case uint16:
		return float64(number), true
	case uint32:
		return float64(number), true
	case uint64:
		return float64(number), true
	case float32:
		return float64(number), true
	case float64:
		return number, true
	case json.Number:
		parsed, err := number.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func valueLength(value interface{}) (int, bool) {
	switch v := value.(type) {
	case string:
		return len(v), true
	case []string:
		return len(v), true
	case []interface{}:
		return len(v), true
	case map[string]interface{}:
		return len(v), true
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return 0, false
		}
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return rv.Len(), true
		default:
			return 0, false
		}
	}
}
