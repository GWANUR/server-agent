package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"server-agent/internal/app"
	"syscall"
)

func main() {

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	agent, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialize agent: %v", err)
	}

	if err := agent.Run(ctx); err != nil {
		log.Fatalf("agent stopped with error: %v", err)
	}
}