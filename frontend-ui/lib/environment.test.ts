import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { STAGE_HOSTNAME, isStagingHost } from './environment.ts';

describe('Environment & Staging Host Helpers', () => {
  it('detects stage.plannedwallet.com case-insensitively', () => {
    assert.strictEqual(isStagingHost(STAGE_HOSTNAME), true);
    assert.strictEqual(isStagingHost('STAGE.PLANNEDWALLET.COM'), true);
    assert.strictEqual(isStagingHost('stage.plannedwallet.com:8443'), true);
    assert.strictEqual(isStagingHost('  stage.plannedwallet.com  '), true);
  });

  it('returns false for production, localhost, subdomains, or empty values', () => {
    assert.strictEqual(isStagingHost('plannedwallet.com'), false);
    assert.strictEqual(isStagingHost('app.plannedwallet.com'), false);
    assert.strictEqual(isStagingHost('localhost'), false);
    assert.strictEqual(isStagingHost('127.0.0.1'), false);
    assert.strictEqual(isStagingHost(''), false);
    assert.strictEqual(isStagingHost(null), false);
    assert.strictEqual(isStagingHost(undefined), false);
  });
});
