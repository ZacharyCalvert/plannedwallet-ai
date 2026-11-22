import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  computePerspectiveCategoryPie,
  computeRunwayProjection,
  filterPerspectiveTrendByRange,
} from './perspectives.ts';
import type { AccountRecord, Category, RecurringTransaction } from './types.ts';

describe('Perspective Calculations Helpers', () => {
  const sampleRecurring: RecurringTransaction[] = [
    {
      id: 'bill-1',
      user_id: 'u1',
      account_id: 'a1',
      category_id: 'cat-util',
      amount: -304.375,
      last_date_applied: null,
      frequency: 'monthly',
      interval: 1,
      schedule: null,
      name: 'Power Company',
      entity: 'Power Company',
      description: 'Electric',
      reminder: false,
      next_due_date: null,
      due_status: 'unscheduled',
      days_until_due: null,
      created_at: '',
      updated_at: '',
    },
    {
      id: 'inc-1',
      user_id: 'u1',
      account_id: 'a1',
      category_id: 'cat-inc',
      amount: 140,
      last_date_applied: null,
      frequency: 'weekly',
      interval: 1,
      schedule: null,
      name: 'Side Income',
      entity: 'Side Income',
      description: 'Weekly payout',
      reminder: false,
      next_due_date: null,
      due_status: 'unscheduled',
      days_until_due: null,
      created_at: '',
      updated_at: '',
    },
  ];

  it('computes projected $0 runway date from balance, custom daily spend, and enabled bills/income', () => {
    // Daily income = 140 / 7 = 20/day; daily bill = 304.375 * 12 / 365.25 ~= 10/day
    // Net recurring = +10/day; custom daily spend = 30/day => net daily burn = -20/day
    const proj = computeRunwayProjection(2000, 30, sampleRecurring, new Set(), '2026-01-01');
    assert.equal(proj.isSustainable, false);
    assert.equal(proj.daysUntilZero, 100);
    assert.equal(proj.projectedZeroDate, '2026-04-11');
    assert.equal(proj.points[0].totalValue, 2000);
    assert.equal(proj.points[proj.points.length - 1].totalValue, 0);

    // Disabling the weekly income increases burn to ~40/day => ~50 days
    const withoutIncome = computeRunwayProjection(2000, 30, sampleRecurring, new Set(['inc-1']), '2026-01-01');
    assert.equal(withoutIncome.isSustainable, false);
    assert.equal(withoutIncome.daysUntilZero, 50);
    assert.equal(withoutIncome.projectedZeroDate, '2026-02-20');
  });

  it('marks runway as sustainable when income covers bills and daily spend', () => {
    const proj = computeRunwayProjection(5000, 5, sampleRecurring, new Set(), '2026-01-01');
    assert.equal(proj.isSustainable, true);
    assert.equal(proj.daysUntilZero, null);
    assert.equal(proj.projectedZeroDate, null);
  });

  it('filters perspective trend by range and includes linked recurring items in category pie', () => {
    const trend = [
      { date: '2025-01-01', total_value: 1000 },
      { date: '2026-10-15', total_value: 1500 },
      { date: '2026-11-10', total_value: 1800 },
    ];
    const thirty = filterPerspectiveTrendByRange(trend, '30d', '2026-11-15');
    assert.deepEqual(
      thirty.map((p) => p.date),
      ['2026-11-10']
    );

    const cats: Category[] = [
      { id: 'cat-util', name: 'Utilities', is_default: true, created_at: '', updated_at: '' },
      { id: 'cat-inc', name: 'Income', is_default: true, created_at: '', updated_at: '' },
    ];
    const accRecs: AccountRecord[] = [
      {
        id: 'r1',
        account_id: 'a1',
        category_id: 'cat-util',
        value: -100,
        date_sent: '2026-11-10',
        date_cleared: null,
        entity: 'Water',
        description: '',
        created_at: '',
        updated_at: '',
      },
    ];
    const pie = computePerspectiveCategoryPie(
      accRecs,
      sampleRecurring,
      cats,
      'all',
      true,
      new Set(),
      '2026-11-15'
    );
    assert.equal(pie.slices.length, 2);
    assert.equal(pie.slices[0].categoryName, 'Utilities');
  });

  it('builds non-overlapping X-axis labels across historical and runway projection span', async () => {
    const { buildNonOverlappingXAxisLabels } = await import('./perspectives.ts');
    const t0 = new Date('2026-08-01').getTime();
    const t1 = new Date('2027-10-19').getTime();
    const labels = buildNonOverlappingXAxisLabels(t0, t1, 85, 710, (iso) => iso.slice(0, 7), 85);

    assert.ok(labels.length >= 2 && labels.length <= 7);
    assert.equal(labels[0].label, '2026-08');
    assert.equal(labels[labels.length - 1].label, '2027-10');
    for (let i = 1; i < labels.length; i++) {
      assert.ok(labels[i].x - labels[i - 1].x >= 85);
    }
  });
});
