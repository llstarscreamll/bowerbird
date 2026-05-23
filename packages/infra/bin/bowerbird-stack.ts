import * as path from 'node:path';
import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';
import * as apigwv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as events from 'aws-cdk-lib/aws-events';
import * as eventTargets from 'aws-cdk-lib/aws-events-targets';
import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import * as route53 from 'aws-cdk-lib/aws-route53';
import * as route53Targets from 'aws-cdk-lib/aws-route53-targets';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as ssm from 'aws-cdk-lib/aws-ssm';
import * as lambdaEventSources from 'aws-cdk-lib/aws-lambda-event-sources';

export interface BowerbirdStackProps extends cdk.StackProps {
  envName: string;
}

export class BowerbirdStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: BowerbirdStackProps) {
    super(scope, id, props);

    const prefix = `${props.envName}-bowerbird`;

    const rootDomain = process.env.ROOT_DOMAIN ?? 'money-path.co';
    const appSubdomain = process.env.APP_SUBDOMAIN ?? 'app';
    const apiSubdomain = process.env.API_SUBDOMAIN ?? 'api';
    const appDomain = appSubdomain ? `${appSubdomain}.${rootDomain}` : rootDomain;
    const apiDomain = `${apiSubdomain}.${rootDomain}`;

    const zone = route53.HostedZone.fromLookup(this, 'HostedZone', {
      domainName: rootDomain,
    });

    const webCert = new acm.Certificate(this, 'WebCertificate', {
      domainName: appDomain,
      validation: acm.CertificateValidation.fromDns(zone),
    });

    const apiCert = new acm.Certificate(this, 'ApiCertificate', {
      domainName: apiDomain,
      validation: acm.CertificateValidation.fromDns(zone),
    });

    const websiteBucket = new s3.Bucket(this, 'WebBucket', {
      bucketName: `${prefix}-web-app`,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      enforceSSL: true,
      versioned: true,
    });

    const ssmParameterName = `/bowerbird/${props.envName}/secrets`;

    const httpLambda = new GoFunction(this, 'ApiHttpLambda', {
      functionName: `${prefix}-api`,
      entry: path.join(__dirname, '../../../apps/api/cmd/lambda/http'),
      architecture: cdk.aws_lambda.Architecture.ARM_64,
      timeout: cdk.Duration.seconds(10),
      environment: {
        GOOS: 'linux',
        SSM_PARAMETER_NAME: ssmParameterName,
      },
    });

    const sqsLambda = new GoFunction(this, 'ApiSQSLambda', {
      functionName: `${prefix}-sqs-processor`,
      entry: path.join(__dirname, '../../../apps/api/cmd/lambda/sqs'),
      architecture: cdk.aws_lambda.Architecture.ARM_64,
      timeout: cdk.Duration.seconds(10),
      environment: {
        SSM_PARAMETER_NAME: ssmParameterName,
      },
    });

    const eventBridgeLambda = new GoFunction(this, 'ApiEventBridgeLambda', {
      functionName: `${prefix}-events-processor`,
      entry: path.join(__dirname, '../../../apps/api/cmd/lambda/eventbridge'),
      architecture: cdk.aws_lambda.Architecture.ARM_64,
      timeout: cdk.Duration.seconds(10),
      environment: {
        SSM_PARAMETER_NAME: ssmParameterName,
      },
    });

    const secretsParam = ssm.StringParameter.fromSecureStringParameterAttributes(this, 'SecretsParam', {
      parameterName: ssmParameterName,
    });
    secretsParam.grantRead(httpLambda);
    secretsParam.grantRead(sqsLambda);
    secretsParam.grantRead(eventBridgeLambda);

    const queue = new sqs.Queue(this, 'BowerbirdQueue', {
      queueName: `${prefix}-queue`,
      visibilityTimeout: cdk.Duration.seconds(30),
      encryption: sqs.QueueEncryption.SQS_MANAGED,
    });

    sqsLambda.addEventSource(new lambdaEventSources.SqsEventSource(queue, { batchSize: 10 }));

    const eventRule = new events.Rule(this, 'BowerbirdEventRule', {
      ruleName: `${prefix}-app-events`,
      eventPattern: {
        source: ['bowerbird.app'],
      },
    });

    eventRule.addTarget(new eventTargets.LambdaFunction(eventBridgeLambda));

    const httpApi = new apigwv2.HttpApi(this, 'BowerbirdHttpApi', {
      apiName: `${prefix}-http-api`,
      corsPreflight: {
        allowHeaders: ['Content-Type', 'Authorization'],
        allowMethods: [apigwv2.CorsHttpMethod.ANY],
        allowOrigins: ['https://' + appDomain],
      },
      disableExecuteApiEndpoint: false,
    });

    httpApi.addRoutes({
      path: '/{proxy+}',
      methods: [apigwv2.HttpMethod.ANY],
      integration: new integrations.HttpLambdaIntegration('ProxyIntegration', httpLambda),
    });

