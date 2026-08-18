// Command bili is a delightful command line for Bilibili.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/fang"
	"github.com/tamnd/bilibili-cli/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := cli.Root()
	// The exit code is the machine-readable half of the answer. A caller that
	// has to grep stderr to tell "risk control refused" from "there is nothing
	// there" has no answer at all, so every failure leaves through cli.ExitCode
	// and the codes are documented in the README.
	if err := fang.Execute(ctx, root,
		fang.WithVersion(cli.Version),
		fang.WithCommit(cli.Commit),
	); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
