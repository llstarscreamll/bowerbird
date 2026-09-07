package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bowerbird/internal/platform"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := platform.NewModule(ctx)
	if err != nil {
		log.Fatalf("scheduler boot: %v", err)
	}
	defer deps.ControlDB.Close()
	defer deps.TenantRegistry.CloseAll()

	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic: %v", r)
			}
		}()
		errCh <- run(ctx, deps)
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-shutdown:
		log.Printf("scheduler shutting down...")
		cancel()
		waitExit(errCh)
	case err := <-errCh:
		if err != nil {
			log.Fatalf("scheduler: %v", err)
		}
	}
}

func run(ctx context.Context, deps *platform.Dependencies) error {
	log.Printf("outbox scheduler started")
	err := deps.Scheduler.Start(ctx)
	stopErr := deps.Scheduler.Stop(context.WithoutCancel(ctx))
	if err != nil && ctx.Err() == nil {
		return err
	}
	return stopErr
}

func waitExit(errCh <-chan error) {
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("scheduler exit: %v", err)
			return
		}
		log.Printf("scheduler exited cleanly")
	case <-time.After(10 * time.Second):
		log.Printf("scheduler shutdown timed out")
	}
}
