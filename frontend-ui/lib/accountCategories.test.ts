import { describe, it } from 'node:test';
import assert from 'node:assert';
import type { AccountRecord, Category } from './types.ts';
import {
  findCategoryByNameCaseInsensitive,
  findLastCategoryForRecipient,
  computeCategoryPieData,
  resolveAndCreateMissingCategories,
  transitionAndDeleteCategory,
} from './accountCategories.ts';

function makeCat(id: string, name: string, isDefault = false): Category {
  return {
    id,
    user_id: isDefault ? null : 'user-1',
    name,
    is_default: isDefault,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function makeRec(
  id: string,
  dateSent: string,
  value: number,
  entity: string,
  categoryId: string | null = null
): AccountRecord {
  return {
    id,
    account_id: 'acc-1',
    category_id: categoryId,
    value,
    date_sent: dateSent,
    date_cleared: null,
    entity,
    created_at: `${dateSent}T12:00:00Z`,
    updated_at: `${dateSent}T12:00:00Z`,
  };
}

describe('Account Categories Helpers', () => {
  it('matches categories case-insensitively', () => {
    const cats = [makeCat('1', 'Groceries', true), makeCat('2', 'Dining Out')];
    assert.strictEqual(findCategoryByNameCaseInsensitive(cats, 'groceries')?.id, '1');
    assert.strictEqual(findCategoryByNameCaseInsensitive(cats, '  DINING OUT ')?.id, '2');
    assert.strictEqual(findCategoryByNameCaseInsensitive(cats, 'Utilities'), undefined);
  });

  it('defaults to the most recent category used for a recipient (case-insensitive)', () => {
    const records = [
      makeRec('r1', '2026-10-01', -20, 'Corner Grocery', 'cat-old'),
      makeRec('r2', '2026-11-05', -45, 'corner grocery', 'cat-new'),
      makeRec('r3', '2026-11-10', -15, 'Corner Grocery', null),
    ];
    assert.strictEqual(findLastCategoryForRecipient(records, 'CORNER GROCERY'), 'cat-new');
    assert.strictEqual(findLastCategoryForRecipient(records, 'Unknown Vendor'), null);
  });

  it('computes pie chart slices across timeframes and uncategorized toggle', () => {
    const cats = [makeCat('g', 'Groceries', true), makeCat('u', 'Utilities', true)];
    const records = [
      makeRec('1', '2025-05-01', -500, 'Old Utility', 'u'),
      makeRec('2', '2026-10-01', -100, 'Corner Grocery', 'g'),
      makeRec('3', '2026-11-10', -200, 'Corner Grocery', 'g'),
      makeRec('4', '2026-11-12', -100, 'Misc Shop', null),
      makeRec('5', '2026-11-14', 1000, 'Payroll', null),
    ];

    const allWithUncat = computeCategoryPieData(records, cats, 'all', true, '2026-11-16');
    assert.strictEqual(allWithUncat.totalExpenses, 900);
    assert.strictEqual(allWithUncat.slices.length, 3);
    assert.strictEqual(allWithUncat.slices[0].categoryName, 'Utilities');

    const d30WithUncat = computeCategoryPieData(records, cats, '30d', true, '2026-11-16');
    assert.strictEqual(d30WithUncat.totalExpenses, 300);
    assert.strictEqual(d30WithUncat.slices.length, 2);

    const d30NoUncat = computeCategoryPieData(records, cats, '30d', false, '2026-11-16');
    assert.strictEqual(d30NoUncat.totalExpenses, 200);
    assert.strictEqual(d30NoUncat.slices.length, 1);
    assert.strictEqual(d30NoUncat.slices[0].percentage, 100);
  });

  it('resolves existing categories case-insensitively and creates missing ones once during bulk import', async () => {
    const existing = [makeCat('1', 'Groceries', true)];
    const createdCalls: string[] = [];
    const { idByLowerName, created } = await resolveAndCreateMissingCategories(
      ['groceries', 'GROCERIES', 'Pet Care', 'pet care', '  '],
      existing,
      async (name) => {
        createdCalls.push(name);
        return makeCat(`new-${createdCalls.length}`, name);
      }
    );

    assert.deepStrictEqual(createdCalls, ['Pet Care']);
    assert.strictEqual(created.length, 1);
    assert.strictEqual(idByLowerName.get('groceries'), '1');
    assert.strictEqual(idByLowerName.get('pet care'), 'new-1');
  });

  it('transitions associated records from frontend before deleting category', async () => {
    const callOrder: string[] = [];
    const recs = [
      { id: 'rec-1', account_id: 'acc-1' },
      { id: 'rec-2', account_id: 'acc-2' },
    ];

    await transitionAndDeleteCategory(
      recs,
      'cat-old',
      'cat-target',
      async (accId, recId, targetId) => {
        callOrder.push(`patch:${accId}:${recId}:${targetId}`);
      },
      async (catId) => {
        callOrder.push(`delete:${catId}`);
      }
    );

    assert.deepStrictEqual(callOrder, [
      'patch:acc-1:rec-1:cat-target',
      'patch:acc-2:rec-2:cat-target',
      'delete:cat-old',
    ]);

    const uncatOrder: string[] = [];
    await transitionAndDeleteCategory(
      recs,
      'cat-old',
      null,
      async () => {
        uncatOrder.push('patch-should-not-run');
      },
      async (catId) => {
        uncatOrder.push(`delete:${catId}`);
      }
    );
    assert.deepStrictEqual(uncatOrder, ['delete:cat-old']);
  });

  it('summarizes uncategorized expense vendors and bulk assigns categories case-insensitively', async () => {
    const {
      summarizeUncategorizedVendors,
      getUncategorizedExpensesForVendor,
      bulkAssignCategoryToRecords,
    } = await import('./accountCategories.ts');

    const recs = [
      { id: '1', account_id: 'acc-1', value: -12.5, date_sent: '2026-11-01', date_cleared: null, category_id: null, entity: 'Corner Grocery', description: 'Milk', created_at: '', updated_at: '' },
      { id: '2', account_id: 'acc-1', value: -20.0, date_sent: '2026-11-03', date_cleared: null, category_id: null, entity: 'corner grocery', description: 'Bread', created_at: '', updated_at: '' },
      { id: '3', account_id: 'acc-1', value: 50.0, date_sent: '2026-11-04', date_cleared: null, category_id: null, entity: 'Corner Grocery', description: 'Refund', created_at: '', updated_at: '' },
      { id: '4', account_id: 'acc-1', value: -8.0, date_sent: '2026-11-02', date_cleared: null, category_id: 'cat-1', entity: 'Corner Grocery', description: 'Already categorized', created_at: '', updated_at: '' },
      { id: '5', account_id: 'acc-1', value: -5.5, date_sent: '2026-11-05', date_cleared: null, category_id: null, entity: 'Coffee Shop', description: 'Latte', created_at: '', updated_at: '' },
    ];

    const summary = summarizeUncategorizedVendors(recs);
    assert.strictEqual(summary.length, 2);
    assert.strictEqual(summary[0].displayName, 'Corner Grocery');
    assert.strictEqual(summary[0].count, 2);
    assert.strictEqual(summary[0].totalAmount, 32.5);
    assert.strictEqual(summary[1].displayName, 'Coffee Shop');
    assert.strictEqual(summary[1].count, 1);

    const matching = getUncategorizedExpensesForVendor(recs, 'CORNER GROCERY');
    assert.deepStrictEqual(
      matching.map((r) => r.id),
      ['2', '1']
    );

    const progressUpdates: string[] = [];
    const updated = await bulkAssignCategoryToRecords(
      matching,
      'cat-groceries',
      async (accId, recId, catId) => {
        const orig = recs.find((r) => r.id === recId)!;
        return { ...orig, account_id: accId, category_id: catId };
      },
      (done, total) => progressUpdates.push(`${done}/${total}`)
    );
    assert.strictEqual(updated.length, 2);
    assert.strictEqual(updated[0].category_id, 'cat-groceries');
    assert.deepStrictEqual(progressUpdates, ['1/2', '2/2']);
  });

  it('filters disabled categories from pie summary and recomputes percentages', async () => {
    const {
      computeCategoryPieData,
      filterAndRecomputePieData,
      getPieSliceKey,
    } = await import('./accountCategories.ts');

    const cats = [
      { id: 'cat-debt', name: 'Debt', is_default: false, created_at: '', updated_at: '' },
      { id: 'cat-food', name: 'Groceries', is_default: true, created_at: '', updated_at: '' },
      { id: 'cat-util', name: 'Utilities', is_default: true, created_at: '', updated_at: '' },
    ];
    const recs = [
      { id: '1', account_id: 'a1', value: -100, date_sent: '2026-11-01', date_cleared: null, category_id: 'cat-debt', entity: 'Loan', description: '', created_at: '', updated_at: '' },
      { id: '2', account_id: 'a1', value: -60, date_sent: '2026-11-02', date_cleared: null, category_id: 'cat-food', entity: 'Market', description: '', created_at: '', updated_at: '' },
      { id: '3', account_id: 'a1', value: -40, date_sent: '2026-11-03', date_cleared: null, category_id: 'cat-util', entity: 'Power', description: '', created_at: '', updated_at: '' },
    ];

    const full = computeCategoryPieData(recs, cats, 'all', true, '2026-11-10');
    assert.strictEqual(full.totalExpenses, 200);
    assert.strictEqual(full.slices.length, 3);

    const debtSlice = full.slices.find((s) => s.categoryId === 'cat-debt')!;
    const filtered = filterAndRecomputePieData(full, new Set([getPieSliceKey(debtSlice)]));
    assert.strictEqual(filtered.totalExpenses, 100);
    assert.strictEqual(filtered.slices.length, 2);
    assert.strictEqual(filtered.slices[0].categoryName, 'Groceries');
    assert.strictEqual(filtered.slices[0].percentage, 60);
    assert.strictEqual(filtered.slices[1].categoryName, 'Utilities');
    assert.strictEqual(filtered.slices[1].percentage, 40);
  });
});
