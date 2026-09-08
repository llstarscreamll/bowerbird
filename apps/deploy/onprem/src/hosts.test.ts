import assert from 'node:assert/strict';
import test from 'node:test';
import { parseHosts } from './hosts.ts';

test('parseHosts accepts a raw array and fills defaults', () => {
  const hosts = parseHosts([{ id: 'acme', address: '203.0.113.10' }]);
  assert.deepEqual(hosts, [
    {
      id: 'acme',
      address: '203.0.113.10',
      user: 'bowerbird',
      port: 22,
      remoteDir: '/opt/bowerbird',
    },
  ]);
});

test('parseHosts accepts a wrapped hosts object', () => {
  const hosts = parseHosts({
    hosts: [{ id: 'globex', address: '203.0.113.20', user: 'ops', port: 2222, remoteDir: '/srv/bb' }],
  });
  assert.equal(hosts[0].user, 'ops');
  assert.equal(hosts[0].port, 2222);
  assert.equal(hosts[0].remoteDir, '/srv/bb');
});

test('parseHosts rejects a bad id', () => {
  assert.throws(() => parseHosts([{ id: 'Acme', address: '203.0.113.10' }]), /id must match/);
});

test('parseHosts rejects a missing address', () => {
  assert.throws(() => parseHosts([{ id: 'acme' }]), /address is required/);
});
