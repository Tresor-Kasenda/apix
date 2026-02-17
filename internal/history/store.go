package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const filePath = ".apix/history.jsonl"

type Entry struct {
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	Status       int       `json:"status"`
	DurationMS   int64     `json:"duration_ms"`
	ResponseSize int       `json:"response_size"`
	Timestamp    time.Time `json:"timestamp"`
}

func Append(entry Entry) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encoding history entry: %w", err)
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening history file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing history entry: %w", err)
	}

	return nil
}

func Read(limit int) ([]Entry, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading history file: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	entries := make([]Entry, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing history entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning history file: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	result := make([]Entry, 0, limit)
	for i := len(entries) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, entries[i])
	}

	return result, nil
}

func Clear() error {
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("clearing history: %w", err)
	}
	return nil
}
