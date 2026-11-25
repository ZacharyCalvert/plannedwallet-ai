import type { Account, AccountSnapshot, AssetValue } from './types.ts';

export type DailyUpdateKind = 'snapshot' | 'asset';

export interface DailyUpdateItem {
  kind: DailyUpdateKind;
  id: string;
  name: string;
  subtitle: string;
  lastValue: number | null;
  lastDate: string | null;
}

export function dailyUpdateItemKey(item: Pick<DailyUpdateItem, 'kind' | 'id'>): string {
  return `${item.kind}:${item.id}`;
}

function compareDefaultDailyUpdateItems(a: DailyUpdateItem, b: DailyUpdateItem): number {
  const kindOrderA = a.kind === 'snapshot' ? 0 : 1;
  const kindOrderB = b.kind === 'snapshot' ? 0 : 1;
  if (kindOrderA !== kindOrderB) {
    return kindOrderA - kindOrderB;
  }
  const nameCmp = a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
  if (nameCmp !== 0) {
    return nameCmp;
  }
  if (a.name !== b.name) {
    return a.name < b.name ? -1 : 1;
  }
  if (a.id !== b.id) {
    return a.id < b.id ? -1 : 1;
  }
  return 0;
}

export function sortDailyUpdateItems(
  items: DailyUpdateItem[],
  dailyUpdateOrder: string[] = []
): DailyUpdateItem[] {
  const rank = new Map<string, number>();
  dailyUpdateOrder.forEach((key, idx) => {
    if (!rank.has(key)) {
      rank.set(key, idx);
    }
  });
  return [...items].sort((a, b) => {
    const rankA = rank.get(dailyUpdateItemKey(a));
    const rankB = rank.get(dailyUpdateItemKey(b));
    if (rankA !== undefined && rankB !== undefined) {
      return rankA - rankB;
    }
    if (rankA !== undefined) return -1;
    if (rankB !== undefined) return 1;
    return compareDefaultDailyUpdateItems(a, b);
  });
}

function getLatestSnapshot(snapshots: AccountSnapshot[]): AccountSnapshot | null {
  if (snapshots.length === 0) return null;
  return [...snapshots].sort((a, b) => {
    const dateA = a.snapshot_date.slice(0, 10);
    const dateB = b.snapshot_date.slice(0, 10);
    if (dateA !== dateB) return dateB.localeCompare(dateA);
    return b.created_at.localeCompare(a.created_at);
  })[0];
}

function getLatestAssetValue(values: AssetValue[]): AssetValue | null {
  if (values.length === 0) return null;
  return [...values].sort((a, b) => {
    const dateA = a.value_date.slice(0, 10);
    const dateB = b.value_date.slice(0, 10);
    if (dateA !== dateB) return dateB.localeCompare(dateA);
    return b.created_at.localeCompare(a.created_at);
  })[0];
}

export function hasSnapshotForDate(snapshots: AccountSnapshot[], isoDate: string): boolean {
  return snapshots.some((s) => s.snapshot_date.slice(0, 10) === isoDate);
}

export function hasAssetValueForDate(values: AssetValue[], isoDate: string): boolean {
  return values.some((v) => v.value_date.slice(0, 10) === isoDate);
}

export function buildPendingSnapshotItem(
  account: Account,
  snapshots: AccountSnapshot[],
  todayIso: string
): DailyUpdateItem | null {
  if (hasSnapshotForDate(snapshots, todayIso)) {
    return null;
  }
  const latest = getLatestSnapshot(snapshots);
  return {
    kind: 'snapshot',
    id: account.id,
    name: account.name,
    subtitle: account.description || 'Snapshot account',
    lastValue: latest ? latest.value : null,
    lastDate: latest ? latest.snapshot_date.slice(0, 10) : null,
  };
}

export function buildPendingAssetItem(
  assetId: string,
  assetLabel: string,
  accountNames: string[],
  values: AssetValue[],
  todayIso: string
): DailyUpdateItem | null {
  if (hasAssetValueForDate(values, todayIso)) {
    return null;
  }
  const latest = getLatestAssetValue(values);
  const subtitle =
    accountNames.length === 1
      ? `Used in ${accountNames[0]}`
      : `Used in ${accountNames.length} asset accounts`;
  return {
    kind: 'asset',
    id: assetId,
    name: assetLabel,
    subtitle,
    lastValue: latest ? latest.value : null,
    lastDate: latest ? latest.value_date.slice(0, 10) : null,
  };
}
