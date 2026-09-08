package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/bowerbird/internal/platform"
	platformMessaging "github.com/bowerbird/internal/platform/messaging"
)

type scheduleEvent struct {
	RuleName string `json:"ruleName"`
}

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, raw json.RawMessage) error {
	deps, err := platform.NewModule(ctx)
	if err != nil {
		return err
	}
	defer deps.ControlDB.Close()
	defer deps.TenantRegistry.CloseAll()

	engine, closeTransport, err := platformMessaging.WireScheduler(deps)
	if err != nil {
		return err
	}
	defer closeTransport()

	ruleName, err := ruleNameFromEvent(raw)
	if err != nil {
		return err
	}
	if err := engine.Fire(ctx, ruleName); err != nil {
		return err
	}
	log.Printf("scheduler fired rule=%s", ruleName)
	return nil
}

func ruleNameFromEvent(raw json.RawMessage) (string, error) {
	var scheduled scheduleEvent
	if err := json.Unmarshal(raw, &scheduled); err == nil && strings.TrimSpace(scheduled.RuleName) != "" {
		return scheduled.RuleName, nil
	}

	var cw events.CloudWatchEvent
	if err := json.Unmarshal(raw, &cw); err == nil {
		var detail scheduleEvent
		if len(cw.Detail) > 0 {
			_ = json.Unmarshal(cw.Detail, &detail)
			if strings.TrimSpace(detail.RuleName) != "" {
				return detail.RuleName, nil
			}
		}
		for _, resource := range cw.Resources {
			if name := ruleNameFromARN(resource); name != "" {
				return name, nil
			}
		}
	}

	return "", errNoRuleName{}
}

type errNoRuleName struct{}

func (errNoRuleName) Error() string { return "scheduler event is missing ruleName" }

func ruleNameFromARN(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
