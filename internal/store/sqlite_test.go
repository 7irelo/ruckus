package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreRunLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "ruckus.db")

	runStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("create sqlite store: %v", err)
	}
	defer func() {
		_ = runStore.Close()
	}()

	if err := runStore.Init(ctx); err != nil {
		t.Fatalf("init sqlite schema: %v", err)
	}

	started := time.Now().UTC()
	run := RunRecord{
		ID:                "run-test-1",
		Experiment:        "kill-container",
		Target:            "myapp",
		Status:            StatusRunning,
		StartedAt:         started,
		Duration:          30 * time.Second,
		Interval:          10 * time.Second,
		Apply:             true,
		UnsafeMaxDuration: false,
		Metadata: map[string]string{
			"target_id": "abc123",
		},
	}

	if err := runStore.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	stored, err := runStore.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if stored.ID != run.ID {
		t.Fatalf("unexpected run ID: got %s want %s", stored.ID, run.ID)
	}
	if stored.Metadata["target_id"] != "abc123" {
		t.Fatalf("expected target_id metadata to persist")
	}

	if err := runStore.MarkStopRequested(ctx, run.ID, true); err != nil {
		t.Fatalf("mark stop requested: %v", err)
	}
	stopRequested, err := runStore.IsStopRequested(ctx, run.ID)
	if err != nil {
		t.Fatalf("check stop requested: %v", err)
	}
	if !stopRequested {
		t.Fatalf("expected stop request to be true")
	}

	if err := runStore.UpdateRunMetadata(ctx, run.ID, map[string]string{"target_id": "xyz789"}); err != nil {
		t.Fatalf("update run metadata: %v", err)
	}

	if err := runStore.AddEvent(ctx, EventRecord{
		RunID:   run.ID,
		Time:    time.Now().UTC(),
		Level:   "info",
		Action:  "run.start",
		Result:  "ok",
		Target:  "myapp",
		Message: "started",
		Details: map[string]string{"k": "v"},
	}); err != nil {
		t.Fatalf("add event: %v", err)
	}

	ended := time.Now().UTC()
	if err := runStore.UpdateRunStatus(ctx, run.ID, StatusCompleted, ended, "completed"); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	listed, err := runStore.ListRuns(ctx, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 run, got %d", len(listed))
	}
	if listed[0].Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s", listed[0].Status)
	}
	if listed[0].Metadata["target_id"] != "xyz789" {
		t.Fatalf("expected updated metadata in listed run")
	}
}
