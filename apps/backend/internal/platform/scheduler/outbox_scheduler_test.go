package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bowerbird/internal/platform/jobs"
	"github.com/bowerbird/internal/platform/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	slugs []string
	err   error
}

func (f fakeLister) ListActiveTenantSlugs(context.Context) ([]string, error) {
	return f.slugs, f.err
}

type enqueuedJob struct {
	tenant string
	job    jobs.Job
}

type fakeQueue struct {
	jobs []enqueuedJob
	err  error
}

func (q *fakeQueue) Enqueue(ctx context.Context, job jobs.Job) error {
	slug, _ := tenant.TenantIDFromContext(ctx)
	q.jobs = append(q.jobs, enqueuedJob{tenant: slug, job: job})
	return q.err
}

func TestOutboxSchedulerEnqueuesPerTenant(t *testing.T) {
	queue := &fakeQueue{}
	s := NewOutboxScheduler(queue, fakeLister{slugs: []string{"acme", "beta"}}, time.Hour)

	s.tick(context.Background())

	require.Len(t, queue.jobs, 2)
	assert.Equal(t, "acme", queue.jobs[0].tenant)
	assert.Equal(t, "beta", queue.jobs[1].tenant)
	assert.Equal(t, OutboxSweeperJobType, queue.jobs[0].job.Type)
	assert.Equal(t, OutboxSweeperJobType, queue.jobs[1].job.Type)
}

func TestOutboxSchedulerSkipsEnqueueWhenListFails(t *testing.T) {
	queue := &fakeQueue{}
	s := NewOutboxScheduler(queue, fakeLister{err: errors.New("db down")}, time.Hour)

	s.tick(context.Background())

	assert.Empty(t, queue.jobs)
}

func TestOutboxSchedulerSkipsEnqueueWhenNoTenants(t *testing.T) {
	queue := &fakeQueue{}
	s := NewOutboxScheduler(queue, fakeLister{}, time.Hour)

	s.tick(context.Background())

	assert.Empty(t, queue.jobs)
}

func TestNewOutboxSchedulerPanicsWhenQueueMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	NewOutboxScheduler(nil, fakeLister{}, time.Hour)
}

func TestNewOutboxSchedulerPanicsWhenListerMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	NewOutboxScheduler(&fakeQueue{}, nil, time.Hour)
}
