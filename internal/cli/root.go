package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"ruckus/internal/adapters/docker"
	"ruckus/internal/core"
	"ruckus/internal/store"
)

type RootOptions struct {
	Human  bool
	DBPath string
}

func Execute() error {
	root, err := NewRootCommand()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return root.ExecuteContext(ctx)
}

func NewRootCommand() (*cobra.Command, error) {
	defaultDBPath, err := store.DefaultDBPath()
	if err != nil {
		return nil, err
	}

	options := &RootOptions{
		Human:  false,
		DBPath: defaultDBPath,
	}

	rootCmd := &cobra.Command{
		Use:   "ruckus",
		Short: "Ruckus is a safe-by-default local Docker chaos engineering CLI",
		Long:  "Ruckus runs time-bounded and auto-reverting chaos experiments against allowlisted local Docker containers.",
	}

	rootCmd.PersistentFlags().BoolVar(&options.Human, "human", false, "render human-friendly output")
	rootCmd.PersistentFlags().StringVar(&options.DBPath, "db-path", defaultDBPath, "path to sqlite run history database")

	rootCmd.AddCommand(newPlanCommand(options))
	rootCmd.AddCommand(newRunCommand(options))
	rootCmd.AddCommand(newStopCommand(options))
	rootCmd.AddCommand(newStatusCommand(options))
	rootCmd.AddCommand(newTargetsCommand(options))
	rootCmd.AddCommand(newHistoryCommand(options))

	return rootCmd, nil
}

type deps struct {
	Runner *core.Runner
	Store  store.Store
}

func buildDeps(ctx context.Context, options *RootOptions, requireDocker bool) (deps, error) {
	logger := buildLogger(options.Human)

	runStore, err := store.NewSQLiteStore(options.DBPath)
	if err != nil {
		return deps{}, err
	}
	if err := runStore.Init(ctx); err != nil {
		_ = runStore.Close()
		return deps{}, err
	}

	var dockerAdapter *docker.Adapter
	if requireDocker {
		dockerAdapter, err = docker.NewLocalAdapter()
		if err != nil {
			_ = runStore.Close()
			return deps{}, err
		}
		if pingErr := dockerAdapter.Ping(ctx); pingErr != nil {
			_ = runStore.Close()
			return deps{}, pingErr
		}
	}

	return deps{
		Runner: core.NewRunner(runStore, dockerAdapter, logger),
		Store:  runStore,
	}, nil
}

func closeDeps(commandDeps deps) {
	if commandDeps.Store != nil {
		_ = commandDeps.Store.Close()
	}
}

func mustRunWithDeps(
	cmd *cobra.Command,
	options *RootOptions,
	requireDocker bool,
	handler func(context.Context, *cobra.Command, deps) error,
) error {
	commandDeps, err := buildDeps(cmd.Context(), options, requireDocker)
	if err != nil {
		return err
	}
	defer closeDeps(commandDeps)

	if err := handler(cmd.Context(), cmd, commandDeps); err != nil {
		return err
	}
	return nil
}

func printHumanLines(lines []string) {
	for _, line := range lines {
		fmt.Fprintln(os.Stdout, line)
	}
}
