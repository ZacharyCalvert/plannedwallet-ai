import type { AccountRecord, Category } from './types.ts';
import type { BalanceRange } from './accountTrend.ts';

export interface PieChartSlice {
  categoryId: string | null;
  categoryName: string;
  amount: number;
  percentage: number;
  recordCount: number;
  color: string;
}

export interface CategoryPieSummary {
  totalExpenses: number;
  slices: PieChartSlice[];
}

const SLICE_PALETTE = [
  '#2563eb',
  '#059669',
  '#d97706',
  '#7c3aed',
  '#e11d48',
  '#0891b2',
  '#65a30d',
  '#db2777',
  '#ea580c',
  '#4f46e5',
];
const UNCATEGORIZED_COLOR = '#94a3b8';

export function normalizeCategoryKey(name: string): string {
  return name.trim().toLowerCase();
}

export function findCategoryByNameCaseInsensitive(
  categories: Category[],
  name: string
): Category | undefined {
  const target = normalizeCategoryKey(name);
  if (!target) return undefined;
  return categories.find((c) => normalizeCategoryKey(c.name) === target);
}

export function buildCategoryLookupMap(categories: Category[]): Map<string, Category> {
  const map = new Map<string, Category>();
  for (const cat of categories) {
    const key = normalizeCategoryKey(cat.name);
    if (!map.has(key)) {
      map.set(key, cat);
    }
  }
  return map;
}

export function findLastCategoryForRecipient(
  records: AccountRecord[],
  recipient: string
): string | null {
  const normTarget = recipient.trim().toLowerCase();
  if (!normTarget) return null;

  const matching = records.filter(
    (r) => r.entity.trim().toLowerCase() === normTarget && Boolean(r.category_id)
  );
  if (matching.length === 0) return null;

  matching.sort((a, b) => {
    if (a.date_sent !== b.date_sent) return b.date_sent.localeCompare(a.date_sent);
    const aCreated = a.created_at || '';
    const bCreated = b.created_at || '';
    if (aCreated !== bCreated) return bCreated.localeCompare(aCreated);
    return b.id.localeCompare(a.id);
  });
  return matching[0].category_id || null;
}

export function getRecordsForCategory(
  records: AccountRecord[],
  categoryId: string
): AccountRecord[] {
  return records
    .filter((r) => r.category_id === categoryId)
    .sort((a, b) => b.date_sent.localeCompare(a.date_sent));
}

function getLocalTodayIso(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function subtractDaysIso(isoDate: string, days: number): string {
  const [y, m, d] = isoDate.split('-').map((n) => parseInt(n, 10));
  const dt = new Date(Date.UTC(y, m - 1, d));
  dt.setUTCDate(dt.getUTCDate() - days);
  return dt.toISOString().slice(0, 10);
}

function rangeToDays(range: Exclude<BalanceRange, 'all'>): number {
  switch (range) {
    case '30d':
      return 30;
    case '60d':
      return 60;
    case '90d':
      return 90;
    case '1y':
      return 365;
  }
}

export function getRangeStartDate(
  records: AccountRecord[],
  range: BalanceRange,
  referenceDateStr?: string
): string | null {
  if (range === 'all') return null;
  const todayStr = referenceDateStr || getLocalTodayIso();
  let latest = todayStr;
  for (const r of records) {
    if (r.date_sent > latest) latest = r.date_sent;
  }
  return subtractDaysIso(latest, rangeToDays(range));
}

export function computeCategoryPieData(
  records: AccountRecord[],
  categories: Category[],
  range: BalanceRange,
  includeUncategorized: boolean,
  referenceDateStr?: string
): CategoryPieSummary {
  const cutoff = getRangeStartDate(records, range, referenceDateStr);
  const nameById = new Map<string, string>();
  for (const c of categories) {
    nameById.set(c.id, c.name);
  }

  const groups = new Map<string, { id: string | null; name: string; amount: number; count: number }>();
  let totalExpenses = 0;

  for (const r of records) {
    if (r.value >= 0) continue;
    if (cutoff && r.date_sent < cutoff) continue;

    const resolvedName = r.category_id ? nameById.get(r.category_id) : undefined;
    const isUncategorized = !r.category_id || !resolvedName;
    if (isUncategorized && !includeUncategorized) continue;

    const amt = Math.abs(r.value);
    totalExpenses += amt;
    const key = isUncategorized ? '__uncategorized__' : (r.category_id as string);
    const group = groups.get(key) || {
      id: isUncategorized ? null : (r.category_id as string),
      name: isUncategorized ? 'Uncategorized' : (resolvedName as string),
      amount: 0,
      count: 0,
    };
    group.amount += amt;
    group.count += 1;
    groups.set(key, group);
  }

  const sorted = Array.from(groups.values()).sort((a, b) => {
    if (b.amount !== a.amount) return b.amount - a.amount;
    return a.name.localeCompare(b.name);
  });

  let colorIdx = 0;
  const slices: PieChartSlice[] = sorted.map((g) => {
    const color = g.id === null ? UNCATEGORIZED_COLOR : SLICE_PALETTE[colorIdx++ % SLICE_PALETTE.length];
    return {
      categoryId: g.id,
      categoryName: g.name,
      amount: g.amount,
      percentage: totalExpenses > 0 ? (g.amount / totalExpenses) * 100 : 0,
      recordCount: g.count,
      color,
    };
  });

  return { totalExpenses, slices };
}

export async function resolveAndCreateMissingCategories(
  rawCategoryNames: string[],
  existingCategories: Category[],
  createFn: (name: string) => Promise<Category>
): Promise<{ idByLowerName: Map<string, string>; created: Category[] }> {
  const lookup = buildCategoryLookupMap(existingCategories);
  const idByLowerName = new Map<string, string>();
  for (const [k, cat] of lookup.entries()) {
    idByLowerName.set(k, cat.id);
  }

  const created: Category[] = [];
  for (const raw of rawCategoryNames) {
    const trimmed = raw.trim();
    const key = normalizeCategoryKey(trimmed);
    if (!key || idByLowerName.has(key)) continue;

    const newCat = await createFn(trimmed);
    idByLowerName.set(key, newCat.id);
    created.push(newCat);
  }
  return { idByLowerName, created };
}

export async function transitionAndDeleteCategory(
  recordsToTransition: Pick<AccountRecord, 'id' | 'account_id'>[],
  deletingCategoryId: string,
  transitionToCategoryId: string | null,
  patchFn: (accountId: string, recordId: string, targetCategoryId: string) => Promise<void>,
  deleteFn: (categoryId: string) => Promise<void>
): Promise<void> {
  if (transitionToCategoryId) {
    for (const rec of recordsToTransition) {
      await patchFn(rec.account_id, rec.id, transitionToCategoryId);
    }
  }
  await deleteFn(deletingCategoryId);
}
