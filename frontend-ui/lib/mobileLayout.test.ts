import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  MOBILE_BREAKPOINT_PX,
  getMobileHeaderTitle,
  isMobileViewportWidth,
} from './mobileLayout.ts';

describe('mobileLayout helpers', () => {
  it('classifies mobile widths cleanly at the 768px breakpoint', () => {
    assert.equal(isMobileViewportWidth(375), true);
    assert.equal(isMobileViewportWidth(MOBILE_BREAKPOINT_PX), true);
    assert.equal(isMobileViewportWidth(MOBILE_BREAKPOINT_PX + 1), false);
    assert.equal(isMobileViewportWidth(1280), false);
    assert.equal(isMobileViewportWidth(0), false);
    assert.equal(isMobileViewportWidth(Number.NaN), false);
  });

  it('resolves human-readable mobile top bar titles across all dashboard views', () => {
    const accounts = [{ id: 'a1', name: 'Primary Checking' }];
    const perspectives = [{ id: 'p1', name: 'Household View' }];

    assert.equal(
      getMobileHeaderTitle({ kind: 'account', accountId: 'a1' }, accounts, perspectives),
      'Primary Checking'
    );
    assert.equal(
      getMobileHeaderTitle({ kind: 'account', accountId: 'missing' }, accounts, perspectives),
      'Account'
    );
    assert.equal(
      getMobileHeaderTitle({ kind: 'perspective', perspectiveId: 'p1' }, accounts, perspectives),
      'Household View'
    );
    assert.equal(getMobileHeaderTitle({ kind: 'bills' }, accounts, perspectives), 'Bills');
    assert.equal(getMobileHeaderTitle({ kind: 'incomes' }, accounts, perspectives), 'Income');
    assert.equal(getMobileHeaderTitle({ kind: 'new-account' }, accounts, perspectives), 'New Account');
    assert.equal(getMobileHeaderTitle({ kind: 'new-perspective' }, accounts, perspectives), 'New Perspective');
    assert.equal(getMobileHeaderTitle({ kind: 'user' }, accounts, perspectives), 'User Details');
    assert.equal(getMobileHeaderTitle({ kind: 'home' }, accounts, perspectives), 'Dashboard');
  });
});
