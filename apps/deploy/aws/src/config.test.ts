import assert from 'node:assert/strict';
import test from 'node:test';
import { resourcePrefix } from './names.ts';

test('resourcePrefix includes env and product name', () => {
  assert.equal(resourcePrefix('prod'), 'prod-bowerbird');
  assert.equal(resourcePrefix('staging'), 'staging-bowerbird');
});
