package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/tacokumo/portal-api/internal/server/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	c := cmd.New()
	if err := c.ExecuteContext(ctx); err != nil {
		panic(err)
	}
}
