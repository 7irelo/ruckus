package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"ruckus/internal/store"
)

func newHistoryCommand(options *RootOptions) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "List past experiment runs",
		Long:  "Display a table (or JSON) of past chaos experiment runs from the local history database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mustRunWithDeps(cmd, options, false, func(ctx context.Context, _ *cobra.Command, d deps) error {
				runs, err := d.Store.ListRuns(ctx, limit)
				if err != nil {
					return err
				}

				if options.Human {
					return printHumanHistory(runs)
				}
				return printJSONHistory(runs)
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", store.DefaultStatusLimit, "maximum number of runs to display")

	return cmd
}

func printHumanHistory(runs []store.RunRecord) error {
	if len(runs) == 0 {
		fmt.Println("No experiment runs recorded yet.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tEXPERIMENT\tTARGET\tSTATUS\tSTARTED AT\tDURATION")

	for _, r := range runs {
		duration := "-"
		if r.Duration > 0 {
			duration = r.Duration.Round(time.Second).String()
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.ID[:8],
			r.Experiment,
			r.Target,
			r.Status,
			r.StartedAt.Format("2006-01-02 15:04:05"),
			duration,
		)
	}

	return w.Flush()
}

func printJSONHistory(runs []store.RunRecord) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(runs)
}
