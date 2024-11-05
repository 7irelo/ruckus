package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func newStatusCommand(rootOptions *RootOptions) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show active and historical runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mustRunWithDeps(cmd, rootOptions, false, func(ctx context.Context, _ *cobra.Command, commandDeps deps) error {
				runs, err := commandDeps.Runner.Status(ctx, limit)
				if err != nil {
					return err
				}

				if rootOptions.Human {
					writer := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
					fmt.Fprintln(writer, "RUN ID\tSTATUS\tEXPERIMENT\tTARGET\tSTARTED\tENDED\tRESULT")
					for _, run := range runs {
						ended := "-"
						if run.EndedAt != nil {
							ended = run.EndedAt.Format(time.RFC3339)
						}
						fmt.Fprintf(
							writer,
							"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
							run.ID,
							run.Status,
							run.Experiment,
							run.Target,
							run.StartedAt.Format(time.RFC3339),
							ended,
							run.Result,
						)
					}
					return writer.Flush()
				}

				encoded, err := json.MarshalIndent(runs, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(os.Stdout, string(encoded))
				return nil
			})
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of runs to show")
	return cmd
}
