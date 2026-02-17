package request

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var varPattern = regexp.MustCompile(`\$\{(\w+)\}`)

func ResolveVariables(input string, vars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(input, func(match string) string {
		key := varPattern.FindStringSubmatch(match)[1]
		if val, ok := vars[key]; ok {
			return val
		}
		return match
	})
}

func BuildVariableMap(envVars map[string]string, token string, flagVars map[string]string) map[string]string {
	vars := make(map[string]string)

	for k, v := range envVars {
		vars[k] = v
	}

	if token != "" {
		vars["TOKEN"] = token
	}

	vars["TIMESTAMP"] = strconv.FormatInt(time.Now().Unix(), 10)
	vars["UUID"] = generateUUID()
	vars["RANDOM"] = generateRandom()

	for k, v := range flagVars {
		vars[k] = v
	}

	return vars
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func generateRandom() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x", b)
}
