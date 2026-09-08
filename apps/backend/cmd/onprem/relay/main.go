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
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
	"github.com/bowerbird/internal/platform/outbox/relay"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := platform.NewModule(ctx)
	if err != nil {
		log.Fatalf("relay boot: %v", err)
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
		log.Printf("relay shutting down...")
		cancel()
		waitExit("relay", errCh)
	case err := <-errCh:
		if err != nil {
			log.Fatalf("relay: %v", err)
		}
	}
}

func run(ctx context.Context, deps *platform.Dependencies) error {
	transport, closeTransport, err := platformMessaging.NewBrokerTransport(deps)
	if err != nil {
		return err
	}
	defer closeTransport()

	lister := relay.NewControlPlaneTenantLister(deps.ControlDB)
	relayCfg := relay.Config{BatchSize: 10, PollInterval: time.Second * 5, PerTenantCap: 10}
	multi := relay.NewMultiTenantRelay(deps.TenantRegistry, lister, transport, relayCfg)
	log.Printf("outbox relay started (target=%s, multi-tenant)", deps.Config.DeploymentTarget)
	multi.RunLoop(ctx)
	return nil
}

func waitExit(name string, errCh <-chan error) {
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("%s exit: %v", name, err)
			return
		}
		log.Printf("%s exited cleanly", name)
	case <-time.After(10 * time.Second):
		log.Printf("%s shutdown timed out", name)
	}
}
