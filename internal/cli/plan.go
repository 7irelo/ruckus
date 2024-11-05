package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPlanCommand(rootOptions *RootOptions) *cobra.Command {
	flags := &experimentFlags{}

	cmd := &cobra.Command{
		Use:   "plan <experiment>",
		Short: "Print what an experiment would do without making changes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mustRunWithDeps(cmd, rootOptions, true, func(ctx context.Context, _ *cobra.Command, commandDeps deps) error {
				experiment := args[0]
				options := flags.toRunOptions(experiment)
				lines, err := commandDeps.Runner.Plan(ctx, options)
				if err != nil {
					return err
				}
				for _, line := range lines {
					fmt.Fprintln(os.Stdout, line)
				}
				return nil
			})
		},
	}

	flags = bindExperimentFlags(cmd)
	return cmd
}
