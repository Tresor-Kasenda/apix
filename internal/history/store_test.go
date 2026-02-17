package history

import (
	"os"
	"testing"
	"time"
)

func TestAppendReadLimitClear(t *testing.T) {
	withTempDirAsWorkingDirHistory(t)

	base := time.Date(2026, 2, 17, 10, 0, 0, 0, time.UTC)
	if err := Append(Entry{Method: "GET", Path: "/one", Status: 200, DurationMS: 10, Timestamp: base}); err != nil {
		t.Fatalf("append #1: %v", err)
	}
	if err := Append(Entry{Method: "POST", Path: "/two", Status: 201, DurationMS: 15, Timestamp: base.Add(time.Second)}); err != nil {
		t.Fatalf("append #2: %v", err)
	}
	if err := Append(Entry{Method: "DELETE", Path: "/three", Status: 204, DurationMS: 20, Timestamp: base.Add(2 * time.Second)}); err != nil {
		t.Fatalf("append #3: %v", err)
	}

	latestTwo, err := Read(2)
	if err != nil {
		t.Fatalf("read limit: %v", err)
	}
	if len(latestTwo) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(latestTwo))
	}
	if latestTwo[0].Path != "/three" || latestTwo[1].Path != "/two" {
		t.Fatalf("expected reverse chronological order, got %+v", latestTwo)
	}

	all, err := Read(0)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}

	if err := Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	afterClear, err := Read(10)
	if err != nil {
		t.Fatalf("read after clear: %v", err)
	}
	if len(afterClear) != 0 {
		t.Fatalf("expected empty history after clear, got %d entries", len(afterClear))
	}
}

func TestReadNoHistoryFile(t *testing.T) {
	withTempDirAsWorkingDirHistory(t)

	entries, err := Read(10)
	if err != nil {
		t.Fatalf("read no file: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func withTempDirAsWorkingDirHistory(t *testing.T) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}
}
