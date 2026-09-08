package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bowerbird/internal/platform"
	httphost "github.com/bowerbird/internal/platform/http/host"
)

func main() {
	ctxApp, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	platformModule, err := platform.NewModule(ctxApp)
	if err != nil {
		log.Fatalf("failed to build dependencies at boot: %v", err)
	}
	defer platformModule.ControlDB.Close()
	defer platformModule.TenantRegistry.CloseAll()

	handler, err := httphost.New(platformModule)
	if err != nil {
		log.Fatalf("failed to wire http api: %v", err)
	}

	cfg := platformModule.Config
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("http api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	cancelApp()

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}
