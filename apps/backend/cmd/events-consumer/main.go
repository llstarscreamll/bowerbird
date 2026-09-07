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
	rabbitmqbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

const consumerTag = "events-consumer"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := platform.NewModule(ctx)
	if err != nil {
		log.Fatalf("events-consumer boot: %v", err)
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
		log.Printf("events-consumer shutting down...")
		cancel()
		waitExit(errCh)
	case err := <-errCh:
		if err != nil {
			log.Fatalf("events-consumer: %v", err)
		}
	}
}

func run(ctx context.Context, deps *platform.Dependencies) error {
	handlers := platformMessaging.WireMessagingHandlers(deps)
	conn := rabbitmqbroker.NewConnection(deps.Config.RabbitMQURL)
	conn.RunLoop(ctx, func(amqpConn *amqp.Connection) error {
		ch, err := amqpConn.Channel()
		if err != nil {
			return err
		}
		defer func() {
			_ = ch.Cancel(consumerTag, false)
			_ = ch.Close()
		}()
		if err := rabbitmqbroker.DeclareTopology(ch); err != nil {
			return err
		}
		msgs, err := ch.Consume(rabbitmqbroker.EventsQueue, consumerTag, false, false, false, false, nil)
		if err != nil {
			return err
		}
		log.Printf("events consumer started")
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg, ok := <-msgs:
				if !ok {
					return amqp.ErrClosed
				}
				handleCtx := context.WithoutCancel(ctx)
				if err := handlers.Events.HandleCloudEventJSON(handleCtx, msg.Body); err != nil {
					log.Printf("events consumer error: tenant=%v correlation=%v err=%v", msg.Headers["tenant_slug"], msg.Headers["correlation_id"], err)
					if nackErr := rabbitmqbroker.HandleConsumerFailure(ch, rabbitmqbroker.EventsQueue, msg, err); nackErr != nil {
						log.Printf("events consumer nack error: %v", nackErr)
					}
					continue
				}
				_ = msg.Ack(false)
			}
		}
	}, func(err error) {
		log.Printf("events consumer disconnected: %v", err)
	})
	return nil
}

func waitExit(errCh <-chan error) {
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("events-consumer exit: %v", err)
			return
		}
		log.Printf("events-consumer exited cleanly")
	case <-time.After(10 * time.Second):
		log.Printf("events-consumer shutdown timed out")
	}
}
