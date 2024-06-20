package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"ruckus/internal/adapters/docker"
	"ruckus/internal/core/experiments"
	"ruckus/internal/safety"
	"ruckus/internal/store"
)

type Runner struct {
	store    store.Store
	docker   *docker.Adapter
	registry *experiments.Registry
	logger   zerolog.Logger
	nowFn    func() time.Time
}

func NewRunner(runStore store.Store, dockerAdapter *docker.Adapter, logger zerolog.Logger) *Runner {
	return &Runner{
		store:    runStore,
		docker:   dockerAdapter,
		registry: experiments.NewRegistry(),
		logger:   logger,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (r *Runner) Plan(ctx context.Context, options RunOptions) ([]string, error) {
	if err := r.validateSharedOptions(options); err != nil {
		return nil, err
	}
	if r.docker == nil {
		return nil, errors.New("docker adapter is not configured")
	}

	target, err := r.docker.InspectContainer(ctx, options.Target)
	if err != nil {
		return nil, err
	}
	if !safety.IsAllowlisted(target.Labels) {
		return nil, fmt.Errorf("target %q is not allowlisted: container must have label %s=%s", target.Name, safety.AllowLabelKey, safety.AllowLabelValue)
	}

	experiment, err := r.registry.Get(options.Experiment)
	if err != nil {
		return nil, err
	}

	request := optionsToRequest("", target, options)
	lines, err := experiment.Plan(ctx, request, experiments.Environment{
		Docker:  r.docker,
		Cleanup: safety.NewCleanupManager(),
		RecordAction: func(_ string, _ string, _ string, _ map[string]string) {
			// Plan mode does not mutate systems and does not emit run events.
		},
		PersistMetadata: func(_ map[string]string) error {
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	header := []string{
		"mode: plan (dry-run)",
		"execution: no changes will be made",
	}

	return append(header, lines...), nil
}

func (r *Runner) Run(ctx context.Context, options RunOptions) (string, error) {
	if err := safety.ValidateRunApproval(options.Apply, options.YesIUnderstand); err != nil {
		return "", err
	}
	if err := r.validateSharedOptions(options); err != nil {
		return "", err
	}
	if r.docker == nil {
		return "", errors.New("docker adapter is not configured")
	}

	target, err := r.docker.InspectContainer(ctx, options.Target)
	if err != nil {
		return "", err
	}
	if !safety.IsAllowlisted(target.Labels) {
		return "", fmt.Errorf("target %q is not allowlisted: container must have label %s=%s", target.Name, safety.AllowLabelKey, safety.AllowLabelValue)
	}

	experiment, err := r.registry.Get(options.Experiment)
	if err != nil {
		return "", err
	}

	runID := NewRunID()
	now := r.nowFn()
	runMetadata := options.Metadata()
	runMetadata["target_id"] = target.ID
	runMetadata["target_name"] = target.Name

	runRecord := store.RunRecord{
		ID:                runID,
		Experiment:        options.Experiment,
		Target:            target.Name,
		Status:            store.StatusRunning,
		StartedAt:         now,
		Duration:          options.Duration,
		Interval:          options.Interval,
		Apply:             true,
		UnsafeMaxDuration: options.UnsafeMaxDuration,
		Result:            "",
		Metadata:          runMetadata,
		StopRequested:     false,
	}
	if err := r.store.CreateRun(ctx, runRecord); err != nil {
		return "", err
	}

	var (
		metadataMu sync.Mutex
	)
	persistMetadata := func(updates map[string]string) error {
		metadataMu.Lock()
		for key, value := range updates {
			runMetadata[key] = value
		}
		metadataSnapshot := cloneMetadata(runMetadata)
		metadataMu.Unlock()

		return r.store.UpdateRunMetadata(context.Background(), runID, metadataSnapshot)
	}

	recordAction := func(action string, result string, message string, details map[string]string) {
		eventTime := r.nowFn()

		logEvent := r.logger.With().
			Str("run_id", runID).
			Str("target", target.Name).
			Str("experiment", options.Experiment).
			Str("action", action).
			Str("result", result).
			Time("ts", eventTime).
			Logger()

		if result == "error" {
			logEvent.Error().Fields(details).Msg(message)
		} else {
			logEvent.Info().Fields(details).Msg(message)
		}

		_ = r.store.AddEvent(context.Background(), store.EventRecord{
			RunID:   runID,
			Time:    eventTime,
			Level:   ifElse(result == "error", "error", "info"),
			Action:  action,
			Result:  result,
			Target:  target.Name,
			Details: details,
			Message: message,
		})
	}

	request := optionsToRequest(runID, target, options)
	cleanup := safety.NewCleanupManager()
	env := experiments.Environment{
		Docker:          r.docker,
		Cleanup:         cleanup,
		RecordAction:    recordAction,
		PersistMetadata: persistMetadata,
	}

	recordAction("run.start", "ok", "experiment started", map[string]string{
		"duration": options.Duration.String(),
		"interval": options.Interval.String(),
	})

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	go r.watchStopRequests(runCtx, runID, cancelRun)

	runErr := experiment.Run(runCtx, request, env)
	cleanupErr := cleanup.Run(context.Background())

	stopRequested, stopCheckErr := r.store.IsStopRequested(context.Background(), runID)
	if stopCheckErr != nil && !errors.Is(stopCheckErr, store.ErrRunNotFound) {
		recordAction("run.stop-check", "error", stopCheckErr.Error(), nil)
	}

	status := store.StatusCompleted
	result := "completed"

	switch {
	case runErr == nil:
		status = store.StatusCompleted
		result = "completed"
	case errors.Is(runErr, context.Canceled):
		status = store.StatusStopped
		if stopRequested {
			result = "stopped by user"
		} else {
			result = "stopped via context cancellation"
		}
	default:
		status = store.StatusFailed
		result = runErr.Error()
	}

	if cleanupErr != nil {
		status = store.StatusFailed
		result = fmt.Sprintf("%s; cleanup error: %v", result, cleanupErr)
		recordAction("run.cleanup", "error", cleanupErr.Error(), nil)
	}

	endedAt := r.nowFn()
	if err := r.store.UpdateRunStatus(context.Background(), runID, status, endedAt, result); err != nil {
		return runID, err
	}

	recordAction("run.finish", ifElse(status == store.StatusFailed, "error", "ok"), result, map[string]string{
		"status": string(status),
	})

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		return runID, runErr
	}
	if cleanupErr != nil {
		return runID, cleanupErr
	}

	return runID, nil
}

func (r *Runner) Stop(ctx context.Context, runID string) error {
	run, err := r.store.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	if run.Status != store.StatusRunning {
		return fmt.Errorf("run %s is not active (status: %s)", run.ID, run.Status)
	}

	if err := r.store.MarkStopRequested(ctx, runID, true); err != nil {
		return err
	}

	experiment, err := r.registry.Get(run.Experiment)
	if err != nil {
		return err
	}

	env := experiments.Environment{
		Docker:  r.docker,
		Cleanup: safety.NewCleanupManager(),
		RecordAction: func(action string, result string, message string, details map[string]string) {
			eventTime := r.nowFn()
			logEvent := r.logger.With().
				Str("run_id", runID).
				Str("target", run.Target).
				Str("experiment", run.Experiment).
				Str("action", action).
				Str("result", result).
				Time("ts", eventTime).
				Logger()

			if result == "error" {
				logEvent.Error().Fields(details).Msg(message)
			} else {
				logEvent.Info().Fields(details).Msg(message)
			}

			_ = r.store.AddEvent(context.Background(), store.EventRecord{
				RunID:   runID,
				Time:    eventTime,
				Level:   ifElse(result == "error", "error", "info"),
				Action:  action,
				Result:  result,
				Target:  run.Target,
				Details: details,
				Message: message,
			})
		},
		PersistMetadata: func(_ map[string]string) error {
			return nil
		},
	}

	revertErr := experiment.Revert(ctx, run, env)
	status := store.StatusStopped
	result := "stopped by user"
	if revertErr != nil {
		status = store.StatusFailed
		result = fmt.Sprintf("stop requested but revert failed: %v", revertErr)
	}

	if err := r.store.UpdateRunStatus(ctx, runID, status, r.nowFn(), result); err != nil {
		return err
	}

	if revertErr != nil {
		return revertErr
	}
	return nil
}

func (r *Runner) Status(ctx context.Context, limit int) ([]store.RunRecord, error) {
	return r.store.ListRuns(ctx, limit)
}

func (r *Runner) Targets(ctx context.Context) ([]docker.ContainerInfo, error) {
	if r.docker == nil {
		return nil, errors.New("docker adapter is not configured")
	}
	return r.docker.ListEligibleContainers(ctx, safety.AllowLabelKey, safety.AllowLabelValue)
}

func (r *Runner) watchStopRequests(ctx context.Context, runID string, cancel context.CancelFunc) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stopRequested, err := r.store.IsStopRequested(context.Background(), runID)
			if err != nil {
				continue
			}
			if stopRequested {
				cancel()
				return
			}
		}
	}
}

func (r *Runner) validateSharedOptions(options RunOptions) error {
	if err := safety.ValidateTarget(options.Target); err != nil {
		return err
	}
	if err := safety.ValidateDuration(options.Duration, options.UnsafeMaxDuration); err != nil {
		return err
	}
	if err := safety.ValidateInterval(options.Interval); err != nil {
		return err
	}

	experiment, err := r.registry.Get(options.Experiment)
	if err != nil {
		return err
	}
	_ = experiment

	switch options.Experiment {
	case ExperimentNetLatency:
		if options.Interface == "" {
			return errors.New("net-latency requires --iface")
		}
		if options.Latency <= 0 {
			return errors.New("net-latency requires --latency > 0")
		}
		if options.Jitter < 0 {
			return errors.New("net-latency requires --jitter >= 0")
		}
	case ExperimentCPUStress:
		if options.CPUWorkers <= 0 {
			return errors.New("cpu-stress requires --cpu-workers > 0")
		}
		if options.StressImage == "" {
			return errors.New("cpu-stress requires --stress-image")
		}
	}

	return nil
}

func optionsToRequest(runID string, target docker.ContainerInfo, options RunOptions) experiments.Request {
	return experiments.Request{
		RunID:           runID,
		Target:          options.Target,
		TargetID:        target.ID,
		TargetName:      target.Name,
		Experiment:      options.Experiment,
		Duration:        options.Duration,
		Interval:        options.Interval,
		Latency:         options.Latency,
		Jitter:          options.Jitter,
		Interface:       options.Interface,
		CPUWorkers:      options.CPUWorkers,
		StressImage:     options.StressImage,
		AllowHostStress: options.AllowHostStress,
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	copyMap := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copyMap[key] = value
	}
	return copyMap
}

func ifElse[T any](condition bool, a T, b T) T {
	if condition {
		return a
	}
	return b
}

func ParseBool(metadata map[string]string, key string) bool {
	parsed, _ := strconv.ParseBool(metadata[key])
	return parsed
}
