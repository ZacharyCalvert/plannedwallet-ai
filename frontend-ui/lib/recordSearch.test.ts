import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  isSearchHotkeyEvent,
  parseNumericSearchQuery,
  searchAccountRecords,
} from './recordSearch.ts';
import type { AccountRecord, Category } from './types.ts';

const sampleCategories: Category[] = [
  { id: 'cat-fuel', name: 'Fuel & Gas', is_default: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cat-food', name: 'Groceries', is_default: true, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
];

const sampleRecords: AccountRecord[] = [
  {
    id: 'r1',
    account_id: 'a1',
    category_id: 'cat-fuel',
    value: -5.55,
    date_sent: '2026-11-01',
    date_cleared: '2026-11-02',
    entity: 'QT Station',
    description: 'Morning coffee',
    created_at: '2026-11-01T10:00:00Z',
    updated_at: '2026-11-01T10:00:00Z',
  },
  {
    id: 'r2',
    account_id: 'a1',
    category_id: null,
    value: 5.55,
    date_sent: '2026-11-03',
    date_cleared: null,
    entity: 'Rebate Credit',
    description: 'Cashback adjustment',
    created_at: '2026-11-03T10:00:00Z',
    updated_at: '2026-11-03T10:00:00Z',
  },
  {
    id: 'r3',
    account_id: 'a1',
    category_id: 'cat-food',
    value: -42.1,
    date_sent: '2026-11-04',
    date_cleared: '2026-11-05',
    entity: 'Corner Grocery',
    description: 'QT snack run',
    created_at: '2026-11-04T10:00:00Z',
    updated_at: '2026-11-04T10:00:00Z',
  },
  {
    id: 'r4',
    account_id: 'a1',
    category_id: null,
    value: -5606.12,
    date_sent: '2026-11-05',
    date_cleared: '2026-11-06',
    entity: 'Mortgage Payment',
    description: 'Monthly payment',
    created_at: '2026-11-05T10:00:00Z',
    updated_at: '2026-11-05T10:00:00Z',
  },
];

describe('Record Search Helpers', () => {
  it('parses numeric queries including currency symbols and signs', () => {
    assert.equal(parseNumericSearchQuery('5.55'), 5.55);
    assert.equal(parseNumericSearchQuery('$1,250.00'), 1250);
    assert.equal(parseNumericSearchQuery('5,606.12'), 5606.12);
    assert.equal(parseNumericSearchQuery('-5.55'), -5.55);
    assert.equal(parseNumericSearchQuery('QT'), null);
    assert.equal(parseNumericSearchQuery(''), null);
  });

  it('matches both positive and negative records by absolute value for numeric queries', () => {
    const matches = searchAccountRecords(sampleRecords, '5.55', sampleCategories);
    assert.deepEqual(
      matches.map((r) => r.id),
      ['r1', 'r2']
    );

    // Stripping commas: both "5606.12" and "5,606.12" match -5606.12 (displayed as -$5,606.12)
    assert.deepEqual(
      searchAccountRecords(sampleRecords, '5606.12', sampleCategories).map((r) => r.id),
      ['r4']
    );
    assert.deepEqual(
      searchAccountRecords(sampleRecords, '5,606.12', sampleCategories).map((r) => r.id),
      ['r4']
    );
  });

  it('matches entity, description, dates, and category names case-insensitively for text queries', () => {
    const qtMatches = searchAccountRecords(sampleRecords, 'qt', sampleCategories);
    assert.deepEqual(
      qtMatches.map((r) => r.id),
      ['r1', 'r3']
    );

    const catMatches = searchAccountRecords(sampleRecords, 'groceries', sampleCategories);
    assert.deepEqual(
      catMatches.map((r) => r.id),
      ['r3']
    );
  });

  it('detects / outside text inputs and Cmd/Ctrl+F or Cmd/Ctrl+K anywhere', () => {
    assert.equal(isSearchHotkeyEvent({ key: '/', metaKey: false, ctrlKey: false, altKey: false }, 'BODY'), true);
    assert.equal(isSearchHotkeyEvent({ key: '/', metaKey: false, ctrlKey: false, altKey: false }, 'INPUT'), false);
    assert.equal(isSearchHotkeyEvent({ key: '/', metaKey: false, ctrlKey: false, altKey: false }, 'TEXTAREA'), false);
    assert.equal(isSearchHotkeyEvent({ key: '/', metaKey: false, ctrlKey: false, altKey: false }, 'DIV', true), false);
    assert.equal(isSearchHotkeyEvent({ key: 'k', metaKey: true, ctrlKey: false, altKey: false }, 'INPUT'), true);
    assert.equal(isSearchHotkeyEvent({ key: 'K', metaKey: false, ctrlKey: true, altKey: false }, 'INPUT'), true);
    assert.equal(isSearchHotkeyEvent({ key: 'f', metaKey: true, ctrlKey: false, altKey: false }, 'BODY'), true);
    assert.equal(isSearchHotkeyEvent({ key: 'F', metaKey: false, ctrlKey: true, altKey: false }, 'INPUT'), true);
  });
});
