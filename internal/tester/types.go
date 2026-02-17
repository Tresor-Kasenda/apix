package tester

import "time"

type AssertionFailure struct {
	Target   string
	Operator string
	Expected interface{}
	Actual   interface{}
	Message  string
}

type RequestResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Failures []AssertionFailure
	Error    string
}

type SuiteResult struct {
	Total    int
	Passed   int
	Failed   int
	Duration time.Duration
	Results  []RequestResult
}

func (s SuiteResult) ExitCode() int {
	if s.Failed > 0 {
		return 1
	}
	return 0
}
