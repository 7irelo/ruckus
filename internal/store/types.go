package store

import (
	"context"
	"errors"
	"time"
)

const (
	DefaultStatusLimit = 50
)

var (
	ErrRunNotFound = errors.New("run not found")
)

type RunStatus string

const (
	StatusPlanned   RunStatus = "planned"
	StatusRunning   RunStatus = "running"
	StatusCompleted RunStatus = "completed"
	StatusFailed    RunStatus = "failed"
	StatusStopped   RunStatus = "stopped"
)

type RunRecord struct {
	ID                string
	Experiment        string
	Target            string
	Status            RunStatus
	StartedAt         time.Time
	EndedAt           *time.Time
	Duration          time.Duration
	Interval          time.Duration
	Apply             bool
	UnsafeMaxDuration bool
	Result            string
	Metadata          map[string]string
	StopRequested     bool
}

type EventRecord struct {
	RunID   string
	Time    time.Time
	Level   string
	Action  string
	Result  string
	Target  string
	Details map[string]string
	Message string
}

type Store interface {
	Init(ctx context.Context) error
	Close() error
	CreateRun(ctx context.Context, run RunRecord) error
	UpdateRunStatus(ctx context.Context, runID string, status RunStatus, endedAt time.Time, result string) error
	UpdateRunMetadata(ctx context.Context, runID string, metadata map[string]string) error
	MarkStopRequested(ctx context.Context, runID string, requested bool) error
	IsStopRequested(ctx context.Context, runID string) (bool, error)
	GetRun(ctx context.Context, runID string) (RunRecord, error)
	ListRuns(ctx context.Context, limit int) ([]RunRecord, error)
	AddEvent(ctx context.Context, event EventRecord) error
}
