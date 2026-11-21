import type {
  Account,
  AccountRecord,
  AccountSnapshot,
  AssetStore,
  AssetStoreRecord,
  Category,
} from './types.ts';

/**
 * Escapes a single CSV cell according to RFC 4180.
 */
export function csvEscapeCell(value: string | number | null | undefined): string {
  if (value === null || value === undefined) {
    return '';
  }
  const str = String(value);
  if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

/**
 * Builds a CSV string from header names and row arrays.
 */
export function buildCsv(
  headers: string[],
  rows: Array<Array<string | number | null | undefined>>
): string {
  const lines = [headers.map(csvEscapeCell).join(',')];
  for (const row of rows) {
    lines.push(row.map(csvEscapeCell).join(','));
  }
  return `${lines.join('\n')}\n`;
}

/**
 * Sanitizes an account name into a safe filename stem.
 */
export function sanitizeExportFilename(name: string): string {
  const cleaned = name
    .trim()
    .replace(/[^a-zA-Z0-9._-]+/g, '_')
    .replace(/^_+|_+$/g, '');
  return cleaned || 'account';
}

/**
 * Generates a unique CSV filename for an account within a multi-account archive.
 */
export function buildAccountCsvFilename(account: Account, usedNames?: Set<string>): string {
  const base = sanitizeExportFilename(account.name);
  let candidate = `${base}.csv`;
  if (usedNames && usedNames.has(candidate.toLowerCase())) {
    candidate = `${base}_${account.id.slice(0, 6)}.csv`;
  }
  usedNames?.add(candidate.toLowerCase());
  return candidate;
}

/**
 * Formats checking, savings, or credit card records into CSV.
 */
export function formatCheckingRecordsCsv(
  records: AccountRecord[],
  categoryMap: Map<string, string>
): string {
  const sorted = [...records].sort((a, b) => a.date_sent.localeCompare(b.date_sent));
  const headers = [
    'date_sent',
    'date_cleared',
    'value',
    'entity',
    'description',
    'category',
    'reconcile_report_id',
  ];
  const rows = sorted.map((r) => [
    r.date_sent,
    r.date_cleared ?? '',
    r.value,
    r.entity,
    r.description ?? '',
    r.category_id ? (categoryMap.get(r.category_id) ?? '') : '',
    r.reconcile_report_id ?? '',
  ]);
  return buildCsv(headers, rows);
}

/**
 * Formats snapshot account valuations into CSV.
 */
export function formatSnapshotsCsv(snapshots: AccountSnapshot[]): string {
  const sorted = [...snapshots].sort((a, b) => a.snapshot_date.localeCompare(b.snapshot_date));
  const headers = ['snapshot_date', 'value', 'created_at', 'updated_at'];
  const rows = sorted.map((s) => [s.snapshot_date, s.value, s.created_at, s.updated_at]);
  return buildCsv(headers, rows);
}

/**
 * Formats asset account storage holdings and quantity records into CSV.
 */
export function formatAssetStoresCsv(
  stores: AssetStore[],
  recordsByStore: Record<string, AssetStoreRecord[]>
): string {
  const headers = [
    'store_label',
    'store_description',
    'asset_label',
    'record_date',
    'quantity',
    'latest_unit_value',
    'latest_total_value',
  ];
  const rows: Array<Array<string | number | null | undefined>> = [];
  for (const store of stores) {
    const storeRecords = [...(recordsByStore[store.id] ?? [])].sort((a, b) =>
      a.record_date.localeCompare(b.record_date)
    );
    if (storeRecords.length === 0) {
      rows.push([
        store.label,
        store.description ?? '',
        store.asset_label,
        '',
        '',
        store.latest_value,
        store.total_value,
      ]);
      continue;
    }
    for (const rec of storeRecords) {
      rows.push([
        store.label,
        store.description ?? '',
        store.asset_label,
        rec.record_date,
        rec.quantity,
        store.latest_value,
        store.total_value,
      ]);
    }
  }
  return buildCsv(headers, rows);
}

export function buildCategoryNameMap(categories: Category[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const cat of categories) {
    map.set(cat.id, cat.name);
  }
  return map;
}
