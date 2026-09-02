package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bowerbird/internal/platform"
	awsConfig "github.com/bowerbird/internal/platform/awsconfig"
	"github.com/bowerbird/internal/platform/config"
	platformJobs "github.com/bowerbird/internal/platform/jobs"
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
	"github.com/bowerbird/internal/platform/outbox/relay"
	"github.com/bowerbird/internal/platform/outbox/relay/broker"
	awsbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/aws"
	rabbitmqbroker "github.com/bowerbird/internal/platform/outbox/relay/broker/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: worker <relay|events|jobs|scheduler>")
	}

	ctxApp, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	platformModule, err := platform.NewModule(ctxApp)
	if err != nil {
		log.Fatalf("boot: %v", err)
	}
	defer platformModule.ControlDB.Close()
	defer platformModule.TenantRegistry.CloseAll()
	logObjectStorage(platformModule.Config)

	// We use a separate goroutine to run the worker and handle shutdown gracefully.
	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic: %v", r)
			}
		}()
		switch os.Args[1] {
		case "relay":
			runRelay(ctxApp, platformModule)
		case "events":
			runEventsConsumer(ctxApp, platformModule)
		case "jobs":
			runJobsConsumer(ctxApp, platformModule)
		case "scheduler":
			runScheduler(ctxApp, platformModule)
		default:
			log.Fatalf("unknown subcommand: %s", os.Args[1])
		}
		errCh <- nil
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-shutdown:
		log.Printf("shutting down worker...")
		cancelApp() // Signals components to stop
		// Wait for graceful shutdown (e.g. current jobs to finish processing before context fully aborts them,
		// though in this setup cancelApp() will immediately cancel the context passed to HandleJob)
		select {
		case <-errCh:
			log.Printf("worker exited cleanly")
		case <-time.After(10 * time.Second):
			log.Printf("worker shutdown timed out")
		}
	case err := <-errCh:
		if err != nil {
			log.Fatalf("worker error: %v", err)
		}
	}
}

func runRelay(ctx context.Context, deps *platform.Dependencies) {
	transport, jobRoutingKeys, err := newBrokerTransport(deps)
	if err != nil {
		log.Fatalf("broker transport: %v", err)
	}
	_ = jobRoutingKeys // used only for RabbitMQ transport construction

	lister := relay.NewControlPlaneTenantLister(deps.ControlDB)
	relayCfg := relay.Config{BatchSize: 10, PollInterval: time.Second * 5, PerTenantCap: 10}
	multi := relay.NewMultiTenantRelay(deps.TenantRegistry, lister, transport, relayCfg)
	log.Printf("outbox relay started (target=%s, multi-tenant)", deps.Config.DeploymentTarget)
	multi.RunLoop(ctx)
}

func newBrokerTransport(deps *platform.Dependencies) (broker.Transport, []string, error) {
	cfg := deps.Config
	handlers := platformMessaging.WireMessagingHandlers(deps)
	jobKeys := handlers.Jobs.JobTypes()

	switch cfg.DeploymentTarget {
	case config.DeploymentTargetAWS:
		return awsbroker.NewTransport(
			awsConfig.NewEventBridgeClient(deps.AWSConfig, cfg.AWSEndpointURL),
			awsConfig.NewSQSClient(deps.AWSConfig, cfg.AWSEndpointURL),
			cfg.EventBusName,
			cfg.SQSQueueURL,
			cfg.MessagingAttestationSecret,
		), jobKeys, nil
	default:
		conn := rabbitmqbroker.NewConnection(cfg.RabbitMQURL)
		transport, err := rabbitmqbroker.NewTransport(conn, cfg.MessagingAttestationSecret, jobKeys...)
		return transport, jobKeys, err
	}
}

func runScheduler(ctx context.Context, deps *platform.Dependencies) {
	log.Printf("outbox scheduler started")
	if err := deps.Scheduler.Start(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("scheduler: %v", err)
	}
}

func runEventsConsumer(ctx context.Context, deps *platform.Dependencies) {
	handlers := platformMessaging.WireMessagingHandlers(deps)
	conn := rabbitmqbroker.NewConnection(deps.Config.RabbitMQURL)
	conn.RunLoop(ctx, func(amqpConn *amqp.Connection) error {
		ch, err := amqpConn.Channel()
		if err != nil {
			return err
		}
		if err := rabbitmqbroker.DeclareTopology(ch); err != nil {
			return err
		}
		msgs, err := ch.Consume(rabbitmqbroker.EventsQueue, "", false, false, false, false, nil)
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
				if err := handlers.Events.HandleCloudEventJSON(ctx, msg.Body); err != nil {
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
}

func runJobsConsumer(ctx context.Context, deps *platform.Dependencies) {
	handlers := platformMessaging.WireMessagingHandlers(deps)
	conn := rabbitmqbroker.NewConnection(deps.Config.RabbitMQURL)
	conn.RunLoop(ctx, func(amqpConn *amqp.Connection) error {
		ch, err := amqpConn.Channel()
		if err != nil {
			return err
		}
		if err := rabbitmqbroker.DeclareTopology(ch); err != nil {
			return err
		}
		if err := rabbitmqbroker.BindJobsQueue(ch, handlers.Jobs.JobTypes()...); err != nil {
			return err
		}
		msgs, err := ch.Consume(rabbitmqbroker.JobsQueue, "", false, false, false, false, nil)
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
				if err := handlers.Jobs.HandleJob(ctx, parseJobAMQP(msg)); err != nil {
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

func logObjectStorage(cfg config.Config) {
	if cfg.DeploymentTarget != config.DeploymentTargetOnPrem {
		return
	}
	endpoint := cfg.MinIOEndpointURL
	if endpoint == "" {
		endpoint = cfg.AWSEndpointURL
	}
	log.Printf("object storage ready: endpoint=%s bucket=%s", endpoint, cfg.S3BucketName)
}
