package task

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// CompletedJobs returns the set of completed job identifiers (name without .txt).
func CompletedJobs(taskDir string) (map[string]bool, error) {
	raw, err := os.ReadFile(filepath.Join(taskDir, statusFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("read status: %w", err)
	}
	completed := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			completed[line] = true
		}
	}
	return completed, sc.Err()
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

// MarkCompleted appends a job identifier to the status file.
func MarkCompleted(taskDir, jobName string) error {
	f, err := os.OpenFile(filepath.Join(taskDir, statusFileName),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open status: %w", err)
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, jobKey(jobName)); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	return nil
}
