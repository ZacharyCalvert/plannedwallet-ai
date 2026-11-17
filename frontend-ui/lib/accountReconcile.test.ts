import { describe, it } from 'node:test';
import assert from 'node:assert';
import type { AccountRecord } from './types.ts';
import {
  filterReconcileCandidates,
  computeReconcileBalanceSummary,
  matchesAmountFilter,
  getNextSortConfig,
  sortReconcileCandidates,
} from './accountReconcile.ts';

function makeRec(
  id: string,
  dateSent: string,
  value: number,
  reconcileReportId: string | null = null,
  overrides: Partial<AccountRecord> = {}
): AccountRecord {
  return {
    id,
    account_id: 'acc-1',
    reconcile_report_id: reconcileReportId,
    value,
    date_sent: dateSent,
    date_cleared: null,
    entity: 'Local Shop',
    created_at: `${dateSent}T12:00:00Z`,
    updated_at: `${dateSent}T12:00:00Z`,
    ...overrides,
  };
}

describe('Account Reconcile Helpers', () => {
  it('excludes already reconciled records and filters out records older than sent cutoff date', () => {
    const records = [
      makeRec('r1', '2026-10-05', -25, 'rep-existing'),
      makeRec('r2', '2026-09-25', -30, null),
      makeRec('r3', '2026-10-01', -50, null),
      makeRec('r4', '2026-10-20', -40, null),
    ];
    assert.deepStrictEqual(
      filterReconcileCandidates(records, '2026-10-01', '').map((r) => r.id),
      ['r3', 'r4']
    );
    assert.deepStrictEqual(
      filterReconcileCandidates(records, '10/01/2026', '').map((r) => r.id),
      ['r3', 'r4']
    );
  });

  it('filters records by exact signed or unsigned transaction amount', () => {
    assert.strictEqual(matchesAmountFilter(-30, '30'), true);
    assert.strictEqual(matchesAmountFilter(30, '30'), true);
    assert.strictEqual(matchesAmountFilter(-30, '-30.00'), true);
    assert.strictEqual(matchesAmountFilter(30, '-30.00'), false);

    const records = [
      makeRec('r1', '2026-11-02', -25),
      makeRec('r2', '2026-11-03', -30),
      makeRec('r3', '2026-11-04', 30),
    ];
    assert.deepStrictEqual(
      filterReconcileCandidates(records, '', '30').map((r) => r.id),
      ['r2', 'r3']
    );
    assert.deepStrictEqual(
      filterReconcileCandidates(records, '', '-30').map((r) => r.id),
      ['r2']
    );
  });

  it('computes starting + selected records == ending balance summary', () => {
    const selected = [makeRec('r1', '2026-11-02', -30), makeRec('r2', '2026-11-05', -70)];
    const balanced = computeReconcileBalanceSummary(0, -100, selected);
    assert.strictEqual(balanced.selectedSum, -100);
    assert.strictEqual(balanced.calculatedEnding, -100);
    assert.strictEqual(balanced.difference, 0);
    assert.strictEqual(balanced.isBalanced, true);

    const unbalanced = computeReconcileBalanceSummary(0, -105, selected);
    assert.strictEqual(unbalanced.isBalanced, false);
    assert.strictEqual(unbalanced.difference, -5);
  });

  it('toggles direction for active column and resets to ascending on new column', () => {
    const initial = { key: 'date_sent' as const, direction: 'asc' as const };
    const flipped = getNextSortConfig(initial, 'date_sent');
    assert.deepStrictEqual(flipped, { key: 'date_sent', direction: 'desc' });

    const back = getNextSortConfig(flipped, 'date_sent');
    assert.deepStrictEqual(back, { key: 'date_sent', direction: 'asc' });

    const switched = getNextSortConfig(flipped, 'value');
    assert.deepStrictEqual(switched, { key: 'value', direction: 'asc' });
  });

  it('sorts reconcile candidate records across all sortable columns', () => {
    const categories = new Map<string, string>([
      ['cat-food', 'Groceries'],
      ['cat-bills', 'Bills'],
    ]);
    const records = [
      makeRec('r1', '2026-11-05', -25, null, {
        entity: 'Corner Grocery',
        date_cleared: '2026-11-07',
        category_id: 'cat-food',
        description: 'Weekly staples',
      }),
      makeRec('r2', '2026-11-01', 150, null, {
        entity: 'Employer Payroll',
        date_cleared: null,
        category_id: null,
        description: '',
      }),
      makeRec('r3', '2026-11-03', -90, null, {
        entity: 'Utility Provider',
        date_cleared: '2026-11-04',
        category_id: 'cat-bills',
        description: 'Monthly electric',
      }),
    ];

    assert.deepStrictEqual(
      sortReconcileCandidates(records, { key: 'date_sent', direction: 'desc' }, categories).map((r) => r.id),
      ['r1', 'r3', 'r2']
    );
    assert.deepStrictEqual(
      sortReconcileCandidates(records, { key: 'date_cleared', direction: 'asc' }, categories).map((r) => r.id),
      ['r3', 'r1', 'r2']
    );
    assert.deepStrictEqual(
      sortReconcileCandidates(records, { key: 'entity', direction: 'asc' }, categories).map((r) => r.id),
      ['r1', 'r2', 'r3']
    );
    assert.deepStrictEqual(
      sortReconcileCandidates(records, { key: 'category', direction: 'asc' }, categories).map((r) => r.id),
      ['r3', 'r1', 'r2']
    );
    assert.deepStrictEqual(
      sortReconcileCandidates(records, { key: 'value', direction: 'asc' }, categories).map((r) => r.id),
      ['r3', 'r1', 'r2']
    );
    assert.deepStrictEqual(
      sortReconcileCandidates(records, { key: 'description', direction: 'desc' }, categories).map((r) => r.id),
      ['r2', 'r1', 'r3']
    );
  });
});
