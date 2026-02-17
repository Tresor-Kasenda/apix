package env

import (
	"fmt"
	"os"
	"strings"
)

func EnsureGitignoreEntry(entry string) error {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return fmt.Errorf("gitignore entry cannot be empty")
	}

	const gitignorePath = ".gitignore"
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", gitignorePath, err)
		}
		if writeErr := os.WriteFile(gitignorePath, []byte(entry+"\n"), 0o644); writeErr != nil {
			return fmt.Errorf("writing %s: %w", gitignorePath, writeErr)
		}
		return nil
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"

	if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", gitignorePath, err)
	}
	return nil
}
