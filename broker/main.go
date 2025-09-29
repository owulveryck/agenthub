package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/owulveryck/agenthub/internal/agenthub"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start the broker using the new abstraction
	if err := agenthub.StartBroker(ctx); err != nil {
		panic(err)
	}
}
