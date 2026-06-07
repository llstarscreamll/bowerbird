package jobs

import "context"

type Job struct {
	Type    string
	Payload []byte
}

type Queue interface {
	Dispatch(ctx context.Context, job Job) error
}
