import type { AccountRecord } from './types.ts';

export type ReconcileSortKey =
  | 'date_sent'
  | 'date_cleared'
  | 'entity'
  | 'category'
  | 'value'
  | 'description';

export type SortDirection = 'asc' | 'desc';

export interface ReconcileSortConfig {
  key: ReconcileSortKey;
  direction: SortDirection;
}

export interface ReconcileBalanceSummary {
  selectedSum: number;
  calculatedEnding: number;
  difference: number;
  isBalanced: boolean;
}

export function parseReconcileAmount(raw: string): number {
  const clean = raw.replace(/[$,\s]/g, '').trim();
  if (!clean || clean === '-' || clean === '+' || clean === '.') {
    return NaN;
  }
  if (!/^[+-]?\d*(\.\d+)?$/.test(clean)) {
    return NaN;
  }
  return parseFloat(clean);
}

export function matchesAmountFilter(recordValue: number, amountFilterStr: string): boolean {
  const trimmed = amountFilterStr.trim();
  if (!trimmed) return true;

  const parsed = parseReconcileAmount(trimmed);
  if (isNaN(parsed)) return false;

  const hasExplicitSign = trimmed.startsWith('-') || trimmed.startsWith('+');
  const recCents = Math.round(recordValue * 100);
  const targetCents = Math.round(parsed * 100);

  if (hasExplicitSign) {
    return recCents === targetCents;
  }
  return Math.abs(recCents) === Math.abs(targetCents);
}

function normalizeIsoFilterDate(raw: string): string {
  const clean = raw.trim();
  if (!clean) return '';
  if (/^\d{4}-\d{2}-\d{2}$/.test(clean)) return clean;
  const slashMatch = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/.exec(clean);
  if (slashMatch) {
    const [, m, d, y] = slashMatch;
    return `${y}-${m.padStart(2, '0')}-${d.padStart(2, '0')}`;
  }
  return '';
}

export function filterReconcileCandidates(
  records: AccountRecord[],
  cutoffDate: string,
  amountFilterStr: string
): AccountRecord[] {
  const cleanCutoff = normalizeIsoFilterDate(cutoffDate);
  return records
    .filter((rec) => {
      if (rec.reconcile_report_id) return false;
      if (cleanCutoff && rec.date_sent < cleanCutoff) return false;
      if (!matchesAmountFilter(rec.value, amountFilterStr)) return false;
      return true;
    })
    .sort((a, b) => {
      if (a.date_sent !== b.date_sent) return a.date_sent.localeCompare(b.date_sent);
      return (a.created_at || '').localeCompare(b.created_at || '');
    });
}

export function computeReconcileBalanceSummary(
  startingBalance: number,
  endingBalance: number,
  selectedRecords: AccountRecord[]
): ReconcileBalanceSummary {
  const selectedSum = selectedRecords.reduce((sum, r) => sum + r.value, 0);
  const calculatedEnding = startingBalance + selectedSum;
  const difference = endingBalance - calculatedEnding;
  const isBalanced =
    Number.isFinite(startingBalance) &&
    Number.isFinite(endingBalance) &&
    Math.abs(difference) < 0.005;

  return { selectedSum, calculatedEnding, difference, isBalanced };
}

export function getNextSortConfig(
  current: ReconcileSortConfig,
  clickedKey: ReconcileSortKey
): ReconcileSortConfig {
  if (current.key === clickedKey) {
    return { key: clickedKey, direction: current.direction === 'asc' ? 'desc' : 'asc' };
  }
  return { key: clickedKey, direction: 'asc' };
}

function compareOptionalText(aVal: string, bVal: string): number {
  const aTrim = aVal.trim();
  const bTrim = bVal.trim();
  if (!aTrim && !bTrim) return 0;
  if (!aTrim) return 1;
  if (!bTrim) return -1;
  return aTrim.localeCompare(bTrim, undefined, { sensitivity: 'base' });
}

type RecordComparator = (
  a: AccountRecord,
  b: AccountRecord,
  categoryNameById?: Map<string, string>
) => number;

const SORT_COMPARATORS: Record<ReconcileSortKey, RecordComparator> = {
  date_sent: (a, b) => a.date_sent.localeCompare(b.date_sent),
  date_cleared: (a, b) => compareOptionalText(a.date_cleared || '', b.date_cleared || ''),
  entity: (a, b) => a.entity.localeCompare(b.entity, undefined, { sensitivity: 'base' }),
  category: (a, b, cats) =>
    compareOptionalText(
      (a.category_id && cats?.get(a.category_id)) || '',
      (b.category_id && cats?.get(b.category_id)) || ''
    ),
  value: (a, b) => a.value - b.value,
  description: (a, b) => compareOptionalText(a.description || '', b.description || ''),
};

export function sortReconcileCandidates(
  records: AccountRecord[],
  sort: ReconcileSortConfig,
  categoryNameById?: Map<string, string>
): AccountRecord[] {
  const dir = sort.direction === 'asc' ? 1 : -1;
  const compare = SORT_COMPARATORS[sort.key];
  return [...records].sort((a, b) => {
    const cmp = compare(a, b, categoryNameById);
    if (cmp !== 0) return cmp * dir;
    if (a.date_sent !== b.date_sent) return a.date_sent.localeCompare(b.date_sent);
    const createdCmp = (a.created_at || '').localeCompare(b.created_at || '');
    return createdCmp !== 0 ? createdCmp : a.id.localeCompare(b.id);
  });
}
