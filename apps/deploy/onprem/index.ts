import * as fs from 'node:fs';
import * as path from 'node:path';
import dotenv from 'dotenv';
import { deployFleet } from './src/fleet';
import { loadHosts } from './src/hosts';

const repoRoot = path.resolve(__dirname, '../../..');
const envPath = path.join(repoRoot, '.env');
if (fs.existsSync(envPath)) {
  dotenv.config({ path: envPath });
}

const hostsFile = process.env.ONPREM_HOSTS_FILE?.trim() || path.join(repoRoot, 'apps/deploy/onprem/hosts.json');
const hosts = loadHosts(hostsFile);
const release = process.env.ONPREM_RELEASE?.trim() ?? '';
const sshKeyPath = process.env.ONPREM_SSH_KEY_PATH?.trim() ?? '';

if (hosts.length > 0 && !release) {
  throw new Error('ONPREM_RELEASE is required when the on-prem host inventory is non-empty');
}
if (hosts.length > 0 && !sshKeyPath) {
  throw new Error('ONPREM_SSH_KEY_PATH is required when the on-prem host inventory is non-empty');
}

const outputs = deployFleet({ hosts, release, repoRoot, sshKeyPath });

export const hostCount = hosts.length;
export const hostReleases = outputs.hostReleases;
