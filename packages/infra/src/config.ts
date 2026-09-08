import * as fs from 'node:fs';
import * as path from 'node:path';
import dotenv from 'dotenv';

const repoRoot = path.resolve(__dirname, '../../..');
const envPath = path.join(repoRoot, '.env');
if (fs.existsSync(envPath)) {
  dotenv.config({ path: envPath });
}

export interface InfraConfig {
  envName: string;
  isProd: boolean;
  awsAccountId: string;
  awsRegion: 'us-east-1';
  rootDomain: string;
  appSubdomain: string;
  mediaSubdomain: string;
  appDomain: string;
  mediaDomain: string;
  cloudflareApiToken: string;
  neonApiKey: string;
  neonOrgId?: string;
  neonRegionId: string;
  neonPgVersion: number;
  geminiApiKey: string;
  geminiModel: string;
  geminiEndpoint: string;
  googleClientId: string;
  googleClientSecret: string;
  microsoftClientId: string;
  microsoftClientSecret: string;
  alarmEmail?: string;
  repoRoot: string;
  webBuildPath: string;
  lambdaBuildDir: string;
}

function required(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`${name} is required in the repo-root .env`);
  }
  return value;
}

function optional(name: string, fallback = ''): string {
  return process.env[name]?.trim() ?? fallback;
}

export function loadConfig(): InfraConfig {
  const envName = required('ENV');
  const awsRegion = optional('AWS_REGION', 'us-east-1');
  if (awsRegion !== 'us-east-1') {
    throw new Error('AWS_REGION must be us-east-1 so CloudFront certificates stay in this stack');
  }

  const rootDomain = required('ROOT_DOMAIN');
  const appSubdomain = optional('APP_SUBDOMAIN', 'app');
  const mediaSubdomain = optional('MEDIA_SUBDOMAIN', 'media');

  return {
    envName,
    isProd: envName === 'prod' || envName === 'production',
    awsAccountId: required('AWS_ACCOUNT_ID'),
    awsRegion: awsRegion as 'us-east-1',
    rootDomain,
    appSubdomain,
    mediaSubdomain,
    appDomain: `${appSubdomain}.${rootDomain}`,
    mediaDomain: `${mediaSubdomain}.${rootDomain}`,
    cloudflareApiToken: required('CLOUDFLARE_API_TOKEN'),
    neonApiKey: required('NEON_API_KEY'),
    neonOrgId: optional('NEON_ORG_ID') || undefined,
    neonRegionId: optional('NEON_REGION_ID', 'aws-us-east-1'),
    neonPgVersion: Number(optional('NEON_PG_VERSION', '16')),
    geminiApiKey: required('GEMINI_API_KEY'),
    geminiModel: optional('GEMINI_MODEL', 'gemini-2.0-flash'),
    geminiEndpoint: optional('GEMINI_ENDPOINT', 'https://generativelanguage.googleapis.com'),
    googleClientId: optional('GOOGLE_CLIENT_ID'),
    googleClientSecret: optional('GOOGLE_CLIENT_SECRET'),
    microsoftClientId: optional('MICROSOFT_CLIENT_ID'),
    microsoftClientSecret: optional('MICROSOFT_CLIENT_SECRET'),
    alarmEmail: optional('ALARM_EMAIL') || undefined,
    repoRoot,
    webBuildPath: path.join(repoRoot, 'apps/pwa/dist/pwa/browser'),
    lambdaBuildDir: path.join(__dirname, '..', '.build', 'lambda'),
  };
}

export { defaultTags, resourcePrefix } from './names';
