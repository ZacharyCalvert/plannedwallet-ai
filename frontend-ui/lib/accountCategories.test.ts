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
});
