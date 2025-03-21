package main

import (
	"os"

	cdk "github.com/aws/aws-cdk-go/awscdk/v2"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkGoStackProps struct {
	cdk.StackProps
}

func NewCdkGoStack(scope constructs.Construct, id string, props *CdkGoStackProps) cdk.Stack {
	var sProps cdk.StackProps
	if props != nil {
		sProps = props.StackProps
	}
	stack := cdk.NewStack(scope, &id, &sProps)

	lambda.NewFunction(stack, jsii.String("GoServer"), &lambda.FunctionProps{
		FunctionName: jsii.String("API"),
		Runtime:      lambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         lambda.Code_FromAsset(jsii.String("cmd/lambda-api/bootstrap.zip"), nil),
		Architecture: lambda.Architecture_ARM_64(),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := cdk.NewApp(nil)

	NewCdkGoStack(app, "BowerbirdApp", &CdkGoStackProps{
		cdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

func env() *cdk.Environment {
	return &cdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
