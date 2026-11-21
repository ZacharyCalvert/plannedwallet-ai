import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildAccountCsvFilename,
  csvEscapeCell,
  formatAssetStoresCsv,
  formatCheckingRecordsCsv,
  formatSnapshotsCsv,
} from './exportFormatters.ts';
import type { Account, AccountRecord, AccountSnapshot, AssetStore, AssetStoreRecord } from './types.ts';

describe('Data Export CSV Helpers', () => {
  it('escapes RFC 4180 CSV cells properly', () => {
    assert.strictEqual(csvEscapeCell('plain'), 'plain');
    assert.strictEqual(csvEscapeCell('a,b'), '"a,b"');
    assert.strictEqual(csvEscapeCell('say "hi"'), '"say ""hi"""');
    assert.strictEqual(csvEscapeCell(null), '');
  });

  it('deduplicates account CSV filenames cleanly', () => {
    const used = new Set<string>();
    const a1: Account = {
      id: '11111111-1111-1111-1111-111111111111',
      admin_id: 'u1',
      name: 'Everyday Checking',
      type: 'checking',
      description: 'Main',
      created_at: '',
      updated_at: '',
    };
    const a2: Account = { ...a1, id: '22222222-2222-2222-2222-222222222222' };
    assert.strictEqual(buildAccountCsvFilename(a1, used), 'Everyday_Checking.csv');
    assert.strictEqual(buildAccountCsvFilename(a2, used), 'Everyday_Checking_222222.csv');
  });

  it('formats checking records CSV with resolved category names', () => {
    const records: AccountRecord[] = [
      {
        id: 'r2',
        account_id: 'a1',
        category_id: 'c1',
        value: -42.5,
        date_sent: '2026-11-10',
        date_cleared: '2026-11-11',
        entity: 'Corner Grocery',
        description: 'Weekly run, snacks',
        created_at: '',
        updated_at: '',
      },
      {
        id: 'r1',
        account_id: 'a1',
        value: 1500,
        date_sent: '2026-11-01',
        date_cleared: null,
        entity: 'Employer Payroll',
        description: '',
        created_at: '',
        updated_at: '',
      },
    ];
    const csv = formatCheckingRecordsCsv(records, new Map([['c1', 'Groceries']]));
    const lines = csv.trim().split('\n');
    assert.strictEqual(lines[0], 'date_sent,date_cleared,value,entity,description,category,reconcile_report_id');
    assert.strictEqual(lines[1], '2026-11-01,,1500,Employer Payroll,,,');
    assert.strictEqual(lines[2], '2026-11-10,2026-11-11,-42.5,Corner Grocery,"Weekly run, snacks",Groceries,');
  });

  it('formats snapshot valuations CSV chronologically', () => {
    const snaps: AccountSnapshot[] = [
      { id: 's2', account_id: 'a2', value: 12000, snapshot_date: '2026-11-15', created_at: 'c2', updated_at: 'u2' },
      { id: 's1', account_id: 'a2', value: 10000, snapshot_date: '2026-11-01', created_at: 'c1', updated_at: 'u1' },
    ];
    const csv = formatSnapshotsCsv(snaps);
    const lines = csv.trim().split('\n');
    assert.strictEqual(lines[0], 'snapshot_date,value,created_at,updated_at');
    assert.strictEqual(lines[1], '2026-11-01,10000,c1,u1');
    assert.strictEqual(lines[2], '2026-11-15,12000,c2,u2');
  });

  it('formats asset stores and store quantity records CSV', () => {
    const stores: AssetStore[] = [
      {
        id: 'st1',
        account_id: 'a3',
        asset_id: 'as1',
        label: 'Home Safe',
        description: 'Physical bullion',
        asset_label: 'Gold',
        latest_quantity: 2,
        latest_value: 2500,
        total_value: 5000,
        created_at: '',
        updated_at: '',
      },
    ];
    const records: Record<string, AssetStoreRecord[]> = {
      st1: [
        { id: 'sr1', asset_store_id: 'st1', quantity: 2, record_date: '2026-11-05', created_at: '', updated_at: '' },
      ],
    };
    const csv = formatAssetStoresCsv(stores, records);
    const lines = csv.trim().split('\n');
    assert.strictEqual(lines[0], 'store_label,store_description,asset_label,record_date,quantity,latest_unit_value,latest_total_value');
    assert.strictEqual(lines[1], 'Home Safe,Physical bullion,Gold,2026-11-05,2,2500,5000');
  });
});
