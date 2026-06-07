package jobs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqsTypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/bowerbird/internal/platform/tenant"
)

type SQSQueue struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSQueue(client *sqs.Client, queueURL string) *SQSQueue {
	if client == nil {
		panic("sqs client is required")
	}
	if queueURL == "" {
		panic("queue url is required")
	}

	return &SQSQueue{client: client, queueURL: queueURL}
}

func (q *SQSQueue) Dispatch(ctx context.Context, job Job) error {
	tenantID, err := tenant.TenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	attrs := map[string]sqsTypes.MessageAttributeValue{
		"JobType": {
			DataType:    aws.String("String"),
			StringValue: aws.String(job.Type),
		},
	}

	attrs["TenantID"] = sqsTypes.MessageAttributeValue{
		DataType:    aws.String("String"),
		StringValue: aws.String(tenantID),
	}

	_, err = q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(q.queueURL),
		MessageBody:       aws.String(string(job.Payload)),
		MessageAttributes: attrs,
	})

	return err
}
