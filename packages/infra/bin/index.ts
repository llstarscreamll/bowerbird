#!/usr/bin/env node
import 'source-map-support/register';
import * as dotenv from 'dotenv';
import * as cdk from 'aws-cdk-lib';
import { BowerbirdStack } from './bowerbird-stack';

dotenv.config({ path: '.env' });

const app = new cdk.App();

const envName = process.env.ENV;
const account = process.env.AWS_ACCOUNT_ID;
const region = process.env.AWS_REGION ?? 'us-east-1';

if (!envName) {
  throw new Error('ENV is required in packages/infra/.env');
}

if (!account) {
  throw new Error('AWS_ACCOUNT_ID is required in packages/infra/.env');
}

if (region !== 'us-east-1') {
  throw new Error('AWS_REGION must be us-east-1 because CloudFront certificates are created in this stack');
}

new BowerbirdStack(app, 'BowerbirdStack', {
  envName,
  env: {
    account,
    region,
  },
});
