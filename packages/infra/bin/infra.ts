#!/usr/bin/env node
import 'source-map-support/register';
import * as dotenv from 'dotenv';
import * as cdk from 'aws-cdk-lib';
import { TurnoStack } from '../lib/turno-stack';

dotenv.config({ path: '.env' });

const app = new cdk.App();

const account = process.env.AWS_ACCOUNT_ID;
const region = process.env.AWS_REGION ?? 'us-east-1';

if (!account) {
  throw new Error('AWS_ACCOUNT_ID is required in packages/infra/.env');
}

if (region !== 'us-east-1') {
  throw new Error('AWS_REGION must be us-east-1 because CloudFront certificates are created in this stack');
}

new TurnoStack(app, 'TurnoStack', {
  env: {
    account,
    region
  }
});
