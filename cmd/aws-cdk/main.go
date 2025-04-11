package main

import (
	"os"

	cdk "github.com/aws/aws-cdk-go/awscdk/v2"
	apiGatewayV2 "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	apiGatewayV2Integrations "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	certManager "github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	route53 "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	route53Targets "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
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

	httpApi := apiGatewayV2.NewHttpApi(stack, jsii.String("HttpApi"), &apiGatewayV2.HttpApiProps{
		ApiName:            jsii.String("BowerbirdApi"),
		CreateDefaultStage: jsii.Bool(true),
	})

	hostedZone := route53.HostedZone_FromLookup(stack, jsii.String("HostedZone"), &route53.HostedZoneProviderProps{
		DomainName: jsii.String("money-path.co"),
	})

	certificate := certManager.Certificate_FromCertificateArn(stack, jsii.String("AcmCertificate"), jsii.String("arn:aws:acm:us-east-1:336301087573:certificate/048e55de-9012-4c68-ad1e-6eb5d05478df"))

	domainName := apiGatewayV2.NewDomainName(stack, jsii.String("ApiGatewayCustomDomain"), &apiGatewayV2.DomainNameProps{
		DomainName:   jsii.String("money-path.co"),
		EndpointType: apiGatewayV2.EndpointType_REGIONAL,
		Certificate:  certificate,
	})

	apiGatewayV2.NewApiMapping(stack, jsii.String("ApiMapping"), &apiGatewayV2.ApiMappingProps{
		Api:        httpApi,
		DomainName: domainName,
	})

	route53.NewARecord(stack, jsii.String("ApiARecord"), &route53.ARecordProps{
		Zone:       hostedZone,
		Target:     route53.RecordTarget_FromAlias(route53Targets.NewApiGatewayv2DomainProperties(domainName.RegionalDomainName(), domainName.RegionalHostedZoneId())),
		RecordName: jsii.String("api"),
	})

	integration := apiGatewayV2Integrations.NewHttpLambdaIntegration(
		jsii.String("LambdaProxyIntegration"),
		lambdaFn,
		&apiGatewayV2Integrations.HttpLambdaIntegrationProps{},
	)

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
