package experiments

import (
	"context"
	"fmt"
	"time"

	"ruckus/internal/adapters/docker"
	"ruckus/internal/safety"
	"ruckus/internal/store"
)

type Request struct {
	RunID           string
	Target          string
	TargetID        string
	TargetName      string
	Experiment      string
	Duration        time.Duration
	Interval        time.Duration
	Latency         time.Duration
	Jitter          time.Duration
	Interface       string
	CPUWorkers      int
	StressImage     string
	AllowHostStress bool
}

type Environment struct {
	Docker          *docker.Adapter
	Cleanup         *safety.CleanupManager
	RecordAction    func(action string, result string, message string, details map[string]string)
	PersistMetadata func(metadata map[string]string) error
}

type Experiment interface {
	Name() string
	Plan(ctx context.Context, request Request, env Environment) ([]string, error)
	Run(ctx context.Context, request Request, env Environment) error
	Revert(ctx context.Context, run store.RunRecord, env Environment) error
}

type UnsupportedError struct {
	Reason string
}

func (e UnsupportedError) Error() string {
	return fmt.Sprintf("unsupported in v1: %s", e.Reason)
}
