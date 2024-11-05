package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRunCommand(rootOptions *RootOptions) *cobra.Command {
	flags := &experimentFlags{}
	var apply bool
	var yesIUnderstand bool

	cmd := &cobra.Command{
		Use:   "run <experiment> --apply --yes-i-understand",
		Short: "Execute an experiment with safety acknowledgements",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mustRunWithDeps(cmd, rootOptions, true, func(ctx context.Context, _ *cobra.Command, commandDeps deps) error {
				experiment := args[0]
				options := flags.toRunOptions(experiment)
				options.Apply = apply
				options.YesIUnderstand = yesIUnderstand

				runID, err := commandDeps.Runner.Run(ctx, options)
				if err != nil {
					if runID != "" {
						fmt.Fprintf(os.Stdout, "run_id: %s\n", runID)
					}
					return err
				}

				fmt.Fprintf(os.Stdout, "run_id: %s\n", runID)
				return nil
			})
		},
	}

	flags = bindExperimentFlags(cmd)
	cmd.Flags().BoolVar(&apply, "apply", false, "required to execute destructive actions")
	cmd.Flags().BoolVar(&yesIUnderstand, "yes-i-understand", false, "required acknowledgement for destructive actions")

	return cmd
}
