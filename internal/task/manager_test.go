package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStatus_Continuous(t *testing.T) {
	dir := t.TempDir()
	statusContent := `session_id: web-s_test
confirmed: true
jobs:
  001_first:
    status: completed
  002_second:
    status: completed
`
	if err := os.WriteFile(filepath.Join(dir, statusFileName), []byte(statusContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadStatus(dir)
	if err != nil {
		t.Fatalf("LoadStatus failed: %v", err)
	}

	if s.SessionID != "web-s_test" {
		t.Errorf("expected session_id web-s_test, got %q", s.SessionID)
	}
	if !s.Confirmed {
		t.Error("expected confirmed=true")
	}
	if len(s.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(s.Jobs))
	}

	jn := s.Jobs["001_first"]
	if jn.Status != JobCompleted {
		t.Errorf("expected job 001_first status completed, got %q", jn.Status)
	}
	if jn.SessionID != "" {
		t.Errorf("expected empty session_id for continuous job, got %q", jn.SessionID)
	}
}

func TestLoadStatus_Discrete(t *testing.T) {
	dir := t.TempDir()
	statusContent := `confirmed: false
jobs:
  001_first:
    status: completed
    session_id: web-s_job1
  002_second:
    status: completed
    session_id: web-s_job2
`
	if err := os.WriteFile(filepath.Join(dir, statusFileName), []byte(statusContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadStatus(dir)
	if err != nil {
		t.Fatalf("LoadStatus failed: %v", err)
	}

	if s.SessionID != "" {
		t.Errorf("expected no session_id for discrete mode, got %q", s.SessionID)
	}

	jn := s.Jobs["001_first"]
	if jn.SessionID != "web-s_job1" {
		t.Errorf("expected session_id web-s_job1, got %q", jn.SessionID)
	}
}

func TestCompletedJobs(t *testing.T) {
	dir := t.TempDir()
	statusContent := `jobs:
  001_first:
    status: completed
  002_second:
    status: running
  003_third:
    status: submitted
`
	if err := os.WriteFile(filepath.Join(dir, statusFileName), []byte(statusContent), 0o644); err != nil {
		t.Fatal(err)
	}

	completed, err := CompletedJobs(dir)
	if err != nil {
		t.Fatalf("CompletedJobs failed: %v", err)
	}

	if len(completed) != 1 {
		t.Fatalf("expected 1 completed job, got %d", len(completed))
	}
	if !completed["001_first"] {
		t.Error("expected 001_first to be completed")
	}
}

func TestSetJobStatus(t *testing.T) {
	dir := t.TempDir()
	// Create empty status
	s := &StatusData{Jobs: map[string]JobNode{}}
	if err := saveStatus(dir, s); err != nil {
		t.Fatal(err)
	}

	if err := SetJobStatus(dir, "001_first.txt", JobSubmitted); err != nil {
		t.Fatalf("SetJobStatus failed: %v", err)
	}

	jn, err := GetJobNode(dir, "001_first.txt")
	if err != nil {
		t.Fatalf("GetJobNode failed: %v", err)
	}
	if jn == nil {
		t.Fatal("expected job node, got nil")
	}
	if jn.Status != JobSubmitted {
		t.Errorf("expected status submitted, got %q", jn.Status)
	}
}

func TestSetSessionID(t *testing.T) {
	dir := t.TempDir()
	s := &StatusData{Jobs: map[string]JobNode{}}
	if err := saveStatus(dir, s); err != nil {
		t.Fatal(err)
	}

	if err := SetSessionID(dir, "web-s_test"); err != nil {
		t.Fatalf("SetSessionID failed: %v", err)
	}

	loaded, err := LoadStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SessionID != "web-s_test" {
		t.Errorf("expected session_id web-s_test, got %q", loaded.SessionID)
	}
}

func TestMarkCompleted(t *testing.T) {
	dir := t.TempDir()
	s := &StatusData{Jobs: map[string]JobNode{}}
	if err := saveStatus(dir, s); err != nil {
		t.Fatal(err)
	}

	if err := MarkCompleted(dir, "001_first.txt", "web-s_job1"); err != nil {
		t.Fatalf("MarkCompleted failed: %v", err)
	}

	jn, err := GetJobNode(dir, "001_first.txt")
	if err != nil {
		t.Fatal(err)
	}
	if jn.Status != JobCompleted {
		t.Errorf("expected status completed, got %q", jn.Status)
	}
	if jn.SessionID != "web-s_job1" {
		t.Errorf("expected session_id web-s_job1, got %q", jn.SessionID)
	}
}
