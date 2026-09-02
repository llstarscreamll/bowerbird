package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	url  string
	conn *amqp.Connection
}

func NewConnection(url string) *Connection {
	return &Connection{url: url}
}

func (c *Connection) Connect() error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Connection) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}
	return c.conn.Channel()
}

func (c *Connection) RunLoop(ctx context.Context, onConnect func(*amqp.Connection) error, onDisconnect func(error)) {
	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		if ctx.Err() != nil {
			return
		}

		if err := c.Connect(); err != nil {
			if onDisconnect != nil {
				onDisconnect(err)
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second
		if err := onConnect(c.conn); err != nil {
			_ = c.Close()
			if ctx.Err() != nil {
				return
			}
			if onDisconnect != nil {
				onDisconnect(err)
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = minDuration(backoff*2, maxBackoff)
			continue
		}

		select {
		case closeErr := <-c.conn.NotifyClose(make(chan *amqp.Error)):
			_ = c.Close()
			if ctx.Err() != nil {
				return
			}
			if onDisconnect != nil {
				onDisconnect(closeErr)
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			backoff = minDuration(backoff*2, maxBackoff)
		case <-ctx.Done():
			_ = c.Close()
			return
		}
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// BackoffForTest exposes backoff doubling for unit tests.
func (c *Connection) BackoffForTest(current time.Duration) time.Duration {
	return minDuration(current*2, 5*time.Minute)
}

func DeclareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(DLXExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(DeadLetterQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(DeadLetterQueue, "", DLXExchange, false, nil); err != nil {
		return err
	}

	dlqArgs := amqp.Table{"x-dead-letter-exchange": DLXExchange}
	if err := ch.ExchangeDeclare(EventsExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(EventsQueue, true, false, false, false, dlqArgs); err != nil {
		return err
	}
	if err := ch.QueueBind(EventsQueue, "#", EventsExchange, false, nil); err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(JobsExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(JobsQueue, true, false, false, false, dlqArgs); err != nil {
		return err
	}
	if err := declareRetryQueues(ch, JobsQueue, ConsumerRetryDelays); err != nil {
		return err
	}
	if err := declareRetryQueues(ch, EventsQueue, ConsumerRetryDelays); err != nil {
		return err
	}
	return nil
}
