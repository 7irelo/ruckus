package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newTargetsCommand(rootOptions *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "targets",
		Short: "List allowlisted containers eligible for experiments",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mustRunWithDeps(cmd, rootOptions, true, func(ctx context.Context, _ *cobra.Command, commandDeps deps) error {
				targets, err := commandDeps.Runner.Targets(ctx)
				if err != nil {
					return err
				}

				if rootOptions.Human {
					writer := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
					fmt.Fprintln(writer, "NAME\tCONTAINER ID\tIMAGE\tSTATE")
					for _, target := range targets {
						fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", target.Name, target.ID, target.Image, target.State)
					}
					return writer.Flush()
				}

				encoded, err := json.MarshalIndent(targets, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(os.Stdout, string(encoded))
				return nil
			})
		},
	}

	return cmd
}
