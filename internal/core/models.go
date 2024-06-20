package core

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	ExperimentKillContainer = "kill-container"
	ExperimentNetLatency    = "net-latency"
	ExperimentCPUStress     = "cpu-stress"
)

type RunOptions struct {
	Experiment        string
	Target            string
	Duration          time.Duration
	Interval          time.Duration
	UnsafeMaxDuration bool
	Apply             bool
	YesIUnderstand    bool

	Latency         time.Duration
	Jitter          time.Duration
	Interface       string
	CPUWorkers      int
	StressImage     string
	AllowHostStress bool
}

func (o RunOptions) Metadata() map[string]string {
	metadata := map[string]string{
		"latency":           o.Latency.String(),
		"jitter":            o.Jitter.String(),
		"interface":         o.Interface,
		"cpu_workers":       fmt.Sprintf("%d", o.CPUWorkers),
		"stress_image":      o.StressImage,
		"allow_host_stress": fmt.Sprintf("%t", o.AllowHostStress),
	}
	return metadata
}

func NewRunID() string {
	return "run-" + uuid.NewString()
}
