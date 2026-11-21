import type { AccountRecord, Category } from './types.ts';

export function parseNumericSearchQuery(raw: string): number | null {
  const clean = raw.replace(/[$,\s]/g, '').trim();
  if (!clean) return null;
  if (!/^[+-]?(\d+(\.\d*)?|\.\d+)$/.test(clean)) {
    return null;
  }
  const parsed = parseFloat(clean);
  return Number.isFinite(parsed) ? parsed : null;
}

export function searchAccountRecords(
  records: AccountRecord[],
  query: string,
  categories: Category[] = []
): AccountRecord[] {
  const trimmed = query.trim();
  if (!trimmed) {
    return records;
  }

  const numericTarget = parseNumericSearchQuery(trimmed);
  if (numericTarget !== null) {
    const targetAbsCents = Math.abs(Math.round(numericTarget * 100));
    return records.filter((rec) => Math.abs(Math.round(rec.value * 100)) === targetAbsCents);
  }

  const lowerQuery = trimmed.toLowerCase();
  const categoryNameById = new Map(categories.map((cat) => [cat.id, cat.name.toLowerCase()]));

  return records.filter((rec) => {
    if (rec.entity.toLowerCase().includes(lowerQuery)) return true;
    if ((rec.description || '').toLowerCase().includes(lowerQuery)) return true;
    if (rec.date_sent.toLowerCase().includes(lowerQuery)) return true;
    if ((rec.date_cleared || '').toLowerCase().includes(lowerQuery)) return true;
    if (rec.category_id) {
      const catName = categoryNameById.get(rec.category_id);
      if (catName && catName.includes(lowerQuery)) return true;
    }
    return false;
  });
}

export function isSearchHotkeyEvent(
  e: { key: string; metaKey: boolean; ctrlKey: boolean; altKey: boolean },
  targetTagName?: string,
  isContentEditable?: boolean
): boolean {
  const lowerKey = e.key.toLowerCase();
  if ((e.metaKey || e.ctrlKey) && !e.altKey && (lowerKey === 'k' || lowerKey === 'f')) {
    return true;
  }
  if (e.metaKey || e.ctrlKey || e.altKey) {
    return false;
  }
  if (e.key !== '/') {
    return false;
  }
  const tag = (targetTagName || '').toUpperCase();
  if (isContentEditable || tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
    return false;
  }
  return true;
}
