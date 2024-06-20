package experiments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ruckus/internal/adapters/docker"
	"ruckus/internal/store"
)

type CPUStressExperiment struct{}

func (e *CPUStressExperiment) Name() string {
	return "cpu-stress"
}

func (e *CPUStressExperiment) Plan(_ context.Context, request Request, _ Environment) ([]string, error) {
	mode := "container network namespace"
	if request.AllowHostStress {
		mode += " (fallback host-level stress enabled)"
	} else {
		mode += " (host-level fallback disabled)"
	}

	lines := []string{
		fmt.Sprintf("experiment: %s", e.Name()),
		fmt.Sprintf("target: %s (%s)", request.TargetName, request.TargetID),
		fmt.Sprintf("duration: %s", request.Duration),
		fmt.Sprintf("workers: %d", request.CPUWorkers),
		fmt.Sprintf("stress image: %s", request.StressImage),
		fmt.Sprintf("mode: %s", mode),
	}
	return lines, nil
}

func (e *CPUStressExperiment) Run(ctx context.Context, request Request, env Environment) error {
	stressName := fmt.Sprintf("ruckus-stress-%s", shortID(request.RunID))
	stressContainerID, mode, err := runStress(ctx, request, env, stressName)
	if err != nil {
		return err
	}

	env.Cleanup.Register("stop stress container", func(cleanupCtx context.Context) error {
		return env.Docker.StopAndRemoveContainer(cleanupCtx, stressContainerID)
	})

	metadata := map[string]string{
		"stress_container_id": stressContainerID,
		"stress_mode":         mode,
	}
	if persistErr := env.PersistMetadata(metadata); persistErr != nil {
		return persistErr
	}

	env.RecordAction("cpu.stress.start", "ok", "stress container started", map[string]string{
		"stress_container_id": stressContainerID,
		"mode":                mode,
		"workers":             fmt.Sprintf("%d", request.CPUWorkers),
		"image":               request.StressImage,
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(request.Duration):
		return nil
	}
}

func (e *CPUStressExperiment) Revert(ctx context.Context, run store.RunRecord, env Environment) error {
	stressContainerID := run.Metadata["stress_container_id"]
	if stressContainerID == "" {
		return nil
	}
	return env.Docker.StopAndRemoveContainer(ctx, stressContainerID)
}

func runStress(ctx context.Context, request Request, env Environment, stressName string) (string, string, error) {
	targetMode := "container:" + request.TargetID
	containerID, err := env.Docker.RunStressContainer(ctx, docker.StressOptions{
		Name:        stressName,
		Image:       request.StressImage,
		CPUWorkers:  request.CPUWorkers,
		NetworkMode: targetMode,
		HostLevel:   false,
		Target:      request.TargetID,
	})
	if err == nil {
		return containerID, "container-netns", nil
	}

	if !request.AllowHostStress {
		return "", "", UnsupportedError{
			Reason: fmt.Sprintf("unable to run stress container in target namespace (%v); host-level fallback requires --allow-host-stress", err),
		}
	}

	containerID, hostErr := env.Docker.RunStressContainer(ctx, docker.StressOptions{
		Name:       stressName,
		Image:      request.StressImage,
		CPUWorkers: request.CPUWorkers,
		HostLevel:  true,
		Target:     request.TargetID,
	})
	if hostErr != nil {
		return "", "", fmt.Errorf("run host-level stress fallback: %w", hostErr)
	}
	return containerID, "host", nil
}

func shortID(runID string) string {
	cleaned := strings.ReplaceAll(runID, "run-", "")
	if len(cleaned) <= 12 {
		return cleaned
	}
	return cleaned[:12]
}
