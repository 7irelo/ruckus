package cli

import (
	"time"

	"github.com/spf13/cobra"

	"ruckus/internal/core"
	"ruckus/internal/safety"
)

type experimentFlags struct {
	target            string
	duration          time.Duration
	interval          time.Duration
	unsafeMaxDuration bool

	latency         time.Duration
	jitter          time.Duration
	iface           string
	cpuWorkers      int
	stressImage     string
	allowHostStress bool
}

func bindExperimentFlags(cmd *cobra.Command) *experimentFlags {
	flags := &experimentFlags{
		duration:        safety.DefaultDuration,
		interval:        safety.DefaultInterval,
		latency:         100 * time.Millisecond,
		jitter:          20 * time.Millisecond,
		iface:           "eth0",
		cpuWorkers:      1,
		stressImage:     "progrium/stress",
		allowHostStress: false,
	}

	cmd.Flags().StringVar(&flags.target, "target", "", "target container name or ID (required)")
	cmd.Flags().DurationVar(&flags.duration, "duration", safety.DefaultDuration, "experiment duration")
	cmd.Flags().DurationVar(&flags.interval, "interval", safety.DefaultInterval, "action interval for repeating experiments")
	cmd.Flags().BoolVar(&flags.unsafeMaxDuration, "unsafe-max-duration", false, "allow duration above 5m hard safety cap")

	cmd.Flags().DurationVar(&flags.latency, "latency", 100*time.Millisecond, "latency to inject for net-latency")
	cmd.Flags().DurationVar(&flags.jitter, "jitter", 20*time.Millisecond, "latency jitter to inject for net-latency")
	cmd.Flags().StringVar(&flags.iface, "iface", "eth0", "container network interface for net-latency")

	cmd.Flags().IntVar(&flags.cpuWorkers, "cpu-workers", 1, "number of CPU workers for cpu-stress")
	cmd.Flags().StringVar(&flags.stressImage, "stress-image", "progrium/stress", "stress image used by cpu-stress")
	cmd.Flags().BoolVar(&flags.allowHostStress, "allow-host-stress", false, "allow host-level stress fallback (unsafe)")

	_ = cmd.MarkFlagRequired("target")

	return flags
}

func (f *experimentFlags) toRunOptions(experiment string) core.RunOptions {
	return core.RunOptions{
		Experiment:        experiment,
		Target:            f.target,
		Duration:          f.duration,
		Interval:          f.interval,
		UnsafeMaxDuration: f.unsafeMaxDuration,
		Latency:           f.latency,
		Jitter:            f.jitter,
		Interface:         f.iface,
		CPUWorkers:        f.cpuWorkers,
		StressImage:       f.stressImage,
		AllowHostStress:   f.allowHostStress,
	}
}
