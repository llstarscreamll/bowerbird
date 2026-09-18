package jobs

import "context"

type Job struct {
	Type    string
	Payload []byte
}

type TaskQueue interface {
	Enqueue(ctx context.Context, job Job) error
}
