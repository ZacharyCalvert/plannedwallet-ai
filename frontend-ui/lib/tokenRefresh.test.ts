import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  parseJwtExpirationMs,
  shouldRefreshTokenOnActivity,
  TOKEN_REFRESH_FAILURE_COOLDOWN_MS,
  TOKEN_REFRESH_THRESHOLD_MS,
} from './tokenRefresh.ts';

function buildFakeJwt(payload: Record<string, unknown>): string {
  const encode = (obj: Record<string, unknown>) =>
    Buffer.from(JSON.stringify(obj), 'utf8').toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(payload)}.signature`;
}

describe('Token Refresh Helpers', () => {
  it('parses JWT expiration timestamp in milliseconds', () => {
    const token = buildFakeJwt({ sub: 'u1', exp: 1_800_000_900 });
    assert.equal(parseJwtExpirationMs(token), 1_800_000_900_000);
    assert.equal(parseJwtExpirationMs('not-a-jwt'), null);
    assert.equal(parseJwtExpirationMs(buildFakeJwt({ sub: 'u1' })), null);
  });

  it('triggers refresh only when focused, valid, and under 5 minutes remaining', () => {
    const nowMs = 1_800_000_000_000;
    const expTenMinLeft = Math.floor((nowMs + 10 * 60 * 1000) / 1000);
    const expExactlyFiveMin = Math.floor((nowMs + TOKEN_REFRESH_THRESHOLD_MS) / 1000);
    const expFourMinLeft = Math.floor((nowMs + 4 * 60 * 1000) / 1000);
    const expExpired = Math.floor((nowMs - 10_000) / 1000);

    assert.equal(
      shouldRefreshTokenOnActivity({
        token: buildFakeJwt({ exp: expTenMinLeft }),
        isAppFocused: true,
        inFlight: false,
        nowMs,
      }),
      false
    );

    assert.equal(
      shouldRefreshTokenOnActivity({
        token: buildFakeJwt({ exp: expExactlyFiveMin }),
        isAppFocused: true,
        inFlight: false,
        nowMs,
      }),
      false
    );

    assert.equal(
      shouldRefreshTokenOnActivity({
        token: buildFakeJwt({ exp: expFourMinLeft }),
        isAppFocused: true,
        inFlight: false,
        nowMs,
      }),
      true
    );

    assert.equal(
      shouldRefreshTokenOnActivity({
        token: buildFakeJwt({ exp: expFourMinLeft }),
        isAppFocused: false,
        inFlight: false,
        nowMs,
      }),
      false
    );

    assert.equal(
      shouldRefreshTokenOnActivity({
        token: buildFakeJwt({ exp: expFourMinLeft }),
        isAppFocused: true,
        inFlight: true,
        nowMs,
      }),
      false
    );

    assert.equal(
      shouldRefreshTokenOnActivity({
        token: buildFakeJwt({ exp: expExpired }),
        isAppFocused: true,
        inFlight: false,
        nowMs,
      }),
      false
    );
  });

  it('respects cooldown after a failed refresh attempt', () => {
    const nowMs = 1_800_000_000_000;
    const token = buildFakeJwt({ exp: Math.floor((nowMs + 120_000) / 1000) });

    assert.equal(
      shouldRefreshTokenOnActivity({
        token,
        isAppFocused: true,
        inFlight: false,
        nowMs,
        lastFailureAtMs: nowMs - 5_000,
      }),
      false
    );

    assert.equal(
      shouldRefreshTokenOnActivity({
        token,
        isAppFocused: true,
        inFlight: false,
        nowMs,
        lastFailureAtMs: nowMs - TOKEN_REFRESH_FAILURE_COOLDOWN_MS,
      }),
      true
    );
  });
});
