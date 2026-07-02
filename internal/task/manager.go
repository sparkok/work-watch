package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	statusFileName = "status"
	jobsDirName    = "jobs"
	logsDirName    = "logs"
)

// TasksDir is the global tasks root directory.
var TasksDir = "tasks"

// ListTasks enumerates all task subdirectories under TasksDir.
func ListTasks() ([]string, error) {
	entries, err := os.ReadDir(TasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	var tasks []string
	for _, e := range entries {
		if e.IsDir() {
			tasks = append(tasks, e.Name())
		}
	}
	sort.Strings(tasks)
	return tasks, nil
}

// TaskDir returns the on-disk path for a named task.
func TaskDir(taskName string) string {
	return filepath.Join(TasksDir, taskName)
}

// ListJobs reads all *.txt files in the task's jobs/ directory, sorted.
func ListJobs(taskDir string) ([]string, error) {
	jobsDir := filepath.Join(taskDir, jobsDirName)
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	var jobs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
			jobs = append(jobs, e.Name())
		}
	}
	sort.Strings(jobs)
	return jobs, nil
}

// StatusData is the YAML structure for the task status file.
type StatusData struct {
	Completed map[string]string `yaml:"completed"`
	Confirmed bool              `yaml:"confirmed"`
}

func statusPath(taskDir string) string {
	return filepath.Join(taskDir, statusFileName)
}

func loadStatus(taskDir string) (*StatusData, error) {
	raw, err := os.ReadFile(statusPath(taskDir))
	if err != nil {
		if os.IsNotExist(err) {
			return &StatusData{Completed: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("read status: %w", err)
	}
	var s StatusData
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	if s.Completed == nil {
		s.Completed = map[string]string{}
	}
	return &s, nil
}

func saveStatus(taskDir string, s *StatusData) error {
	raw, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	return os.WriteFile(statusPath(taskDir), raw, 0o644)
}

// CompletedJobs returns the set of completed job identifiers (name without .txt).
func CompletedJobs(taskDir string) (map[string]bool, error) {
	s, err := loadStatus(taskDir)
	if err != nil {
		return nil, err
	}
	result := map[string]bool{}
	for k := range s.Completed {
		result[k] = true
	}
	return result, nil
}

// jobKey returns the job identifier stored in status (filename without extension).
func jobKey(jobName string) string {
	return strings.TrimSuffix(jobName, ".txt")
}

// NextIncomplete finds the first incomplete job filename.
// Returns "" when all jobs are complete.
func NextIncomplete(taskDir string) (string, error) {
	jobs, err := ListJobs(taskDir)
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return "", nil
	}
	completed, err := CompletedJobs(taskDir)
	if err != nil {
		return "", err
	}
	for _, j := range jobs {
		if !completed[jobKey(j)] {
			return j, nil
		}
	}
	return "", nil
}

// MarkCompleted records a job as completed with its session ID in the YAML status file.
func MarkCompleted(taskDir, jobName, sessionID string) error {
	s, err := loadStatus(taskDir)
	if err != nil {
		return err
	}
	s.Completed[jobKey(jobName)] = sessionID
	return saveStatus(taskDir, s)
}

// IsConfirmed checks whether the task has been confirmed with PilotDeck.
func IsConfirmed(taskDir string) (bool, error) {
	s, err := loadStatus(taskDir)
	if err != nil {
		return false, err
	}
	return s.Confirmed, nil
}

// MarkConfirmed marks the task as confirmed in the YAML status file.
func MarkConfirmed(taskDir string) error {
	s, err := loadStatus(taskDir)
	if err != nil {
		return err
	}
	s.Confirmed = true
	return saveStatus(taskDir, s)
}
