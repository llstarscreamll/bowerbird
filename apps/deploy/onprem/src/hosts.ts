import * as fs from 'node:fs';

const HOST_ID = /^[a-z0-9][a-z0-9-]*$/;

export interface Host {
  id: string;
  address: string;
  user: string;
  port: number;
  remoteDir: string;
}

export function loadHosts(filePath: string): Host[] {
  if (!fs.existsSync(filePath)) {
    return [];
  }
  const raw: unknown = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  return parseHosts(raw);
}

export function parseHosts(raw: unknown): Host[] {
  const list = Array.isArray(raw) ? raw : asRecord(raw).hosts;
  if (!Array.isArray(list)) {
    throw new Error('hosts file must be an array or { "hosts": [...] }');
  }
  return list.map((entry, i) => normalizeHost(entry, i));
}

function normalizeHost(entry: unknown, index: number): Host {
  const rec = asRecord(entry);
  const id = requiredString(rec, 'id', index);
  if (!HOST_ID.test(id)) {
    throw new Error(`hosts[${index}].id must match ${HOST_ID}`);
  }
  const address = requiredString(rec, 'address', index);
  const port = optionalPort(rec.port, index);
  return {
    id,
    address,
    user: optionalString(rec.user) || 'bowerbird',
    port,
    remoteDir: optionalString(rec.remoteDir) || '/opt/bowerbird',
  };
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('expected an object');
  }
  return value as Record<string, unknown>;
}

function requiredString(rec: Record<string, unknown>, key: string, index: number): string {
  const value = optionalString(rec[key]);
  if (!value) {
    throw new Error(`hosts[${index}].${key} is required`);
  }
  return value;
}

function optionalString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

function optionalPort(value: unknown, index: number): number {
  if (value === undefined || value === null || value === '') {
    return 22;
  }
  const port = typeof value === 'number' ? value : Number(value);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`hosts[${index}].port must be an integer 1–65535`);
  }
  return port;
}
