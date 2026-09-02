package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

const (
	EventsExchange  = "bowerbird.events"
	JobsExchange    = "bowerbird.jobs"
	DLXExchange     = "bowerbird.dlx"
	EventsQueue     = "bowerbird.events.handlers"
	JobsQueue       = "bowerbird.jobs.work"
	DeadLetterQueue = "bowerbird.deadletter"
)

// BindJobsQueue binds the jobs work queue to routing keys on the direct jobs exchange.
// Callers pass keys from registered job handlers (composition root), not from platform.
func BindJobsQueue(ch *amqp.Channel, keys ...string) error {
	seen := map[string]struct{}{}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := ch.QueueBind(JobsQueue, key, JobsExchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}
