package main

import (
	"os"

	cdk "github.com/aws/aws-cdk-go/awscdk/v2"
	apiGatewayV2 "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	apiGatewayV2Integrations "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	certManager "github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	cloudfront "github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	cloudfrontOrigins "github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecrassets"
	lambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	route53 "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53"
	route53Targets "github.com/aws/aws-cdk-go/awscdk/v2/awsroute53targets"
	s3 "github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	s3Deploy "github.com/aws/aws-cdk-go/awscdk/v2/awss3deployment"
	parameterStore "github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkGoStackProps struct {
	cdk.StackProps
}

var appName = "bowerbird"

func NewCdkStack(scope constructs.Construct, id string, props *CdkGoStackProps) cdk.Stack {
	var sProps cdk.StackProps
	if props != nil {
		sProps = props.StackProps
	}
	stack := cdk.NewStack(scope, &id, &sProps)

	secrets := parameterStore.StringParameter_FromSecureStringParameterAttributes(stack, jsii.String("backend-secrets"), &parameterStore.SecureStringParameterAttributes{
		ParameterName: jsii.String("prod-bowerbird-backend"),
	})

	lambdaFn := lambda.NewDockerImageFunction(stack, jsii.String(appName+"-api"), &lambda.DockerImageFunctionProps{
		FunctionName: jsii.String(appName + "-api"),
		Code: lambda.DockerImageCode_FromImageAsset(jsii.String("."), &lambda.AssetImageCodeProps{
			File:     jsii.String("cmd/lambda-api/Dockerfile"),
			Platform: awsecrassets.Platform_LINUX_ARM64(),
			BuildArgs: &map[string]*string{
				"--progress":   jsii.String("plain"),
				"--no-cache":   jsii.String("true"),
				"--provenance": jsii.String("false"),
			},
		}),
		Architecture: lambda.Architecture_ARM_64(),
		Timeout:      cdk.Duration_Seconds(jsii.Number(28)),
		Environment: &map[string]*string{
			"PARAMETER_STORE_KEY_NAME": secrets.ParameterName(),
		},
	})

	secrets.GrantRead(lambdaFn)

	httpApi := apiGatewayV2.NewHttpApi(stack, jsii.String(appName+"http-api"), &apiGatewayV2.HttpApiProps{
		ApiName:            jsii.String(appName + "-api"),
		CreateDefaultStage: jsii.Bool(false),
	})

	integration := apiGatewayV2Integrations.NewHttpLambdaIntegration(
		jsii.String("LambdaProxyIntegration"),
		lambdaFn,
		&apiGatewayV2Integrations.HttpLambdaIntegrationProps{},
	)

	httpApi.AddRoutes(&apiGatewayV2.AddRoutesOptions{
		Path:        jsii.String("/api/{proxy+}"),
		Methods:     &[]apiGatewayV2.HttpMethod{apiGatewayV2.HttpMethod_ANY},
		Integration: integration,
	})

	httpApi.AddStage(jsii.String("DefaultStageSetup"), &apiGatewayV2.HttpStageOptions{
		StageName:              jsii.String("prod"),
		AutoDeploy:             jsii.Bool(true),
		DetailedMetricsEnabled: jsii.Bool(true),
	})

	certificate := certManager.Certificate_FromCertificateArn(stack, jsii.String(appName+"-acm-certificate"), jsii.String("arn:aws:acm:us-east-1:336301087573:certificate/048e55de-9012-4c68-ad1e-6eb5d05478df"))

	webappBucket := s3.NewBucket(stack, jsii.String(appName+"-webapp-bucket"), &s3.BucketProps{
		BucketName:        jsii.String(appName + "-webapp"),
		RemovalPolicy:     cdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
		PublicReadAccess:  jsii.Bool(false),
		EnforceSSL:        jsii.Bool(true),
	})

	// Deploy HTML files with no cache
	s3Deploy.NewBucketDeployment(stack, jsii.String(appName+"-html-deployment"), &s3Deploy.BucketDeploymentProps{
		Sources:           &[]s3Deploy.ISource{s3Deploy.Source_Asset(jsii.String("static/web-app/dist/bowerbird/browser"), nil)},
		DestinationBucket: webappBucket,
		CacheControl: &[]s3Deploy.CacheControl{
			s3Deploy.CacheControl_FromString(jsii.String("no-cache, no-store, must-revalidate")),
		},
		Include: &[]*string{
			jsii.String("*.html"),
		},
	})

	// Deploy static assets with short cache (1 hour)
	s3Deploy.NewBucketDeployment(stack, jsii.String(appName+"-static-assets-deployment"), &s3Deploy.BucketDeploymentProps{
		Sources:           &[]s3Deploy.ISource{s3Deploy.Source_Asset(jsii.String("static/web-app/dist/bowerbird/browser"), nil)},
		DestinationBucket: webappBucket,
		CacheControl: &[]s3Deploy.CacheControl{
			s3Deploy.CacheControl_FromString(jsii.String("public, max-age=3600")),
		},
		Include: &[]*string{
			jsii.String("*.js"),
			jsii.String("*.css"),
			jsii.String("*.png"),
			jsii.String("*.jpg"),
			jsii.String("*.jpeg"),
			jsii.String("*.gif"),
			jsii.String("*.svg"),
			jsii.String("*.ico"),
			jsii.String("*.woff"),
			jsii.String("*.woff2"),
			jsii.String("*.ttf"),
			jsii.String("*.eot"),
			jsii.String("*.json"),
			jsii.String("*.webmanifest"),
		},
		Metadata: &map[string]*string{
			"*.js":  jsii.String("application/javascript"),
			"*.css": jsii.String("text/css"),
		},
	})

	// Deploy service worker with longer cache (1 week)
	s3Deploy.NewBucketDeployment(stack, jsii.String(appName+"-service-worker-deployment"), &s3Deploy.BucketDeploymentProps{
		Sources:           &[]s3Deploy.ISource{s3Deploy.Source_Asset(jsii.String("static/web-app/dist/bowerbird/browser"), nil)},
		DestinationBucket: webappBucket,
		CacheControl: &[]s3Deploy.CacheControl{
			s3Deploy.CacheControl_FromString(jsii.String("public, max-age=604800")),
		},
		Include: &[]*string{
			jsii.String("ngsw.json"),
			jsii.String("worker-basic.min.js"),
		},
	})

	originAccessIdentity := cloudfront.NewOriginAccessIdentity(stack, jsii.String(appName+"-origin-access-identity"), &cloudfront.OriginAccessIdentityProps{})
	s3Origin := cloudfrontOrigins.NewS3Origin(webappBucket, &cloudfrontOrigins.S3OriginProps{
		OriginAccessIdentity: originAccessIdentity,
	})

	domainName := os.Getenv("APP_DOMAIN_NAME")

	distribution := cloudfront.NewDistribution(stack, jsii.String(appName+"-cdn"), &cloudfront.DistributionProps{
		DomainNames:       &[]*string{jsii.String(domainName)},
		Certificate:       certificate,
		DefaultRootObject: jsii.String("index.html"),
		DefaultBehavior: &cloudfront.BehaviorOptions{
			Origin:               s3Origin,
			ViewerProtocolPolicy: cloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			AllowedMethods:       cloudfront.AllowedMethods_ALLOW_GET_HEAD_OPTIONS(),
			CachedMethods:        cloudfront.CachedMethods_CACHE_GET_HEAD_OPTIONS(),
			CachePolicy:          cloudfront.CachePolicy_CACHING_OPTIMIZED(),
		},
		ErrorResponses: &[]*cloudfront.ErrorResponse{
			{
				HttpStatus:         jsii.Number(403),
				ResponsePagePath:   jsii.String("/index.html"),
				ResponseHttpStatus: jsii.Number(200),
				Ttl:                cdk.Duration_Seconds(jsii.Number(60)),
			},
			{
				HttpStatus:         jsii.Number(404),
				ResponsePagePath:   jsii.String("/index.html"),
				ResponseHttpStatus: jsii.Number(200),
				Ttl:                cdk.Duration_Seconds(jsii.Number(60)),
			},
		},
		AdditionalBehaviors: &map[string]*cloudfront.BehaviorOptions{
			"/api/*": {
				Origin: cloudfrontOrigins.NewHttpOrigin(jsii.Sprintf("%s.execute-api.%s.amazonaws.com", *httpApi.ApiId(), *stack.Region()), &cloudfrontOrigins.HttpOriginProps{
					OriginPath: jsii.String("/prod"),
				}),
				Compress:              jsii.Bool(true),
				ViewerProtocolPolicy:  cloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
				AllowedMethods:        cloudfront.AllowedMethods_ALLOW_ALL(),
				CachePolicy:           cloudfront.CachePolicy_CACHING_DISABLED(),
				OriginRequestPolicy:   cloudfront.OriginRequestPolicy_ALL_VIEWER_EXCEPT_HOST_HEADER(),
				ResponseHeadersPolicy: cloudfront.ResponseHeadersPolicy_CORS_ALLOW_ALL_ORIGINS_AND_SECURITY_HEADERS(),
			},
		},
	})

	hostedZone := route53.HostedZone_FromLookup(stack, jsii.String(appName+"-hosted-zone"), &route53.HostedZoneProviderProps{
		DomainName: jsii.String(domainName),
	})

	route53.NewARecord(stack, jsii.String(appName+"-cloudfront-alias-record"), &route53.ARecordProps{
		Zone:       hostedZone,
		Target:     route53.RecordTarget_FromAlias(route53Targets.NewCloudFrontTarget(distribution)),
		RecordName: jsii.String(""), // apex domain
		Ttl:        cdk.Duration_Seconds(jsii.Number(300)),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := cdk.NewApp(nil)

	NewCdkStack(app, appName+"-app", &CdkGoStackProps{
		cdk.StackProps{
			Env:         env(),
			StackName:   jsii.String(appName + "-app"),
			Description: jsii.String("Bowerbird App, finance assistant"),
		},
	})

	app.Synth(nil)
}

func env() *cdk.Environment {
	return &cdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("AWS_DEFAULT_REGION")),
	}
}
