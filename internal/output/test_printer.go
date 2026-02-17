package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Tresor-Kasend/apix/internal/tester"
)

func PrintTestResult(result tester.RequestResult) {
	ms := durationMs(result.Duration)

	if result.Passed {
		green.Printf("  [PASS] %s", result.Name)
		gray.Printf(" (%.0fms)\n", ms)
		return
	}

	red.Printf("  [FAIL] %s", result.Name)
	gray.Printf(" (%.0fms)\n", ms)

	if result.Error != "" {
		fmt.Printf("    error: %s\n", result.Error)
	}
	for _, failure := range result.Failures {
		fmt.Printf(
			"    - %s %s expected=%s actual=%s (%s)\n",
			failure.Target,
			failure.Operator,
			formatTestValue(failure.Expected),
			formatTestValue(failure.Actual),
			failure.Message,
		)
	}
}

func PrintTestSummary(suite tester.SuiteResult) {
	fmt.Println()
	if suite.Failed == 0 {
		success.Printf("  [PASS] passed=%d failed=%d total=%d duration=%.0fms\n",
			suite.Passed, suite.Failed, suite.Total, durationMs(suite.Duration))
		return
	}

	red.Printf("  [FAIL] passed=%d failed=%d total=%d duration=%.0fms\n",
		suite.Passed, suite.Failed, suite.Total, durationMs(suite.Duration))
}

func formatTestValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(encoded)
	}
}

func durationMs(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000.0
}
