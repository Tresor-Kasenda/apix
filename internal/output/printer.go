package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	green   = color.New(color.FgGreen, color.Bold)
	yellow  = color.New(color.FgYellow, color.Bold)
	red     = color.New(color.FgRed, color.Bold)
	cyan    = color.New(color.FgCyan)
	gray    = color.New(color.FgHiBlack)
	bold    = color.New(color.Bold)
	success = color.New(color.FgGreen)
)

func PrintStatus(method, path string, statusCode int, status string, duration time.Duration, bodySize int) {
	c := statusColor(statusCode)
	ms := float64(duration.Microseconds()) / 1000.0
	fmt.Println()
	c.Printf("  %s %s → %d %s", method, path, statusCode, statusText(status))
	gray.Printf(" (%.0fms, %s)\n", ms, formatBodySize(bodySize))
	fmt.Println()
}

func PrintHeaders(headers map[string][]string) {
	bold.Println("  Headers:")
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range headers[k] {
			cyan.Printf("    %s: ", k)
			fmt.Println(v)
		}
	}
	fmt.Println()
}

func PrintBody(body []byte, raw bool) {
	if len(body) == 0 {
		gray.Println("  (empty body)")
		return
	}

	if raw {
		fmt.Println(string(body))
		return
	}

	bold.Println("  Body:")
	var obj interface{}
	if err := json.Unmarshal(body, &obj); err == nil {
		pretty, err := json.MarshalIndent(obj, "  ", "  ")
		if err == nil {
			fmt.Println("  " + string(pretty))
			fmt.Println()
			return
		}
	}

	fmt.Println("  " + string(body))
	fmt.Println()
}

func PrintBodyRaw(body []byte) {
	if len(body) == 0 {
		return
	}
	fmt.Print(string(body))
}

func PrintTokenCaptured() {
	success.Println("  ✓ Token captured and saved")
	fmt.Println()
}

func PrintError(err error) {
	red.Fprintf(color.Error, "  Error: %s\n", err)
}

func PrintInfo(msg string) {
	fmt.Printf("  %s\n", msg)
}

func PrintSuccess(msg string) {
	success.Printf("  ✓ %s\n", msg)
}

func statusColor(code int) *color.Color {
	switch {
	case code >= 200 && code < 300:
		return green
	case code >= 300 && code < 400:
		return yellow
	default:
		return red
	}
}

func statusText(status string) string {
	parts := strings.SplitN(status, " ", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return status
}

func formatBodySize(size int) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024.0)
	}
	return fmt.Sprintf("%.1fMB", float64(size)/(1024.0*1024.0))
}
