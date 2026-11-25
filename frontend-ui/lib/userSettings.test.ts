import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  applyAccountOrder,
  isPinnedBalance,
  reorderItems,
  togglePinnedBalance,
} from './userSettings.ts';

describe('userSettings helpers', () => {
  it('applies saved account order and appends new accounts at the end', () => {
    const accounts = [{ id: 'a1' }, { id: 'a2' }, { id: 'a3' }, { id: 'a4' }];
    const ordered = applyAccountOrder(accounts, ['a3', 'stale-id', 'a1']);
    assert.deepEqual(
      ordered.map((a) => a.id),
      ['a3', 'a1', 'a2', 'a4']
    );
  });

  it('reorders items cleanly by index', () => {
    assert.deepEqual(reorderItems(['a', 'b', 'c'], 2, 0), ['c', 'a', 'b']);
    assert.deepEqual(reorderItems(['a', 'b', 'c'], 0, 2), ['b', 'c', 'a']);
    assert.deepEqual(reorderItems(['a', 'b'], 1, 1), ['a', 'b']);
  });

  it('toggles pinned balance references cleanly', () => {
    const initial = [{ kind: 'account' as const, id: 'acc-1' }];
    assert.equal(isPinnedBalance(initial, 'account', 'acc-1'), true);
    assert.equal(isPinnedBalance(initial, 'perspective', 'acc-1'), false);

    const added = togglePinnedBalance(initial, { kind: 'perspective', id: 'persp-1' });
    assert.equal(added.length, 2);

    const removed = togglePinnedBalance(added, { kind: 'account', id: 'acc-1' });
    assert.deepEqual(removed, [{ kind: 'perspective', id: 'persp-1' }]);
  });
});
