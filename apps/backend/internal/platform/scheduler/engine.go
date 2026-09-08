package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bowerbird/internal/platform/id"
	"github.com/bowerbird/internal/platform/outbox/relay/broker"
	"github.com/bowerbird/internal/platform/outbox/store"
)

const OutboxSweeperJobType = "platform.OutboxSweeper"

type Rule struct {
	Name     string
	Schedule string
	JobType  string
	Payload  []byte
}

type compiledRule struct {
	Rule
	sched Schedule
}

type Engine struct {
	transport broker.Transport
	rules     []compiledRule
}

func PlatformRules() []Rule {
	return []Rule{{
		Name:     "outbox-sweeper",
		Schedule: "rate(1 hour)",
		JobType:  OutboxSweeperJobType,
	}}
}

func NewEngine(transport broker.Transport, rules []Rule) (*Engine, error) {
	if transport == nil {
		return nil, fmt.Errorf("transport is required")
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("at least one rule is required")
	}

	names := make(map[string]struct{}, len(rules))
	compiled := make([]compiledRule, 0, len(rules))
	for _, rule := range rules {
		if err := validateRule(rule, names); err != nil {
			return nil, err
		}
		names[rule.Name] = struct{}{}
		sched, err := ParseSchedule(rule.Schedule)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", rule.Name, err)
		}
		if len(rule.Payload) == 0 {
			rule.Payload = []byte(`{}`)
		}
		compiled = append(compiled, compiledRule{Rule: rule, sched: sched})
	}

	return &Engine{transport: transport, rules: compiled}, nil
}

func validateRule(rule Rule, names map[string]struct{}) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if _, dup := names[rule.Name]; dup {
		return fmt.Errorf("duplicate rule name %q", rule.Name)
	}
	if rule.JobType == "" {
		return fmt.Errorf("rule %q: job type is required", rule.Name)
	}
	if rule.Schedule == "" {
		return fmt.Errorf("rule %q: schedule is required", rule.Name)
	}
	return nil
}

func (e *Engine) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	for i := range e.rules {
		wg.Add(1)
		go func(rule compiledRule) {
			defer wg.Done()
			e.runRule(ctx, rule)
		}(e.rules[i])
	}
	wg.Wait()
	return ctx.Err()
}

func (e *Engine) Stop(context.Context) error { return nil }

func (e *Engine) runRule(ctx context.Context, rule compiledRule) {
	var running sync.Mutex
	for {
		wait := time.Until(rule.sched.Next(time.Now().UTC()))
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if !running.TryLock() {
				log.Printf("scheduler rule %s skipped: still running", rule.Name)
				continue
			}
			e.fire(ctx, rule.Rule)
			running.Unlock()
		}
	}
}

func (e *Engine) Fire(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("rule name is required")
	}
	for _, rule := range e.rules {
		if rule.Name == name {
			e.fire(ctx, rule.Rule)
			return nil
		}
	}
	log.Printf("scheduler rule %s skipped: not registered", name)
	return nil
}

func (e *Engine) fire(ctx context.Context, rule Rule) {
	jobID := id.NewULID()
	if err := e.transport.DeliverJob(ctx, store.JobRow{
		ID:            jobID,
		JobType:       rule.JobType,
		Payload:       rule.Payload,
		CorrelationID: jobID,
	}); err != nil {
		log.Printf("scheduler rule %s: %v", rule.Name, err)
	}
}