    const apiDomainName = new apigwv2.DomainName(this, 'ApiDomain', {
      domainName: apiDomain,
      certificate: apiCert,
    });

    new apigwv2.ApiMapping(this, 'ApiMapping', {
      api: httpApi,
      domainName: apiDomainName,
      stage: httpApi.defaultStage,
    });

    const responseHeadersPolicy = new cloudfront.ResponseHeadersPolicy(this, 'SpaSecurityHeaders', {
      responseHeadersPolicyName: `${prefix}-spa-headers`,
      securityHeadersBehavior: {
        contentSecurityPolicy: {
          contentSecurityPolicy:
            "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' https://" +
            apiDomain,
          override: true,
        },
        contentTypeOptions: { override: true },
        frameOptions: { frameOption: cloudfront.HeadersFrameOption.DENY, override: true },
        referrerPolicy: {
          referrerPolicy: cloudfront.HeadersReferrerPolicy.STRICT_ORIGIN_WHEN_CROSS_ORIGIN,
          override: true,
        },
        strictTransportSecurity: {
          accessControlMaxAge: cdk.Duration.days(365),
          includeSubdomains: true,
          preload: true,
          override: true,
        },
        xssProtection: { protection: true, modeBlock: true, override: true },
      },
    });

    const distribution = new cloudfront.Distribution(this, 'WebDistribution', {
      comment: `${prefix}-web-distribution`,
      defaultBehavior: {
        origin: origins.S3BucketOrigin.withOriginAccessControl(websiteBucket),
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
        cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
        responseHeadersPolicy,
      },
      additionalBehaviors: {
        '/api/*': {
          origin: new origins.HttpOrigin(apiDomain, {
            protocolPolicy: cloudfront.OriginProtocolPolicy.HTTPS_ONLY,
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
          cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
        },
      },
      defaultRootObject: 'index.html',
      domainNames: [appDomain],
      certificate: webCert,
      errorResponses: [
        {
          httpStatus: 403,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.minutes(1),
        },
        {
          httpStatus: 404,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.minutes(1),
        },
      ],
    });

    const webBuildPath = path.join(__dirname, '../../../apps/web/dist/web/browser');

    new s3deploy.BucketDeployment(this, 'WebStaticAssetsDeployment', {
      destinationBucket: websiteBucket,
      prune: false,
      cacheControl: [
        s3deploy.CacheControl.fromString('public'),
        s3deploy.CacheControl.maxAge(cdk.Duration.days(365)),
        s3deploy.CacheControl.immutable(),
      ],
      sources: [
        s3deploy.Source.asset(webBuildPath, {
          exclude: ['index.html', 'ngsw.json', 'ngsw-worker.js', 'safety-worker.js', 'manifest.webmanifest'],
        }),
      ],
    });

    new s3deploy.BucketDeployment(this, 'WebEntryPointsDeployment', {
      destinationBucket: websiteBucket,
      prune: false,
      distribution,
      distributionPaths: ['/index.html', '/ngsw.json', '/ngsw-worker.js', '/safety-worker.js', '/manifest.webmanifest'],
      cacheControl: [
        s3deploy.CacheControl.fromString('public'),
        s3deploy.CacheControl.maxAge(cdk.Duration.seconds(0)),
        s3deploy.CacheControl.mustRevalidate(),
        s3deploy.CacheControl.fromString('s-maxage=300'),
      ],
      sources: [
        s3deploy.Source.asset(path.join(webBuildPath, 'index.html')),
        s3deploy.Source.asset(path.join(webBuildPath, 'ngsw.json')),
        s3deploy.Source.asset(path.join(webBuildPath, 'ngsw-worker.js')),
        s3deploy.Source.asset(path.join(webBuildPath, 'safety-worker.js')),
        s3deploy.Source.asset(path.join(webBuildPath, 'manifest.webmanifest')),
      ],
    });

    new route53.ARecord(this, 'WebAliasRecord', {
      zone,
      recordName: appSubdomain || undefined,
      target: route53.RecordTarget.fromAlias(new route53Targets.CloudFrontTarget(distribution)),
    });

    new route53.ARecord(this, 'ApiAliasRecord', {
      zone,
      recordName: apiSubdomain,
      target: route53.RecordTarget.fromAlias(
        new route53Targets.ApiGatewayv2DomainProperties(
          apiDomainName.regionalDomainName,
          apiDomainName.regionalHostedZoneId,
        ),
      ),
    });

    new cdk.CfnOutput(this, 'WebUrl', { value: `https://${appDomain}` });
    new cdk.CfnOutput(this, 'ApiUrl', { value: `https://${apiDomain}` });
    new cdk.CfnOutput(this, 'QueueUrl', { value: queue.queueUrl });
  }
}
