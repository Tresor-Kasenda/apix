package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunTriggersOnFileEvent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "requests"), 0o755); err != nil {
		t.Fatalf("creating requests dir: %v", err)
	}
	target := filepath.Join(dir, "requests", "login.yaml")
	if err := os.WriteFile(target, []byte("name: login\nmethod: GET\npath: /login\n"), 0o644); err != nil {
		t.Fatalf("writing target file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	triggers := make(chan Trigger, 16)
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, WatcherOptions{
			Path:     target,
			Debounce: 20 * time.Millisecond,
		}, func(trigger Trigger) {
			select {
			case triggers <- trigger:
			default:
			}
			if trigger.Reason == "fs-event" {
				cancel()
			}
		})
	}()

	if _, ok := waitForReason(t, triggers, "initial", 2*time.Second); !ok {
		t.Fatal("expected initial trigger")
	}

	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(target, []byte("name: login\nmethod: GET\npath: /login\n# changed\n"), 0o644); err != nil {
		t.Fatalf("updating target file: %v", err)
	}

	if _, ok := waitForReason(t, triggers, "fs-event", 3*time.Second); !ok {
		t.Fatal("expected fs-event trigger after file change")
	}

	if err := <-done; err != nil {
		t.Fatalf("watch run failed: %v", err)
	}
}

func TestRunTriggersOnPollingChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "requests"), 0o755); err != nil {
		t.Fatalf("creating requests dir: %v", err)
	}
	target := filepath.Join(dir, "requests", "login.yaml")
	if err := os.WriteFile(target, []byte("name: login\nmethod: GET\npath: /login\n"), 0o644); err != nil {
		t.Fatalf("writing target file: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	triggers := make(chan Trigger, 16)
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, WatcherOptions{
			Path:     target,
			Interval: 50 * time.Millisecond,
		}, func(trigger Trigger) {
			select {
			case triggers <- trigger:
			default:
			}
			if trigger.Reason == "poll-change" {
				cancel()
			}
		})
	}()

	if _, ok := waitForReason(t, triggers, "initial", 2*time.Second); !ok {
		t.Fatal("expected initial trigger")
	}

	time.Sleep(120 * time.Millisecond)
	if err := os.WriteFile(target, []byte("name: login\nmethod: GET\npath: /login\n# changed via poll\n"), 0o644); err != nil {
		t.Fatalf("updating target file: %v", err)
	}

	trigger, ok := waitForReason(t, triggers, "poll-change", 3*time.Second)
	if !ok {
		t.Fatal("expected poll-change trigger after file change")
	}
	if trigger.Path == "" {
		t.Fatal("expected trigger path to be set")
	}

	if err := <-done; err != nil {
		t.Fatalf("watch run failed: %v", err)
	}
}

func waitForReason(t *testing.T, ch <-chan Trigger, reason string, timeout time.Duration) (Trigger, bool) {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case <-deadline.C:
			return Trigger{}, false
		case trigger := <-ch:
			if trigger.Reason == reason {
				return trigger, true
			}
		}
	}
}
