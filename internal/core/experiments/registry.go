package experiments

import (
	"fmt"
)

type Registry struct {
	entries map[string]Experiment
}

func NewRegistry() *Registry {
	kill := &KillContainerExperiment{}
	netLatency := &NetLatencyExperiment{}
	cpuStress := &CPUStressExperiment{}

	return &Registry{
		entries: map[string]Experiment{
			kill.Name():       kill,
			netLatency.Name(): netLatency,
			cpuStress.Name():  cpuStress,
		},
	}
}

func (r *Registry) Get(name string) (Experiment, error) {
	experiment, ok := r.entries[name]
	if !ok {
		return nil, fmt.Errorf("unknown experiment %q", name)
	}
	return experiment, nil
}
