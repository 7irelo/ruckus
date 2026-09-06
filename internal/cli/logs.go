package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"ruckus/internal/store"
)

// followPollInterval is how often --follow re-queries the event table. Events
// are written by the runner as an experiment progresses, so polling the store
// is enough to tail a run in progress.
const followPollInterval = 500 * time.Millisecond

func newLogsCommand(options *RootOptions) *cobra.Command {
	var follow bool
	var limit int

	cmd := &cobra.Command{
		Use:   "logs <run-id>",
		Short: "View the event log of an experiment run",
		Long: `Prints the recorded events for a run in chronological order.

With --follow the command keeps printing new events until the run reaches a
terminal state or it is interrupted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mustRunWithDeps(cmd, options, false, func(ctx context.Context, _ *cobra.Command, d deps) error {
				run, err := d.Store.GetRun(ctx, args[0])
				if err != nil {
					return err
				}

				var lastTS int64
				printed, err := printEvents(ctx, d, run.ID, &lastTS, limit, options.Human)
				if err != nil {
					return err
				}

				if !follow {
					if printed == 0 && options.Human {
						fmt.Fprintf(os.Stdout, "No events recorded for run %s yet.\n", run.ID)
					}
					return nil
				}

				ticker := time.NewTicker(followPollInterval)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return nil
					case <-ticker.C:
						if _, err := printEvents(ctx, d, run.ID, &lastTS, limit, options.Human); err != nil {
							return err
						}

						current, err := d.Store.GetRun(ctx, run.ID)
						if err != nil {
							return err
						}
						if isTerminal(current.Status) {
							// Drain anything written between the last poll and
							// the status flip before exiting.
							if _, err := printEvents(ctx, d, run.ID, &lastTS, limit, options.Human); err != nil {
								return err
							}
							return nil
						}
					}
				}
			})
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream new events until the run finishes")
	cmd.Flags().IntVar(&limit, "limit", store.DefaultEventLimit, "maximum number of events to fetch per poll")

	return cmd
}

// printEvents writes every event newer than *lastTS and advances it.
func printEvents(
	ctx context.Context,
	d deps,
	runID string,
	lastTS *int64,
	limit int,
	human bool,
) (int, error) {
	events, err := d.Store.ListEvents(ctx, runID, *lastTS, limit)
	if err != nil {
		return 0, err
	}

	for _, event := range events {
		if human {
			fmt.Fprintln(os.Stdout, formatHumanEvent(event))
		} else {
			encoded, marshalErr := json.Marshal(map[string]any{
				"run_id":  event.RunID,
				"time":    event.Time.Format(time.RFC3339Nano),
				"level":   event.Level,
				"action":  event.Action,
				"result":  event.Result,
				"target":  event.Target,
				"message": event.Message,
				"details": event.Details,
			})
			if marshalErr != nil {
				return 0, fmt.Errorf("encode event: %w", marshalErr)
			}
			fmt.Fprintln(os.Stdout, string(encoded))
		}

		if ts := event.Time.UnixNano(); ts > *lastTS {
			*lastTS = ts
		}
	}

	return len(events), nil
}

func formatHumanEvent(event store.EventRecord) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s  %-5s  %s",
		event.Time.Format("2006-01-02 15:04:05"),
		strings.ToUpper(event.Level),
		event.Action,
	)

	if event.Message != "" {
		fmt.Fprintf(&b, "  %s", event.Message)
	}
	if event.Result != "" {
		fmt.Fprintf(&b, "  (%s)", event.Result)
	}

	return b.String()
}

func isTerminal(status store.RunStatus) bool {
	switch status {
	case store.StatusCompleted, store.StatusFailed, store.StatusStopped:
		return true
	default:
		return false
	}
}
