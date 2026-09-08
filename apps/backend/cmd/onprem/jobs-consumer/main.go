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
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
	rabbitmqbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

const consumerTag = "jobs-consumer"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := platform.NewModule(ctx)
	if err != nil {
		log.Fatalf("jobs-consumer boot: %v", err)
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
		log.Printf("jobs-consumer shutting down...")
		cancel()
		waitExit(errCh)
	case err := <-errCh:
		if err != nil {
			log.Fatalf("jobs-consumer: %v", err)
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
		if err := rabbitmqbroker.BindJobsQueue(ch, handlers.Jobs.JobTypes()...); err != nil {
			return err
		}
		msgs, err := ch.Consume(rabbitmqbroker.JobsQueue, consumerTag, false, false, false, false, nil)
		if err != nil {
			return err
		}
		log.Printf("jobs consumer started (bindings=%v)", handlers.Jobs.JobTypes())
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg, ok := <-msgs:
				if !ok {
					return amqp.ErrClosed
				}
				handleCtx := context.WithoutCancel(ctx)
				if err := handlers.Jobs.HandleJob(handleCtx, parseJobAMQP(msg)); err != nil {
					jobMsg := parseJobAMQP(msg)
					log.Printf("jobs consumer error: id=%s type=%s tenant=%s correlation=%s err=%v",
						jobMsg.MessageID, jobMsg.JobType, jobMsg.TenantSlug, jobMsg.CorrelationID, err)
					if nackErr := rabbitmqbroker.HandleConsumerFailure(ch, rabbitmqbroker.JobsQueue, msg, err); nackErr != nil {
						log.Printf("jobs consumer nack error: %v", nackErr)
					}
					continue
				}
				_ = msg.Ack(false)
			}
		}
	}, func(err error) {
		log.Printf("jobs consumer disconnected: %v", err)
	})
	return nil
}

func parseJobAMQP(msg amqp.Delivery) platformJobs.JobMessage {
	jobMsg := platformJobs.JobMessage{MessageID: msg.MessageId, Body: msg.Body}
	if v, ok := msg.Headers["tenant_slug"].(string); ok {
		jobMsg.TenantSlug = v
	}
	if v, ok := msg.Headers["job_type"].(string); ok {
		jobMsg.JobType = v
	}
	if v, ok := msg.Headers["correlation_id"].(string); ok {
		jobMsg.CorrelationID = v
	}
	if v, ok := msg.Headers["tenant_attestation"].(string); ok {
		jobMsg.TenantAttestation = v
	}
	return jobMsg
}

func waitExit(errCh <-chan error) {
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("jobs-consumer exit: %v", err)
			return
		}
		log.Printf("jobs-consumer exited cleanly")
	case <-time.After(10 * time.Second):
		log.Printf("jobs-consumer shutdown timed out")
	}
}
