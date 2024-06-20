package experiments

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"ruckus/internal/store"
)

var ifacePattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)

type NetLatencyExperiment struct{}

func (e *NetLatencyExperiment) Name() string {
	return "net-latency"
}

func (e *NetLatencyExperiment) Plan(ctx context.Context, request Request, env Environment) ([]string, error) {
	if !ifacePattern.MatchString(request.Interface) {
		return nil, fmt.Errorf("invalid interface %q", request.Interface)
	}

	available, err := env.Docker.IsTCNetemAvailable(ctx, request.TargetID)
	if err != nil {
		return nil, fmt.Errorf("check tc availability: %w", err)
	}
	if !available {
		return nil, UnsupportedError{Reason: "tc/netem is unavailable in the target container"}
	}

	lines := []string{
		fmt.Sprintf("experiment: %s", e.Name()),
		fmt.Sprintf("target: %s (%s)", request.TargetName, request.TargetID),
		fmt.Sprintf("duration: %s", request.Duration),
		fmt.Sprintf("interface: %s", request.Interface),
		fmt.Sprintf("actions: apply tc netem delay %s jitter %s, then auto-revert", request.Latency, request.Jitter),
	}
	return lines, nil
}

func (e *NetLatencyExperiment) Run(ctx context.Context, request Request, env Environment) error {
	if !ifacePattern.MatchString(request.Interface) {
		return fmt.Errorf("invalid interface %q", request.Interface)
	}

	available, err := env.Docker.IsTCNetemAvailable(ctx, request.TargetID)
	if err != nil {
		return fmt.Errorf("check tc availability: %w", err)
	}
	if !available {
		return UnsupportedError{Reason: "tc/netem is unavailable in the target container"}
	}

	latency := tcDuration(request.Latency)
	jitter := tcDuration(request.Jitter)

	env.Cleanup.Register("remove tc netem qdisc", func(cleanupCtx context.Context) error {
		return env.Docker.ClearNetem(cleanupCtx, request.TargetID, request.Interface)
	})

	if err := env.Docker.ApplyNetem(ctx, request.TargetID, request.Interface, latency, jitter); err != nil {
		env.RecordAction("tc.netem.apply", "error", err.Error(), map[string]string{
			"interface": request.Interface,
			"latency":   latency,
			"jitter":    jitter,
		})
		return err
	}

	metadata := map[string]string{
		"interface": request.Interface,
		"latency":   request.Latency.String(),
		"jitter":    request.Jitter.String(),
	}
	if err := env.PersistMetadata(metadata); err != nil {
		return err
	}

	env.RecordAction("tc.netem.apply", "ok", "netem delay injected", map[string]string{
		"interface": request.Interface,
		"latency":   latency,
		"jitter":    jitter,
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(request.Duration):
		return nil
	}
}

func (e *NetLatencyExperiment) Revert(ctx context.Context, run store.RunRecord, env Environment) error {
	iface := run.Metadata["interface"]
	if iface == "" {
		iface = "eth0"
	}
	target := run.Metadata["target_id"]
	if target == "" {
		target = run.Target
	}
	return env.Docker.ClearNetem(ctx, target, iface)
}

func tcDuration(duration time.Duration) string {
	if duration <= 0 {
		return "1ms"
	}
	ms := duration.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	return fmt.Sprintf("%dms", ms)
}
