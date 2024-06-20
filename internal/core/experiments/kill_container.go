package experiments

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"ruckus/internal/store"
)

type KillContainerExperiment struct{}

func (e *KillContainerExperiment) Name() string {
	return "kill-container"
}

func (e *KillContainerExperiment) Plan(_ context.Context, request Request, _ Environment) ([]string, error) {
	restartCount := countIterations(request.Duration, request.Interval)
	lines := []string{
		fmt.Sprintf("experiment: %s", e.Name()),
		fmt.Sprintf("target: %s (%s)", request.TargetName, request.TargetID),
		fmt.Sprintf("duration: %s", request.Duration),
		fmt.Sprintf("interval: %s", request.Interval),
		fmt.Sprintf("actions: docker restart every %s (%d restarts expected)", request.Interval, restartCount),
	}
	return lines, nil
}

func (e *KillContainerExperiment) Run(ctx context.Context, request Request, env Environment) error {
	initialRunning, err := env.Docker.IsContainerRunning(ctx, request.TargetID)
	if err != nil {
		return fmt.Errorf("inspect target state before run: %w", err)
	}

	metadata := map[string]string{
		"initial_running": strconv.FormatBool(initialRunning),
	}
	if persistErr := env.PersistMetadata(metadata); persistErr != nil {
		return persistErr
	}

	env.Cleanup.Register("ensure target container returns to running state", func(cleanupCtx context.Context) error {
		if !initialRunning {
			return nil
		}
		running, inspectErr := env.Docker.IsContainerRunning(cleanupCtx, request.TargetID)
		if inspectErr != nil {
			return inspectErr
		}
		if running {
			return nil
		}
		return env.Docker.StartContainer(cleanupCtx, request.TargetID)
	})

	deadline := time.Now().Add(request.Duration)
	iteration := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		iteration++
		if err := env.Docker.RestartContainer(ctx, request.TargetID); err != nil {
			env.RecordAction("docker.restart", "error", err.Error(), map[string]string{
				"iteration": strconv.Itoa(iteration),
			})
			return fmt.Errorf("restart target on iteration %d: %w", iteration, err)
		}

		env.RecordAction("docker.restart", "ok", "container restarted", map[string]string{
			"iteration": strconv.Itoa(iteration),
		})

		if time.Now().Add(request.Interval).After(deadline) {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(request.Interval):
		}
	}

	return nil
}

func (e *KillContainerExperiment) Revert(ctx context.Context, run store.RunRecord, env Environment) error {
	initialRunning := run.Metadata["initial_running"] == "true"
	if !initialRunning {
		return nil
	}

	target := run.Metadata["target_id"]
	if target == "" {
		target = run.Target
	}

	running, err := env.Docker.IsContainerRunning(ctx, target)
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	return env.Docker.StartContainer(ctx, target)
}

func countIterations(duration time.Duration, interval time.Duration) int {
	if duration <= 0 || interval <= 0 {
		return 0
	}
	return int((duration-1)/interval) + 1
}
