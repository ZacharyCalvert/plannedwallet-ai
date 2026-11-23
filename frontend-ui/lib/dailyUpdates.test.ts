import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  sortDailyUpdateItems,
  buildPendingSnapshotItem,
  buildPendingAssetItem,
} from './dailyUpdates.ts';
import type { DailyUpdateItem } from './dailyUpdates.ts';
import type { Account, AccountSnapshot, AssetValue } from './types.ts';

function makeAccount(id: string, name: string, type = 'snapshot'): Account {
  return {
    id,
    admin_id: 'user-1',
    name,
    type,
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function makeSnapshot(id: string, snapshot_date: string, value: number): AccountSnapshot {
  return {
    id,
    account_id: 'acc-1',
    value,
    snapshot_date,
    created_at: `${snapshot_date}T12:00:00Z`,
    updated_at: `${snapshot_date}T12:00:00Z`,
  };
}

function makeAssetValue(id: string, value_date: string, value: number): AssetValue {
  return {
    id,
    asset_id: 'asset-1',
    value,
    value_date,
    created_at: `${value_date}T12:00:00Z`,
    updated_at: `${value_date}T12:00:00Z`,
  };
}

describe('dailyUpdates', () => {
  it('sorts snapshots first, then assets, subordered by name then id', () => {
    const unsorted: DailyUpdateItem[] = [
      { kind: 'asset', id: 'asset-b', name: 'Silver', subtitle: '', lastValue: null, lastDate: null },
      { kind: 'snapshot', id: 'snap-2', name: 'Brokerage', subtitle: '', lastValue: null, lastDate: null },
      { kind: 'asset', id: 'asset-a', name: 'Gold', subtitle: '', lastValue: null, lastDate: null },
      { kind: 'snapshot', id: 'snap-1', name: 'Brokerage', subtitle: '', lastValue: null, lastDate: null },
      { kind: 'snapshot', id: 'snap-3', name: 'Alpha Fund', subtitle: '', lastValue: null, lastDate: null },
      { kind: 'asset', id: 'asset-c', name: 'Gold', subtitle: '', lastValue: null, lastDate: null },
    ];

    const sorted = sortDailyUpdateItems(unsorted);
    assert.deepEqual(
      sorted.map((item) => `${item.kind}:${item.name}:${item.id}`),
      [
        'snapshot:Alpha Fund:snap-3',
        'snapshot:Brokerage:snap-1',
        'snapshot:Brokerage:snap-2',
        'asset:Gold:asset-a',
        'asset:Gold:asset-c',
        'asset:Silver:asset-b',
      ]
    );
  });

  it('excludes snapshot accounts that already have a record for today', () => {
    const acc = makeAccount('snap-1', 'Trading Portfolio', 'snapshot');
    const today = '2026-11-23';

    const pendingWhenEmpty = buildPendingSnapshotItem(acc, [], today);
    assert.ok(pendingWhenEmpty);
    assert.equal(pendingWhenEmpty.lastValue, null);

    const pendingWhenOnlyOld = buildPendingSnapshotItem(
      acc,
      [makeSnapshot('s-1', '2026-11-22', 12500)],
      today
    );
    assert.ok(pendingWhenOnlyOld);
    assert.equal(pendingWhenOnlyOld.lastValue, 12500);
    assert.equal(pendingWhenOnlyOld.lastDate, '2026-11-22');

    const noneWhenUpdatedToday = buildPendingSnapshotItem(
      acc,
      [makeSnapshot('s-1', '2026-11-22', 12500), makeSnapshot('s-2', '2026-11-23T09:00:00Z', 12750)],
      today
    );
    assert.equal(noneWhenUpdatedToday, null);
  });

  it('excludes asset classes that already have a value record for today', () => {
    const today = '2026-11-23';

    const pendingWhenStale = buildPendingAssetItem(
      'asset-1',
      'Gold Coin',
      ['Precious Metals'],
      [makeAssetValue('v-1', '2026-11-20', 2600)],
      today
    );
    assert.ok(pendingWhenStale);
    assert.equal(pendingWhenStale.subtitle, 'Used in Precious Metals');
    assert.equal(pendingWhenStale.lastValue, 2600);
    assert.equal(pendingWhenStale.lastDate, '2026-11-20');

    const noneWhenUpdatedToday = buildPendingAssetItem(
      'asset-1',
      'Gold Coin',
      ['Precious Metals', 'Safe Deposit'],
      [makeAssetValue('v-1', '2026-11-23', 2650)],
      today
    );
    assert.equal(noneWhenUpdatedToday, null);
  });
});
