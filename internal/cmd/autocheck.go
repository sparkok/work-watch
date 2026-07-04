package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"work-watch/internal/jobs"
	"work-watch/internal/task"
)

// autoCheckAndResume scans all tasks for stale .running markers and resumes
// incomplete tasks discovered on restart (crash/force-exit recovery).
func autoCheckAndResume() {
	taskNames, err := task.ListTasks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "auto-check: list tasks: %v\n", err)
		return
	}

	ctx := context.Background()

	for _, name := range taskNames {
		tDir := task.TaskDir(name)

		// Skip if task has no config (never configured)
		if _, err := os.Stat(task.ConfigPath(tDir)); os.IsNotExist(err) {
			continue
		}

		cfg, err := task.LoadConfig(tDir)
		if err != nil {
			continue
		}
		if cfg.PilotDeck.BaseURL == "" {
			continue
		}

		// Read .running marker: no marker → skip, fresh → skip, stale → recover
		marker, err := task.ReadRunningMarker(tDir)
		if err != nil || marker == nil {
			continue // no marker or read error → clean state
		}

		started, parseErr := time.Parse(time.RFC3339, marker.Started)
		if parseErr != nil || time.Since(started) < staleMarkerThreshold {
			continue // fresh marker or unparseable — still running
		}

		// Stale marker: remove it and proceed with recovery
		_ = task.RemoveRunningMarker(tDir)

		// Check if there are incomplete jobs
		nextJob, _ := task.NextIncomplete(tDir)

		if nextJob == "" {
			// All jobs done but may not be confirmed — auto-confirm
			ok, err := task.TryAcquireRunningMarker(tDir, cfg.SessionID)
			if !ok || err != nil {
				continue
			}
			go func(dir string, c *task.TaskConfig) {
				defer task.RemoveRunningMarker(dir)
				tryAutoConfirm(ctx, dir, c)
			}(tDir, cfg)
		} else {
			// Has incomplete jobs — acquire marker and resume execution
			ok, err := task.TryAcquireRunningMarker(tDir, cfg.SessionID)
			if !ok || err != nil {
				continue
			}
			go func(dir string, c *task.TaskConfig) {
				defer task.RemoveRunningMarker(dir)
				if err := jobs.RunTask(ctx, &jobs.RunOptions{
					TaskDir: dir,
					Cfg:     c,
					Quiet:   true,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "auto-resume '%s': %v\n", filepath.Base(dir), err)
					return
				}
				tryAutoConfirm(ctx, dir, c)
			}(tDir, cfg)
		}
	}
}

// tryAutoConfirm checks whether the task has been confirmed and, if not, runs
// the confirmation flow and marks confirmed on success.
func tryAutoConfirm(ctx context.Context, taskDir string, cfg *task.TaskConfig) {
	confirmed, err := task.IsConfirmed(taskDir)
	if err != nil || confirmed {
		return
	}
	ok, err := confirmTaskOutcome(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auto-confirm '%s': %v\n", filepath.Base(taskDir), err)
		return
	}
	if ok {
		_ = task.MarkConfirmed(taskDir)
	}
}
