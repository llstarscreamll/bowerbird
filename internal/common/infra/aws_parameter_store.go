package infra

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type AWSParameterStore struct {
	client *ssm.Client
}

func (ps *AWSParameterStore) GetParameter(ctx context.Context, name string, secure bool) (string, error) {
	param, err := ps.client.GetParameter(ctx, &ssm.GetParameterInput{Name: &name, WithDecryption: &secure})

	if err != nil {
		return "", err
	}

	return *param.Parameter.Value, nil
}

func NewAwsParameterStore(ctx context.Context) *AWSParameterStore {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic("unable to load AWS SDK config, " + err.Error())
	}

	client := ssm.NewFromConfig(cfg)

	return &AWSParameterStore{client: client}
}
