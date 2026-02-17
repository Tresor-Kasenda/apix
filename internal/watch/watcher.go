package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 150 * time.Millisecond

type Trigger struct {
	Reason string
	Path   string
	Time   time.Time
}

type WatcherOptions struct {
	Path     string
	Interval time.Duration
	Debounce time.Duration
}

type fileState struct {
	exists  bool
	modTime int64
	size    int64
}

func Run(ctx context.Context, opts WatcherOptions, onTrigger func(Trigger)) error {
	if onTrigger == nil {
		return fmt.Errorf("watch trigger callback is required")
	}

	targetPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return fmt.Errorf("resolving watch path %q: %w", opts.Path, err)
	}
	targetPath = filepath.Clean(targetPath)

	if opts.Interval < 0 {
		return fmt.Errorf("watch interval must be >= 0")
	}
	if opts.Debounce <= 0 {
		opts.Debounce = defaultDebounce
	}

	onTrigger(Trigger{
		Reason: "initial",
		Path:   targetPath,
		Time:   time.Now(),
	})

	if opts.Interval > 0 {
		return runPolling(ctx, targetPath, opts.Interval, onTrigger)
	}
	return runFSNotify(ctx, targetPath, opts.Debounce, onTrigger)
}

func runFSNotify(ctx context.Context, targetPath string, debounce time.Duration, onTrigger func(Trigger)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating fs watcher: %w", err)
	}
	defer watcher.Close()

	dir := filepath.Dir(targetPath)
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("watching directory %q: %w", dir, err)
	}

	var lastTrigger time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			if err != nil {
				return fmt.Errorf("watch error: %w", err)
			}
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !isTargetEvent(event, targetPath) {
				continue
			}

			now := time.Now()
			if !lastTrigger.IsZero() && now.Sub(lastTrigger) < debounce {
				continue
			}
			lastTrigger = now

			onTrigger(Trigger{
				Reason: "fs-event",
				Path:   normalizePath(event.Name),
				Time:   now,
			})
		}
	}
}

func runPolling(ctx context.Context, targetPath string, interval time.Duration, onTrigger func(Trigger)) error {
	state, err := readFileState(targetPath)
	if err != nil {
		return fmt.Errorf("stat watch target %q: %w", targetPath, err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			nextState, err := readFileState(targetPath)
			if err != nil {
				return fmt.Errorf("stat watch target %q: %w", targetPath, err)
			}
			if nextState == state {
				continue
			}
			state = nextState

			onTrigger(Trigger{
				Reason: "poll-change",
				Path:   targetPath,
				Time:   now,
			})
		}
	}
}

func isTargetEvent(event fsnotify.Event, targetPath string) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod) == 0 {
		return false
	}
	return normalizePath(event.Name) == targetPath
}

func normalizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func readFileState(path string) (fileState, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileState{exists: false}, nil
		}
		return fileState{}, err
	}

	return fileState{
		exists:  true,
		modTime: info.ModTime().UnixNano(),
		size:    info.Size(),
	}, nil
}
