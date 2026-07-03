package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunningMarker_WriteReadRemove(t *testing.T) {
	dir := t.TempDir()

	// Initially, no marker
	m, err := ReadRunningMarker(dir)
	if err != nil {
		t.Fatalf("ReadRunningMarker error: %v", err)
	}
	if m != nil {
		t.Fatal("expected nil marker when file doesn't exist")
	}

	// Write marker
	if err := WriteRunningMarker(dir, "sess-123"); err != nil {
		t.Fatalf("WriteRunningMarker error: %v", err)
	}

	// Marker file should exist
	markerPath := filepath.Join(dir, ".running")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Fatal(".running file not created")
	}

	// Read it back
	m, err = ReadRunningMarker(dir)
	if err != nil {
		t.Fatalf("ReadRunningMarker error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil marker")
	}
	if m.SessionID != "sess-123" {
		t.Errorf("expected session_id sess-123, got %q", m.SessionID)
	}
	if _, err := time.Parse(time.RFC3339, m.Started); err != nil {
		t.Errorf("started is not valid RFC3339: %q (%v)", m.Started, err)
	}

	// Update marker
	if err := WriteRunningMarker(dir, "sess-456"); err != nil {
		t.Fatalf("WriteRunningMarker (update) error: %v", err)
	}
	m, _ = ReadRunningMarker(dir)
	if m.SessionID != "sess-456" {
		t.Errorf("expected updated session_id sess-456, got %q", m.SessionID)
	}

	// Remove marker
	if err := RemoveRunningMarker(dir); err != nil {
		t.Fatalf("RemoveRunningMarker error: %v", err)
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatal(".running file should be removed")
	}

	// Remove non-existent is a no-op
	if err := RemoveRunningMarker(dir); err != nil {
		t.Fatalf("RemoveRunningMarker on missing file should not error: %v", err)
	}
}

func TestRunningMarker_AtomicWrite(t *testing.T) {
	dir := t.TempDir()

	if err := WriteRunningMarker(dir, "abc"); err != nil {
		t.Fatal(err)
	}

	// No .tmp leftover
	tmpPath := filepath.Join(dir, ".running.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error(".running.tmp should not exist after atomic write")
	}

	// Marker content is correct
	m, _ := ReadRunningMarker(dir)
	if m.SessionID != "abc" {
		t.Errorf("expected abc, got %q", m.SessionID)
	}
}

func TestTryAcquireRunningMarker(t *testing.T) {
	dir := t.TempDir()

	// First acquire should succeed
	acquired, err := TryAcquireRunningMarker(dir, "sess-1")
	if err != nil {
		t.Fatalf("first TryAcquireRunningMarker error: %v", err)
	}
	if !acquired {
		t.Fatal("expected first acquire to succeed")
	}

	// Marker file should exist
	markerPath := filepath.Join(dir, ".running")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Fatal(".running file not created by TryAcquireRunningMarker")
	}

	// Read it back — content should match
	m, err := ReadRunningMarker(dir)
	if err != nil {
		t.Fatalf("ReadRunningMarker error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil marker after acquire")
	}
	if m.SessionID != "sess-1" {
		t.Errorf("expected session_id sess-1, got %q", m.SessionID)
	}

	// Second acquire should fail (file already exists)
	acquired, err = TryAcquireRunningMarker(dir, "sess-2")
	if err != nil {
		t.Fatalf("second TryAcquireRunningMarker error: %v", err)
	}
	if acquired {
		t.Fatal("expected second acquire to fail (file exists)")
	}

	// Remove marker
	if err := RemoveRunningMarker(dir); err != nil {
		t.Fatalf("RemoveRunningMarker error: %v", err)
	}

	// After removal, acquire should succeed again
	acquired, err = TryAcquireRunningMarker(dir, "sess-3")
	if err != nil {
		t.Fatalf("third TryAcquireRunningMarker error: %v", err)
	}
	if !acquired {
		t.Fatal("expected third acquire to succeed after removal")
	}
	m, _ = ReadRunningMarker(dir)
	if m.SessionID != "sess-3" {
		t.Errorf("expected session_id sess-3, got %q", m.SessionID)
	}
}
