package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jkroepke/helm-release-size-analyzer/internal/cli"
)

// main exits the process with the status returned by run.
func main() {
	os.Exit(run())
}

// run executes the CLI with signal-aware cancellation and returns its exit status.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := cli.NewRootCommand(os.Args[1:], os.Stdout, os.Stderr)
	if err := cmd.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		return cli.ExitCode(err)
	}

	return 0
}
