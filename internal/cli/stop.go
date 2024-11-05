package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStopCommand(rootOptions *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <run-id>",
		Short: "Stop an active run and trigger revert steps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mustRunWithDeps(cmd, rootOptions, true, func(ctx context.Context, _ *cobra.Command, commandDeps deps) error {
				runID := args[0]
				if err := commandDeps.Runner.Stop(ctx, runID); err != nil {
					return err
				}
				fmt.Fprintf(os.Stdout, "stopped run: %s\n", runID)
				return nil
			})
		},
	}

	return cmd
}
