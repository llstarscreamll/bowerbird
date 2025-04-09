package main

import (
	"os"

	cdk "github.com/aws/aws-cdk-go/awscdk/v2"
	apiGatewayV2 "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	apiGatewayV2Integrations "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	parameterStore "github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkGoStackProps struct {
	cdk.StackProps
}

func NewCdkStack(scope constructs.Construct, id string, props *CdkGoStackProps) cdk.Stack {
	var sProps cdk.StackProps
	if props != nil {
		sProps = props.StackProps
	}
	stack := cdk.NewStack(scope, &id, &sProps)

	secrets := parameterStore.StringParameter_FromSecureStringParameterAttributes(stack, jsii.String("backend-secrets"), &parameterStore.SecureStringParameterAttributes{
		ParameterName: jsii.String("prod-bowerbird-backend"),
	})

	// Create Lambda function
	lambdaFn := lambda.NewFunction(stack, jsii.String("GoServer"), &lambda.FunctionProps{
		FunctionName: jsii.String("API"),
		Runtime:      lambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("api-server"),
		Code:         lambda.Code_FromAsset(jsii.String("dist/api-server.zip"), nil),
		Architecture: lambda.Architecture_ARM_64(),
		Environment: &map[string]*string{
			"PARAMETER_STORE_KEY_NAME": secrets.ParameterName(),
		},
	})

	// Create HTTP API with Lambda integration
	httpApi := apiGatewayV2.NewHttpApi(stack, jsii.String("HttpApi"), &apiGatewayV2.HttpApiProps{
		ApiName:            jsii.String("BowerbirdApi"),
		CreateDefaultStage: jsii.Bool(true),
	})

	// Create Lambda integration
	integration := apiGatewayV2Integrations.NewHttpLambdaIntegration(
		jsii.String("LambdaProxyIntegration"),
		lambdaFn,
		&apiGatewayV2Integrations.HttpLambdaIntegrationProps{},
	)

	// Add routes
	httpApi.AddRoutes(&apiGatewayV2.AddRoutesOptions{
		Path:        jsii.String("/{proxy+}"),
		Methods:     &[]apiGatewayV2.HttpMethod{apiGatewayV2.HttpMethod_ANY},
		Integration: integration,
	})

	httpApi.AddRoutes(&apiGatewayV2.AddRoutesOptions{
		Path:        jsii.String("/"),
		Methods:     &[]apiGatewayV2.HttpMethod{apiGatewayV2.HttpMethod_ANY},
		Integration: integration,
	})

	// Grant Lambda permission to read from Parameter Store
	secrets.GrantRead(lambdaFn)

	return stack
}

func main() {
	defer jsii.Close()

	app := cdk.NewApp(nil)

	NewCdkStack(app, "BowerbirdApp", &CdkGoStackProps{
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
