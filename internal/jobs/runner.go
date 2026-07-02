package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"work-watch/internal/pilotdeck"
	"work-watch/internal/task"
)

// RunOptions configures a task execution.
type RunOptions struct {
	TaskDir   string
	Debug     bool
	Cfg       *task.TaskConfig
	OnJobDone func(jobName string, resp *pilotdeck.AgentResponse)
}

// RunTask repeatedly executes the next incomplete job until all are done
// or an error occurs.
func RunTask(ctx context.Context, opts *RunOptions) error {
	sessionID := ""
	completed := 0

	for {
		jobName, err := task.NextIncomplete(opts.TaskDir)
		if err != nil {
			return fmt.Errorf("find next job: %w", err)
		}
		if jobName == "" {
			if completed == 0 {
				done, _ := task.CompletedJobs(opts.TaskDir)
				if len(done) > 0 {
					fmt.Printf("All %d job(s) already completed. To re-run, delete the status file.\n", len(done))
				} else {
					fmt.Printf("No job files found in jobs/ directory.\n")
				}
			}
			return nil
		}

		// Read job file
		jobPath := filepath.Join(opts.TaskDir, "jobs", jobName)
		msg, err := os.ReadFile(jobPath)
		if err != nil {
			return fmt.Errorf("read job %s: %w", jobName, err)
		}

		fmt.Printf("Running job: %s\n", jobName)

		// Submit
		resp, err := pilotdeck.SubmitMessage(
			ctx,
			opts.Cfg.PilotDeck.BaseURL,
			opts.Cfg.PilotDeck.APIKey,
			opts.Cfg.PilotDeck.ProjectPath,
			string(msg),
			sessionID,
		)
		if err != nil {
			// Log the failed attempt before returning error
			_ = writeJobLog(opts.TaskDir, jobName, string(msg), nil, err.Error(), nil)

			return fmt.Errorf("network error on %s: %w", jobName, err)
		}
		// Capture session ID from first job
		if sessionID == "" && resp.SessionID != "" {
			sessionID = resp.SessionID
		}

		// Fetch full conversation messages after the job completes
		var messagesJSON []byte
		if resp.SessionID != "" {
			mctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			messagesJSON, _ = pilotdeck.FetchSessionMessages(
				mctx,
				opts.Cfg.PilotDeck.BaseURL,
				opts.Cfg.PilotDeck.APIKey,
				resp.SessionID,
				opts.Cfg.PilotDeck.ProjectPath,
			)
			cancel()
		}

		// Save session ID to task config for later message retrieval
		if sessionID != "" {
			if err := task.SaveSessionID(opts.TaskDir, sessionID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save session: %v\n", err)
			}
		}

		// Always log interaction (with messages if available)
		_ = writeJobLog(opts.TaskDir, jobName, string(msg), resp, "", messagesJSON)

		// Mark completed
		if err := task.MarkCompleted(opts.TaskDir, jobName, sessionID); err != nil {
			return fmt.Errorf("mark completed %s: %w", jobName, err)
		}

		completed++
		if opts.OnJobDone != nil {
			opts.OnJobDone(jobName, resp)
		}
	}
}

// errMsg is non-empty only on failure (resp may be nil).
// messagesJSON is the session conversation fetched via GET /api/sessions/.../messages (may be nil).
func writeJobLog(taskDir, jobName, msg string, resp *pilotdeck.AgentResponse, errMsg string, messagesJSON []byte) error {
	logDir := filepath.Join(taskDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}

	ts := time.Now().Format("20060102_150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", jobKey(jobName), ts))

	reqJSON := fmt.Sprintf(`{"projectPath":"%s","message":%q,"stream":false}`,
		"<redacted>", msg)

	var respSection string
	if resp != nil {
		statusLine := "HTTP 200"
		var body string
		if len(resp.RawResponse) > 0 {
			body = string(resp.RawResponse)
		} else {
			body = fmt.Sprintf(`{"success":%v,"sessionId":%q,"error":%q}`,
				resp.Success, resp.SessionID, resp.Error)
		}
		respSection = fmt.Sprintf("=== RESPONSE ===\n%s\n%s\n", statusLine, body)
		if len(messagesJSON) > 0 {
			respSection += fmt.Sprintf("\n=== SESSION MESSAGES ===\n%s\n", string(messagesJSON))
		}
	} else {
		respSection = fmt.Sprintf("=== RESPONSE ===\nHTTP ERROR\n%s\n",
			fmt.Sprintf(`{"error":%q}`, errMsg))
	}

	content := fmt.Sprintf(`=== REQUEST ===
POST /api/agent
%s
%s`, reqJSON, respSection)

	return os.WriteFile(logPath, []byte(content), 0o644)
}

func jobKey(jobName string) string {
	return jobName[:len(jobName)-len(".txt")]
}
