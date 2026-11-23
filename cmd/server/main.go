package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/internal/transport"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	server, err := transport.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("initialize profile service: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := server.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run profile service: %v", err)
	}
}
