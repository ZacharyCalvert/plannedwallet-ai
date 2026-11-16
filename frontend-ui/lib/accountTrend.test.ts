import { describe, it } from 'node:test';
import assert from 'node:assert';
import type { AccountRecord } from './types.ts';
import {
  buildAccountBalanceSeries,
  filterBalanceSeriesByRange,
  computeBalanceMetrics,
  selectHighlightPoints,
} from './accountTrend.ts';

function makeRecord(id: string, dateSent: string, value: number): AccountRecord {
  return {
    id,
    account_id: 'acc-1',
    value,
    date_sent: dateSent,
    date_cleared: null,
    entity: 'Test Entity',
    created_at: `${dateSent}T12:00:00Z`,
    updated_at: `${dateSent}T12:00:00Z`,
  };
}

describe('Account Balance Trend Helpers', () => {
  it('computes cumulative end-of-day balance across chronologically sorted records', () => {
    const records = [
      makeRecord('3', '2026-11-10', -45.5),
      makeRecord('1', '2026-11-01', 1000),
      makeRecord('2', '2026-11-10', -100),
    ];
    const series = buildAccountBalanceSeries(records);
    assert.strictEqual(series.length, 2);
    assert.deepStrictEqual(series[0], {
      date: '2026-11-01',
      balance: 1000,
      delta: 1000,
      transactionCount: 1,
    });
    assert.deepStrictEqual(series[1], {
      date: '2026-11-10',
      balance: 854.5,
      delta: -145.5,
      transactionCount: 2,
    });
  });

  it('filters 30d, 60d, 90d, 1y, and all ranges with carry-forward starting balance', () => {
    const records = [
      makeRecord('1', '2025-05-01', 2000),
      makeRecord('2', '2026-07-01', -200), // 1800
      makeRecord('3', '2026-09-01', -100), // 1700
      makeRecord('4', '2026-10-01', -50),  // 1650
      makeRecord('5', '2026-11-01', 350),  // 2000
    ];
    const series = buildAccountBalanceSeries(records);
    const ref = '2026-11-16';

    assert.strictEqual(filterBalanceSeriesByRange(series, 'all', ref).length, 5);

    const d30 = filterBalanceSeriesByRange(series, '30d', ref);
    assert.strictEqual(d30[0].date, '2026-10-17');
    assert.strictEqual(d30[0].balance, 1650);
    assert.strictEqual(d30[1].date, '2026-11-01');
    assert.strictEqual(d30[1].balance, 2000);

    const d60 = filterBalanceSeriesByRange(series, '60d', ref);
    assert.strictEqual(d60[0].date, '2026-09-17');
    assert.strictEqual(d60[0].balance, 1700);
    assert.strictEqual(d60.length, 3);

    const d90 = filterBalanceSeriesByRange(series, '90d', ref);
    assert.strictEqual(d90[0].date, '2026-08-18');
    assert.strictEqual(d90[0].balance, 1800);
    assert.strictEqual(d90.length, 4);

    const y1 = filterBalanceSeriesByRange(series, '1y', ref);
    assert.strictEqual(y1[0].date, '2025-11-16');
    assert.strictEqual(y1[0].balance, 2000);
    assert.strictEqual(y1.length, 5);
  });

  it('draws flat carry-forward line when no transactions occurred inside selected window', () => {
    const records = [makeRecord('1', '2025-01-10', 1500)];
    const series = buildAccountBalanceSeries(records);
    const d30 = filterBalanceSeriesByRange(series, '30d', '2026-11-16');
    assert.strictEqual(d30.length, 2);
    assert.strictEqual(d30[0].date, '2026-10-17');
    assert.strictEqual(d30[0].balance, 1500);
    assert.strictEqual(d30[1].date, '2026-11-16');
    assert.strictEqual(d30[1].balance, 1500);
  });

  it('computes net change and min/max balance metrics', () => {
    const series = buildAccountBalanceSeries([
      makeRecord('1', '2026-11-01', 1000),
      makeRecord('2', '2026-11-05', -400),
      makeRecord('3', '2026-11-10', 600),
    ]);
    const metrics = computeBalanceMetrics(series);
    assert.ok(metrics);
    assert.strictEqual(metrics.first.balance, 1000);
    assert.strictEqual(metrics.latest.balance, 1200);
    assert.strictEqual(metrics.change, 200);
    assert.strictEqual(metrics.minVal, 600);
    assert.strictEqual(metrics.maxVal, 1200);
  });

  it('restricts highlight bubbles to at most 1 per 10 pixels on the X axis', () => {
    const dense = Array.from({ length: 31 }, (_, i) => ({ x: i * 1 }));
    const kept = selectHighlightPoints(dense, 10);
    assert.deepStrictEqual(
      kept.map((p) => p.x),
      [0, 10, 20, 30]
    );

    const uneven = [{ x: 0 }, { x: 4 }, { x: 10 }, { x: 13 }, { x: 20 }, { x: 25 }, { x: 32 }];
    const keptUneven = selectHighlightPoints(uneven, 10);
    for (let i = 1; i < keptUneven.length; i++) {
      assert.ok(keptUneven[i].x - keptUneven[i - 1].x >= 10);
    }
    assert.strictEqual(keptUneven[keptUneven.length - 1].x, 32);
  });
});
