package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/Tresor-Kasend/apix/internal/history"
)

func TestHistoryCommandLimit(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	if err := history.Append(history.Entry{
		Method:     "GET",
		Path:       "/first",
		Status:     200,
		DurationMS: 12,
		Timestamp:  time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := history.Append(history.Entry{
		Method:     "POST",
		Path:       "/second",
		Status:     201,
		DurationMS: 14,
		Timestamp:  time.Date(2026, 2, 17, 10, 0, 1, 0, time.UTC),
	}); err != nil {
		t.Fatalf("append second: %v", err)
	}

	cmd := newHistoryCmd()
	cmd.SetArgs([]string{"--limit", "1"})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute history --limit: %v", err)
		}
	})

	if !strings.Contains(out, "/second") {
		t.Fatalf("expected output to contain newest entry, got:\n%s", out)
	}
	if strings.Contains(out, "/first") {
		t.Fatalf("expected output not to contain older entry, got:\n%s", out)
	}
}

func TestHistoryCommandClear(t *testing.T) {
	withTempDirAsWorkingDirCLI(t)

	if err := history.Append(history.Entry{
		Method:     "GET",
		Path:       "/resource",
		Status:     200,
		DurationMS: 10,
	}); err != nil {
		t.Fatalf("append history: %v", err)
	}

	cmd := newHistoryCmd()
	cmd.SetArgs([]string{"--clear"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute history --clear: %v", err)
	}

	entries, err := history.Read(10)
	if err != nil {
		t.Fatalf("read history after clear: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries after clear, got %d", len(entries))
	}
}
