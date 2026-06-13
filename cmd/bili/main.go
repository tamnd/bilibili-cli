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
	if err := fang.Execute(ctx, root,
		fang.WithVersion(cli.Version),
		fang.WithCommit(cli.Commit),
	); err != nil {
		os.Exit(1)
	}
}
