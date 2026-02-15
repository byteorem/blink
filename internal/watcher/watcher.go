// Package watcher provides debounced filesystem event monitoring.
package watcher

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/byteorem/blink/internal/copier"
	"github.com/fsnotify/fsnotify"
)

// Op represents a filesystem operation type.
type Op int

// Filesystem operation types.
const (
	OpCreate Op = iota
	OpWrite
	OpRemove
	OpRename
)

// Event represents a debounced filesystem change.
type Event struct {
	RelPath string
	Op      Op
}

// Watch starts watching srcDir for changes, returning debounced events on a channel.
func Watch(ctx context.Context, srcDir string, ig *copier.Ignorer) (<-chan Event, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Add all existing subdirectories
	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		if rel != "." && ig.ShouldIgnore(rel) {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
	if err != nil {
		_ = w.Close()
		return nil, err
	}

	ch := make(chan Event, 64)

	go func() {
		defer func() { _ = w.Close() }()
		defer close(ch)

		const debounce = 50 * time.Millisecond
		pending := make(map[string]Event)
		var timer *time.Timer
		var timerC <-chan time.Time

		flush := func() {
			for _, ev := range pending {
				ch <- ev
			}
			pending = make(map[string]Event)
			timer = nil
			timerC = nil
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-timerC:
				flush()
			case ev, ok := <-w.Events:
				if !ok {
					return
				}

				rel, err := filepath.Rel(srcDir, ev.Name)
				if err != nil || rel == "." {
					continue
				}

				if ig.ShouldIgnore(rel) {
					continue
				}

				var op Op
				switch {
				case ev.Has(fsnotify.Create):
					op = OpCreate
					// If new directory, add to watcher
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						_ = w.Add(ev.Name)
					}
				case ev.Has(fsnotify.Write):
					op = OpWrite
				case ev.Has(fsnotify.Remove):
					op = OpRemove
				case ev.Has(fsnotify.Rename):
					op = OpRename
				default:
					continue
				}

				pending[rel] = Event{RelPath: rel, Op: op}

				if timer == nil {
					timer = time.NewTimer(debounce)
					timerC = timer.C
				} else {
					timer.Reset(debounce)
				}

			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return ch, nil
}
